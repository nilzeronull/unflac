package main

import (
	"os/exec"
	"strconv"
	"strings"
)

type AudioFile struct {
	Path string
}

func (af *AudioFile) SampleRate() (sampleRate int, err error) {
	var out strings.Builder
	cmd := exec.Command("metaflac", "--show-sample-rate", af.Path)
	cmd.Stdout = &out
	if err = cmd.Run(); err == nil {
		sampleRate, err = strconv.Atoi(strings.TrimSpace(out.String()))
	}

	return
}
