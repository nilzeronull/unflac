package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/vchimishuk/chub/cue"
)

type Input struct {
	Audio  *AudioFile `json:"audio"`
	Tracks []Track    `json:"tracks,omitempty"`

	TrackNumberFmt string `json:"-"`

	Performer  string `json:"performer,omitempty"`
	SongWriter string `json:"songWriter,omitempty"`
	Title      string `json:"title,omitempty"`
	Genre      string `json:"genre,omitempty"`
	Date       string `json:"date,omitempty"`
}

func NewInput(path string) (in *Input, err error) {
	in = new(Input)
	if in.Audio, err = NewAudioFile(path); err != nil {
		return
	}

	var cueReader io.ReadCloser
	if cueReader, err = in.Audio.OpenCueSheet(); err != nil {
		return
	}

	var sheet *cue.Sheet
	if sheet, err = cue.Parse(cueReader); err != nil {
		return
	}

	if len(sheet.Files) != 1 {
		return nil, fmt.Errorf("unsupported number of files: %d", len(sheet.Files))
	} else if sheet.Files[0].Type != cue.FileTypeWave {
		return nil, fmt.Errorf("unsupported file type %d", sheet.Files[0].Type)
	}

	in.Performer = sheet.Performer
	in.SongWriter = sheet.Songwriter
	in.Title = sheet.Title

	var date, genre string
	for _, c := range sheet.Comments {
		if strings.HasPrefix(c, "DATE") {
			words := strings.Fields(c)
			date = words[len(words)-1]
		} else if strings.HasPrefix(c, "GENRE") {
			words := strings.SplitAfterN(c, " ", 2)
			genre = words[1]
		}
	}
	in.Genre = genre
	in.Date = date

	in.Tracks = make([]Track, len(sheet.Files[0].Tracks))
	if len(in.Tracks) >= 100 {
		in.TrackNumberFmt = "%03d"
	} else {
		in.TrackNumberFmt = "%02d"
	}
	for i, ft := range sheet.Files[0].Tracks {
		t := &in.Tracks[i]
		*t = Track{
			Number:     ft.Number,
			Title:      ft.Title,
			Performer:  ft.Performer,
			SongWriter: ft.Songwriter,
			Genre:      genre,
			Date:       date,
		}
		if t.SongWriter == "" {
			if in.SongWriter == "" {
				t.SongWriter = t.Performer
			} else {
				t.SongWriter = in.SongWriter
			}
		}
		if t.Number == 0 {
			t.Number = i + 1
		}
		if err = t.SetIndexes(in.Audio.SampleRate, ft.Indexes); err != nil {
			break
		}
		if i > 0 {
			in.Tracks[i-1].SetNextIndexes(in.Audio.SampleRate, ft.Indexes)
		}
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
	var album string
	if in.Date != "" {
		album = in.Date + " - "
	}
	if in.Title != "" {
		album += in.Title
	} else {
		album += "Unknown Album" // FIXME this name sucks
	}

	performer = pathReplaceChars(performer)
	album = pathReplaceChars(album)

	// FIXME make sure the final path doesn't exist?
	return filepath.Join(performer, album)
}

func (in *Input) Dump() {
	fmt.Printf("%s\n", in.Audio.Path)
	dirPath := filepath.Join(*outputDir, in.OutputPath())
	for _, t := range in.Tracks {
		trackPath := filepath.Join(dirPath, t.OutputPath(in, ".flac"))
		fmt.Printf("%s\n\tfirst=%d last=%d\n", trackPath, t.FirstSample, t.LastSample)
	}
}

func (in *Input) Split() (err error) {
	dirPath := filepath.Join(*outputDir, in.OutputPath())
	if err = os.MkdirAll(dirPath, 0755); err != nil {
		return
	}

	for _, t := range in.Tracks {
		filename := filepath.Join(dirPath, t.OutputPath(in, "."+*format))
		if err = in.Audio.Extract(in, &t, filename); err != nil {
			return
		}
	}

	return
}
