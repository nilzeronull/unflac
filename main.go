package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

var (
	outputDir = flag.String("o", ".", "Output directory")

	extensions = []string{
		".flac",
	}
)

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

func main() {
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
	for _, in := range inputs {
		in.Dump()
		fmt.Printf("\n")
	}
}
