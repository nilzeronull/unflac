# unflac

A command line tool for fast frame accurate FLAC image + cue sheet splitting.

This project is started mostly out of frustration over supporting
[split2flac](https://github.com/ftrvxmtrx/split2flac) with all the
external dependencies and their quirks.

## Installation and running

You need [Go](https://golang.org/) installed.

```
go get github.com/ftrvxmtrx/unflac
unflac -h
```

## Dependencies

 * metaflac
 * ffmpeg

Yeah, that's it.

## TODO

 * get the artist/performer/composer right (and support multiple)
 * an option to format the output path in a specific way
 * detect cue sheet encoding and convert to utf8
 * copying cue, log, images over to the destination
 * clean up on errors
 * replay gain
 * support more input formats (wave, wavpack, mac)

## DONE

 * basic splitting
 * tagging
 * split to other formats (ogg/vorbis, mp3)
 * parallel extraction

## NOTES

 * using id3 v2.3 tags because it seems basically no software reads id3 v2.4
