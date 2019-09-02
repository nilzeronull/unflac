package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
)

var (
	outputDir = flag.String("o", ".", "Output directory")
)

func scanDir(path string) (ins []*Input) {
	var f *os.File
	var fis []os.FileInfo
	var err error

	if f, err = os.Open(path); err == nil {
		if fis, err = f.Readdir(0); err == nil {
			for _, fi := range fis {
				fiPath := path + "/" + fi.Name()
				if fi.IsDir() {
					ins = append(ins, scanDir(fiPath)...)
				} else if strings.HasSuffix(fi.Name(), ".flac") {
					var in *Input
					if in, err = NewInput(fiPath); err != nil {
						break
					}
					ins = append(ins, in)
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
