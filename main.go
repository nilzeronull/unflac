package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/gammazero/workerpool"
)

type ListFlag []string

var (
	outputDir = flag.String("o", ".", "Output directory")
	quiet     = flag.Bool("q", false, "Only print errors")
	dryRun    = flag.Bool("d", false, "Dry run")
	jsonDump  = flag.Bool("j", false, "Dump all inputs as json")
	format    = flag.String("f", "flac", "Output format")

	ffmpegArgs ListFlag

	extensions = []string{
		".flac",
	}
)

func main() {
	flag.Var(&ffmpegArgs, "arg-ffmpeg", "Add an argument to ffmpeg")
	flag.Parse()

	var inputs []*Input
	for _, path := range flag.Args() {
		if fi, err := os.Stat(path); err != nil {
			log.Fatalf("%s: %s", path, err)
		} else if fi.IsDir() {
			inputs = append(inputs, scanDir(path)...)
		} else {
			var in *Input
			if in, err = NewInput(path); err != nil {
				log.Fatalf("%s: %s", path, err)
			}
			inputs = append(inputs, in)
		}
	}
	if len(inputs) == 0 {
		log.Fatal("no input found")
	}

	wp := workerpool.New(runtime.NumCPU())
	firstErr := make(chan error)
	go func() {
		log.Fatalf("%s", <-firstErr)
	}()

	for _, in := range inputs {
		if !*dryRun {
			if err := in.Split(wp, firstErr); err != nil {
				log.Fatalf("%s: %s", in.Audio.Path, err)
			}
		}
	}
	wp.StopWait()

	if *jsonDump {
		json.NewEncoder(os.Stdout).Encode(inputs)
	}
}

func (l *ListFlag) String() string {
	return strings.Join(*l, " ")
}

func (l *ListFlag) Set(s string) error {
	*l = append(*l, s)
	return nil
}

func pathReplaceChars(s string) string {
	return strings.ReplaceAll(s, "/", "∕")
}

func scanDir(path string) (ins []*Input) {
	var f *os.File
	var fis []os.FileInfo
	var err error

	if f, err = os.Open(path); err == nil {
		if fis, err = f.Readdir(0); err == nil {
			for _, fi := range fis {
				fiPath := filepath.Join(path, fi.Name())
				if fi.IsDir() {
					ins = append(ins, scanDir(fiPath)...)
				} else {
					for _, ext := range extensions {
						if strings.HasSuffix(fi.Name(), ext) {
							var in *Input
							if in, err = NewInput(fiPath); err != nil {
								log.Printf("%s: %s", fiPath, err)
								err = nil
							} else {
								ins = append(ins, in)
							}
							break
						}
					}
				}
			}
		}
	}
	if err != nil {
		log.Fatalf("%s: %s", path, err)
	}
	return
}
