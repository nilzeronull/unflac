package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ftrvxmtrx/chub/cue"
	"github.com/gammazero/workerpool"
)

type Input struct {
	Path  string       `json:"path"`
	Audio []*AudioFile `json:"audio"`

	TrackNumberFmt string `json:"-"`

	Composer    string `json:"composer,omitempty"`
	Performer   string `json:"performer,omitempty"`
	SongWriter  string `json:"songWriter,omitempty"`
	Title       string `json:"title,omitempty"`
	Genre       string `json:"genre,omitempty"`
	Date        string `json:"date,omitempty"`
	TotalTracks int    `json:"totalTracks,omitempty"`
	TotalDisks  int    `json:"totalDisks,omitempty"`
}

func NewInput(path string) (in *Input, err error) {
	in = &Input{Path: path}

	var cueReader io.ReadCloser
	if cueReader, err = openFileUTF8(path); err != nil {
		return
	}
	defer cueReader.Close()

	cueRaw, _ := ioutil.ReadAll(cueReader)
	cueRaw = bytes.TrimPrefix(cueRaw, []byte{0xef, 0xbb, 0xbf}) // remove the nasty BOM
	var sheet *cue.Sheet
	if sheet, err = cueSheetFromBytes(cueRaw); err != nil {
		return
	}

	dirPath := filepath.Dir(path)
	var audio *AudioFile
	var filesFromCue []*cue.File
	for _, f := range sheet.Files {
		if f.Type != cue.FileTypeWave {
			continue
		} else if audio, err = NewAudio(filepath.Join(dirPath, f.Name)); err != nil {
			err = fmt.Errorf("%s: %s", f.Name, err)
			return
		}
		in.Audio = append(in.Audio, audio)
		filesFromCue = append(filesFromCue, f)
	}
	if len(in.Audio) < 1 {
		return nil, fmt.Errorf("no audio files")
	}

	in.Performer = sheet.Performer
	in.SongWriter = sheet.Songwriter
	in.Title = sheet.Title

	var diskNumber int
	for _, c := range sheet.Comments {
		if words := strings.SplitAfterN(c, " ", 2); len(words) < 2 {
			continue
		} else {
			switch words[0] {
			case "DATE":
				in.Date = words[1]
			case "GENRE":
				in.Genre = words[1]
			case "COMPOSER":
				in.Composer = words[1]
			case "DISCNUMBER":
				if len(in.Audio) == 1 {
					// FIXME no idea what to do with several discnumber comments in a cue sheet
					if diskNumber, err = strconv.Atoi(words[1]); err != nil {
						return
					}
				}
			case "TOTALDISCS":
				in.TotalDisks, err = strconv.Atoi(words[1])
			}
		}
	}

	for i, file := range filesFromCue {
		audio = in.Audio[i]

		var prevAudioTrack *Track
		audio.Tracks = make([]*Track, 0)
		for _, ft := range file.Tracks {
			if prevAudioTrack != nil && prevAudioTrack.EndAtSample == 0 {
				prevAudioTrack.SetNextIndexes(audio.SampleRate, ft.Indexes)
				prevAudioTrack = nil
			}
			if ft.DataType != cue.DataTypeAudio {
				continue
			}

			t := &Track{
				Number:      ft.Number,
				Title:       ft.Title,
				Performer:   ft.Performer,
				SongWriter:  ft.Songwriter,
				Album:       in.Title,
				Genre:       in.Genre,
				Date:        in.Date,
				TotalTracks: &in.TotalTracks,
				TotalDisks:  in.TotalDisks,
				DiskNumber:  diskNumber,
			}
			for _, c := range ft.Comments {
				if strings.HasPrefix(c, "COMPOSER") {
					words := strings.SplitAfterN(c, " ", 2)
					t.Composer = words[1]
				}
			}

			audio.Tracks = append(audio.Tracks, t)
			in.TotalTracks++
			if t.Number == 0 {
				t.Number = len(audio.Tracks)
			}
			if err = t.SetIndexes(audio.SampleRate, ft.Indexes); err != nil {
				return
			}
			prevAudioTrack = t
		}
	}

	if in.TotalTracks > 99 {
		in.TrackNumberFmt = "%03d"
	} else {
		in.TrackNumberFmt = "%02d"
	}

	return
}

func (in *Input) Artist() string {
	switch {
	case in.Composer != "":
		return in.Composer
	case in.SongWriter != "":
		return in.SongWriter
	case in.Performer != "":
		return in.Performer
	}

	var artist string
	for _, a := range in.Audio {
		for _, t := range a.Tracks[1:] {
			if t.Artist() != artist && artist != "" {
				return "Various Artists"
			}
			artist = t.Artist()
		}
	}

	return "Unknown Artist"
}

func (in *Input) OutputPath() (path string) {
	var album string
	if in.Date != "" {
		album = in.Date + " - "
	}
	if in.Title != "" {
		album += in.Title
	} else {
		album += "Unknown Album" // FIXME this name sucks
	}

	// FIXME make sure the final path doesn't exist?
	return filepath.Join(pathReplaceChars(in.Artist()), pathReplaceChars(album))
}

func (in *Input) TrackFilename(t *Track) (path string) {
	path = fmt.Sprintf(in.TrackNumberFmt, t.Number)
	if t.Title != "" {
		path += " - " + t.Title
	}
	path = pathReplaceChars(path + "." + *format)
	return
}

func (in *Input) Dump() {
	for _, a := range in.Audio {
		fmt.Printf("%s\n", a.Path)
		dirPath := filepath.Join(*outputDir, in.OutputPath())
		for _, t := range a.Tracks {
			if len(trackArgs) > 0 && !trackArgs.Has(t.Number) {
				continue
			}

			trackPath := filepath.Join(dirPath, in.TrackFilename(t))
			fmt.Printf("%s\n\tfirst=%d last=%d\n", trackPath, t.StartAtSample, t.EndAtSample)
		}
	}
}

func (in *Input) Split(pool *workerpool.WorkerPool, firstErr chan<- error) (err error) {
	dirPath := filepath.Join(*outputDir, in.OutputPath())
	if err = os.MkdirAll(dirPath, 0755); err != nil {
		return
	}

	for _, a := range in.Audio {
		for _, t := range a.Tracks {
			if len(trackArgs) > 0 && !trackArgs.Has(t.Number) {
				continue
			}

			pool.Submit(func(a *AudioFile, t *Track) func() {
				return func() {
					trackPath := filepath.Join(dirPath, in.TrackFilename(t))
					if err = a.Extract(t, trackPath); err != nil {
						firstErr <- fmt.Errorf("%s: %s", trackPath, err)
					} else if !*quiet {
						fmt.Printf("%s\n", trackPath)
					}
				}
			}(a, t))
		}
	}

	return
}
