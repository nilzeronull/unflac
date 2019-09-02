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

func (af *AudioFile) Extract(t *Track, filename string) error {
	atrim := fmt.Sprintf("atrim=start_sample=%d", t.FirstSample)
	if t.LastSample != 0 {
		atrim = fmt.Sprintf("%s:end_sample=%d", atrim, t.LastSample)
	}
	cmd := exec.Command("ffmpeg", "-loglevel", "error", "-i", af.Path, "-af", atrim, filename)
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
