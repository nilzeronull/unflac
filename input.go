package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/vchimishuk/chub/cue"
)

type CueFile struct {
	Path string
}

type Input struct {
	Audio  AudioFile
	Cue    CueFile
	Tracks []Track

	TrackNumberFmt string

	Performer string
	Title     string
	Genre     string
	Date      string
}

func ParseInput(path string) (in *Input, err error) {
	in = &Input{
		Audio: AudioFile{Path: path},
		Cue:   CueFile{Path: strings.TrimSuffix(path, ".flac") + ".cue"},
	}

	var sampleRate int
	if sampleRate, err = in.Audio.SampleRate(); err != nil {
		return
	}

	// try external cue sheet
	var cueReader io.ReadCloser
	if cueReader, err = os.Open(in.Cue.Path); err != nil {
		// FIXME fall back to internal one
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
		t.SetIndexes(sampleRate, ft.Indexes)
		if t.Number == 0 {
			t.Number = i + 1
		}
		if i > 0 && in.Tracks[i-1].LastSample == 0 {
			in.Tracks[i-1].LastSample = t.FirstSample - 1
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

func (in *Input) Dump() {
	fmt.Printf("audio: %s\n", in.Audio.Path)
	fmt.Printf("cue: %s\n", in.Cue.Path)
	out := in.OutputPath()
	for _, t := range in.Tracks {
		fmt.Printf("%s/%s\n\tfirst=%d last=%d\n", out, t.OutputPath(in, ".flac"), t.FirstSample, t.LastSample)
	}
}
