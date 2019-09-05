package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/ftrvxmtrx/chub/cue"
	"github.com/gammazero/workerpool"
)

type Input struct {
	Audio  *AudioFile `json:"audio"`
	Tracks []*Track   `json:"tracks,omitempty"`

	TrackNumberFmt string `json:"-"`

	Composer   string `json:"composer,omitempty"`
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

	var prevAudioTrack *Track
	in.Tracks = make([]*Track, 0)
	for _, ft := range sheet.Files[0].Tracks {
		if prevAudioTrack != nil && prevAudioTrack.EndAtSample == 0 {
			prevAudioTrack.SetNextIndexes(in.Audio.SampleRate, ft.Indexes)
			prevAudioTrack = nil
		}
		if ft.DataType != cue.DataTypeAudio {
			continue
		}

		t := &Track{
			Number:     ft.Number,
			Title:      ft.Title,
			Performer:  ft.Performer,
			SongWriter: ft.Songwriter,
			Album:      in.Title,
			Genre:      in.Genre,
			Date:       in.Date,
		}
		for _, c := range ft.Comments {
			if strings.HasPrefix(c, "COMPOSER") {
				words := strings.SplitAfterN(c, " ", 2)
				t.Composer = words[1]
			}
		}

		in.Tracks = append(in.Tracks, t)
		if t.Number == 0 {
			t.Number = len(in.Tracks)
		}
		if err = t.SetIndexes(in.Audio.SampleRate, ft.Indexes); err != nil {
			return
		}
		prevAudioTrack = t
	}
	if len(in.Tracks) > 99 {
		in.TrackNumberFmt = "%03d"
	} else {
		in.TrackNumberFmt = "%02d"
	}
	for _, t := range in.Tracks {
		t.TotalTracks = len(in.Tracks)
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

	for _, t := range in.Tracks[1:] {
		if t.Artist() != in.Tracks[0].Artist() {
			return "Various Artists"
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
	fmt.Printf("%s\n", in.Audio.Path)
	dirPath := filepath.Join(*outputDir, in.OutputPath())
	for _, t := range in.Tracks {
		trackPath := filepath.Join(dirPath, in.TrackFilename(t))
		fmt.Printf("%s\n\tfirst=%d last=%d\n", trackPath, t.StartAtSample, t.EndAtSample)
	}
}

func (in *Input) Split(pool *workerpool.WorkerPool, firstErr chan<- error) (err error) {
	dirPath := filepath.Join(*outputDir, in.OutputPath())
	if err = os.MkdirAll(dirPath, 0755); err != nil {
		return
	}

	for _, t := range in.Tracks {
		pool.Submit(func(t *Track) func() {
			return func() {
				trackPath := filepath.Join(dirPath, in.TrackFilename(t))
				if err = in.Audio.Extract(t, trackPath); err != nil {
					firstErr <- fmt.Errorf("%s: %s", trackPath, err)
				} else if !*quiet {
					fmt.Printf("%s\n", trackPath)
				}
			}
		}(t))
	}

	return
}
