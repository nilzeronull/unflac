package main

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/ftrvxmtrx/chardet"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/ianaindex"
)

func openFileUTF8(path string) (r io.ReadCloser, err error) {
	if r, err = os.Open(path); err != nil {
		return
	}
	defer r.Close()

	buf := new(bytes.Buffer)
	buf.ReadFrom(r)
	d := chardet.NewTextDetector()
	var best *chardet.Result
	var enc encoding.Encoding
	if best, err = d.DetectBest(buf.Bytes()); err != nil {
		return
	} else if enc, err = ianaindex.IANA.Encoding(best.Charset); err != nil {
		return
	}
	return ioutil.NopCloser(enc.NewDecoder().Reader(buf)), nil
}

func pathReplaceChars(s string) string {
	return strings.ReplaceAll(s, "/", "∕")
}
