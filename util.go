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

func decoderToUTF8For(b []byte) (dec *encoding.Decoder, err error) {
	var best *chardet.Result
	var enc encoding.Encoding
	if best, err = chardet.NewTextDetector().DetectBest(b); err != nil {
		return
	} else if best.Confidence >= 25 {
		if enc, err = ianaindex.IANA.Encoding(best.Charset); err != nil {
			err = nil
		}
	}
	if enc == nil {
		enc = encoding.Nop
	}
	dec = enc.NewDecoder()
	return
}

func openFileUTF8(path string) (r io.ReadCloser, err error) {
	if r, err = os.Open(path); err != nil {
		return
	}
	defer r.Close()

	buf := new(bytes.Buffer)
	buf.ReadFrom(r)
	var dec *encoding.Decoder
	if dec, err = decoderToUTF8For(buf.Bytes()); err != nil {
		return
	}
	r = ioutil.NopCloser(dec.Reader(buf))
	return
}

func pathReplaceChars(s string) string {
	return strings.ReplaceAll(s, "/", "∕")
}
