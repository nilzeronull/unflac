package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/ftrvxmtrx/chub/cue"
	"github.com/gammazero/workerpool"
	"golang.org/x/text/encoding"
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
	var sheet *cue.Sheet
	var filesFromCue []*cue.File

	if cueReader, err = openFileUTF8(path); err != nil {
		return
	}
	cueRaw := new(bytes.Buffer)
	cueRaw.ReadFrom(cueReader)

	if sheet, err = cue.Parse(bytes.NewBuffer(cueRaw.Bytes()), 0); err != nil {
		return
	} else {
		buf := new(bytes.Buffer)
		buf.WriteString(sheet.Performer)
		buf.WriteString(sheet.Songwriter)
		buf.WriteString(sheet.Title)
		for _, f := range sheet.Files {
			if f.Type == cue.FileTypeWave {
				for _, t := range f.Tracks {
					buf.WriteString(t.Title)
				}
				buf.WriteString(f.Name)
			}
		}
		var dec *encoding.Decoder
		if dec, err = decoderToUTF8For(buf.Bytes()); err == nil {
			if sheet, err = cue.Parse(dec.Reader(cueRaw), 0); err != nil {
				return
			}
		}
	}

	dirPath := filepath.Dir(path)
	var audio *AudioFile
	for _, f := range sheet.Files {
		if f.Type != cue.FileTypeWave {
			continue
		} else if audio, err = NewAudio(filepath.Join(dirPath, f.Name)); err != nil {
			err = fmt.Errorf("%s: %s", f.Name, err)
			return
		}
		in.TotalDisks++
		in.Audio = append(in.Audio, audio)
		filesFromCue = append(filesFromCue, f)
	}
	if len(in.Audio) < 1 {
		return nil, fmt.Errorf("no audio files")
	}

	in.Performer = sheet.Performer
	in.SongWriter = sheet.Songwriter
	in.Title = sheet.Title

	for _, c := range sheet.Comments {
		if strings.HasPrefix(c, "DATE") {
			words := strings.Fields(c)
			in.Date = words[len(words)-1]
		} else if strings.HasPrefix(c, "GENRE") {
			words := strings.SplitAfterN(c, " ", 2)
			in.Genre = words[1]
		} else if strings.HasPrefix(c, "COMPOSER") {
			words := strings.SplitAfterN(c, " ", 2)
			in.Composer = words[1]
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
				TotalDisks:  &in.TotalDisks,
				DiskNumber:  i + 1,
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
	if in.Composer != "" {
		return in.Composer
	} else if in.SongWriter != "" {
		return in.SongWriter
	} else if in.Performer != "" {
		return in.Performer
	}

	var artist string
	for _, a := range in.Audio {
		for _, t := range a.Tracks[1:] {
			if t.Artist() != artist && artist != "" {
				return "Various Artists"
			} else {
				artist = t.Artist()
			}
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
			pool.Submit(func(t *Track) func() {
				return func() {
					trackPath := filepath.Join(dirPath, in.TrackFilename(t))
					if err = a.Extract(t, trackPath); err != nil {
						firstErr <- fmt.Errorf("%s: %s", trackPath, err)
					} else if !*quiet {
						fmt.Printf("%s\n", trackPath)
					}
				}
			}(t))
		}
	}

	return
}
