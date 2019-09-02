package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/vchimishuk/chub/cue"
)

var (
	outputDir = flag.String("o", ".", "Output directory")
)

type AudioFile struct {
	Path string
	Size int64
}

type CueFile struct {
	Path  string
	Sheet *cue.Sheet
}

type Track struct {
	Number      int
	Performer   string
	SongWriter  string
	Title       string
	Genre       string
	Date        string
	FirstSample int
	LastSample  int
}

type Input struct {
	Audio  AudioFile
	Cue    CueFile
	Tracks []Track

	Performer string
	Title     string
	Genre     string
	Date      string
}

func scanDir(path string) (ins []Input) {
	var f *os.File
	var fis []os.FileInfo
	var err error

	if f, err = os.Open(path); err == nil {
		if fis, err = f.Readdir(0); err == nil {
			for _, fi := range fis {
				fiPath := path + "/" + fi.Name()
				if fi.IsDir() {
					ins = append(ins, scanDir(fiPath)...)
				} else if strings.HasSuffix(fi.Name(), ".flac") {
					ins = append(ins, parseInput(fiPath, fi))
				}
			}
		}
	}

	if err != nil {
		log.Fatalf("%s: %s", path, err)
	}
	return
}

func timeToSamples(sampleRate int, t *cue.Time) int {
	return (t.Min*60+t.Sec)*sampleRate + sampleRate/75*t.Frames
}

func parseInput(path string, fi os.FileInfo) (in Input) {
	in.Audio = AudioFile{Path: path, Size: fi.Size()}
	in.Cue.Path = strings.TrimSuffix(path, ".flac") + ".cue"

	var err error
	var out strings.Builder
	cmd := exec.Command("metaflac", "--list", path)
	cmd.Stdout = &out
	if err = cmd.Run(); err == nil {
		var totalSamples, sampleRate int
		lines := strings.Split(out.String(), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "  sample_rate: ") {
				words := strings.Fields(line)
				sampleRate, err = strconv.Atoi(words[len(words)-2])
			} else if strings.HasPrefix(line, "  total samples: ") {
				words := strings.Fields(line)
				totalSamples, err = strconv.Atoi(words[len(words)-1])
			} else if totalSamples != 0 && sampleRate != 0 {
				break
			}
			if err != nil {
				log.Fatalf("%s: %s", line, err)
			}
		}

		var cueReader io.ReadCloser

		// try external cue sheet
		if cueReader, err = os.Open(in.Cue.Path); err != nil {
			// FIXME fall back to internal one
			log.Fatalf("%s: internal sheet not supported yet", path)
		}

		if in.Cue.Sheet, err = cue.Parse(cueReader); err == nil {
			files := in.Cue.Sheet.Files
			if len(files) != 1 {
				err = fmt.Errorf("unsupported number of files: %d", len(files))
			} else if files[0].Type != cue.FileTypeWave {
				err = fmt.Errorf("unsupported file type %d", files[0].Type)
			} else {
				var date, genre string
				for _, c := range in.Cue.Sheet.Comments {
					if strings.HasPrefix(c, "DATE") {
						words := strings.Fields(c)
						date = words[len(words)-1]
					} else if strings.HasPrefix(c, "GENRE") {
						words := strings.SplitAfterN(c, " ", 2)
						genre = words[1]
					}
				}
				in.Performer = in.Cue.Sheet.Performer
				in.Title = in.Cue.Sheet.Title
				in.Genre = genre
				in.Date = date
				in.Tracks = make([]Track, len(files[0].Tracks))
				for i, ft := range files[0].Tracks {
					t := &in.Tracks[i]
					*t = Track{
						Number:      ft.Number,
						Title:       ft.Title,
						Performer:   ft.Performer,
						SongWriter:  ft.Songwriter,
						Genre:       genre,
						Date:        date,
						FirstSample: timeToSamples(sampleRate, ft.Indexes[0].Time),
					}
					if t.Number == 0 {
						t.Number = i + 1
					}
					if i > 0 {
						in.Tracks[i-1].LastSample = t.FirstSample - 1
					}
				}
				in.Tracks[len(in.Tracks)-1].LastSample = totalSamples - 1
			}
		}
	}

	if err != nil {
		log.Fatalf("%s: %s", path, err)
	}
	return
}

func (in *Input) OutputPath() (path string) {
	performer := in.Performer
	if performer == "" {
		// FIXME go through tracks and see if there is one or several
		// that can be used to construct proper performer string here
		// might end up being "Various Artists" too?
		performer = "Unknown Artist"
	}
	// FIXME remove characters that can't be used in a dir name
	path = performer + "/"
	if in.Date != "" {
		path += in.Date + " - "
	}
	if in.Title != "" {
		path += in.Title
	} else {
		path += "Unknown Album" // FIXME this name sucks
	}
	// FIXME make sure the final path doesn't exist
	return
}

func (t *Track) OutputPath(ext string) (path string) {
	path = fmt.Sprintf("%02d", t.Number)
	if t.Title != "" {
		path += " - " + t.Title
	}
	path += ext
	return
}

func (in *Input) Dump() {
	fmt.Printf("audio: %s\n", in.Audio.Path)
	fmt.Printf("size: %d bytes\n", in.Audio.Size)
	fmt.Printf("cue: %s\n", in.Cue.Path)
	out := in.OutputPath()
	for _, t := range in.Tracks {
		fmt.Printf("%s/%s\n\tfirst=%d last=%d\n", out, t.OutputPath(".flac"), t.FirstSample, t.LastSample)
	}
}

func main() {
	flag.Parse()

	var inputs []Input
	for _, path := range flag.Args() {
		if fi, err := os.Stat(path); err != nil {
			log.Fatalf("%s: %s", path, err)
		} else if fi.IsDir() {
			inputs = append(inputs, scanDir(path)...)
		} else {
			inputs = append(inputs, parseInput(path, fi))
		}
	}
	if len(inputs) == 0 {
		log.Fatal("no input found")
	}
	for _, in := range inputs {
		in.Dump()
		fmt.Printf("\n")
	}
}
