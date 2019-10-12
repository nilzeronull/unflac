package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"text/template"

	"github.com/gammazero/workerpool"
)

type IntListFlag []int
type StringListFlag []string

var (
	outputDir = flag.String("o", ".", "Output directory")
	quiet     = flag.Bool("q", false, "Only print errors")
	dryRun    = flag.Bool("d", false, "Dry run")
	jsonDump  = flag.Bool("j", false, "Dump all inputs as json")
	format    = flag.String("f", "flac", "Output format")

	trackArgs  IntListFlag
	ffmpegArgs StringListFlag
	nameTmpl   *template.Template
)

func main() {
	flag.Var(&ffmpegArgs, "arg-ffmpeg", "Add an argument to ffmpeg")
	flag.Var(&trackArgs, "t", `Extract specific track(s). Example: "-t 1 -t 2"`)
	nameTmplV := flag.String(
		"n",
		`{{.Input.Artist | Elem -}} / {{- with .Input.Date}}{{.}} - {{end}}{{with .Input.Title}}{{. | Elem}}{{else}}Unknown Album{{end -}} / {{- printf .Input.TrackNumberFmt .Track.Number}} - {{.Track.Title | Elem}}`,
		"File naming template",
	)
	flag.Parse()

	var err error
	nameTmpl = template.New("-n").Funcs(template.FuncMap{"Elem": pathReplaceChars})
	if nameTmpl, err = nameTmpl.Parse(*nameTmplV); err != nil {
		log.Fatal(err)
	}

	args := flag.Args()
	if len(args) == 0 {
		args = []string{"."}
	}
	var inputs []*Input
	for _, path := range args {
		if fi, err := os.Stat(path); err != nil {
			log.Fatalf("%s: %s", path, err)
		} else if fi.IsDir() {
			inputs = append(inputs, scanDir(path)...)
		} else if in, err := NewInput(path); err != nil {
			log.Fatalf("%s: %s", path, err)
		} else {
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
				log.Fatalf("%s: %s", in.Path, err)
			}
		}
	}
	wp.StopWait()

	if *jsonDump {
		json.NewEncoder(os.Stdout).Encode(inputs)
	}
}

func (l *IntListFlag) String() string {
	return fmt.Sprintf("%+v", *l)
}

func (l *IntListFlag) Set(s string) (err error) {
	var i int
	if i, err = strconv.Atoi(s); err == nil {
		*l = append(*l, i)
	}
	return
}

func (l *IntListFlag) Has(i int) bool {
	for _, x := range *l {
		if x == i {
			return true
		}
	}
	return false
}

func (l *StringListFlag) String() string {
	return strings.Join(*l, " ")
}

func (l *StringListFlag) Set(s string) error {
	*l = append(*l, s)
	return nil
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
				} else if strings.HasSuffix(strings.ToLower(fi.Name()), ".cue") {
					var in *Input
					if in, err = NewInput(fiPath); err != nil {
						log.Fatalf("%s: %s", fiPath, err)
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
