package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

type AudioFile struct {
	Path       string `json:"path"`
	Ext        string `json:"ext"`
	SampleRate int    `json:"sampleRate,omitempty"`
}

type Tag struct {
	Name  string
	Value string
}

func NewAudioFile(path string) (af *AudioFile, err error) {
	af = &AudioFile{Path: path, Ext: filepath.Ext(path)}

	switch af.Ext {
	case ".flac":
		var out strings.Builder
		cmd := exec.Command("metaflac", "--show-sample-rate", af.Path)
		cmd.Stdout = &out
		if err = cmd.Run(); err == nil {
			af.SampleRate, err = strconv.Atoi(strings.TrimSpace(out.String()))
		}
	}

	return
}

func (af *AudioFile) Extract(t *Track, filename string) (err error) {
	// ffmpeg options
	args := []string{"-loglevel", "error", "-i", af.Path}
	tags := []Tag{
		{"artist", t.SongWriter},
		{"performer", t.Performer},
		{"album", t.Album},
		{"title", t.Title},
		{"genre", t.Genre},
		{"date", t.Date},
	}

	switch *format {
	case "flac":
		tags = append(tags,
			Tag{"tracknumber", strconv.Itoa(t.Number)},
			Tag{"tracktotal", strconv.Itoa(t.TotalTracks)},
		)

	case "mp3":
		tags = append(tags,
			Tag{"track", fmt.Sprintf("%d/%d", t.Number, t.TotalTracks)},
		)
		args = append(args, "-write_id3v2", "1", "-id3v2_version", "3", "-qscale:a", "3")
	}

	for _, t := range tags {
		if t.Value != "" {
			args = append(args, "-metadata", fmt.Sprintf("%s=%s", t.Name, t.Value))
		}
	}

	atrim := fmt.Sprintf("atrim=start_sample=%d", t.StartAtSample)
	if t.EndAtSample != 0 {
		atrim = fmt.Sprintf("%s:end_sample=%d", atrim, t.EndAtSample)
	}
	args = append(args, "-af", atrim, filename)

	cmd := exec.Command("ffmpeg", args...)
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func (af *AudioFile) OpenCueSheet() (r io.ReadCloser, err error) {
	external := strings.TrimSuffix(af.Path, af.Ext) + ".cue"
	if r, err = os.Open(external); err == nil {
		return
	}

	// fall back to internal one
	switch af.Ext {
	case ".flac":
		out := new(bytes.Buffer)
		cmd := exec.Command("metaflac", "--export-cuesheet-to=-", af.Path)
		cmd.Stdout = out
		if err = cmd.Run(); err == nil {
			r = ioutil.NopCloser(out)
		} else {
			err = fmt.Errorf("no CUE sheet found")
		}
	default:
		err = fmt.Errorf("internal CUE sheet reading not implemented for %q files", af.Ext)
	}

	return
}
