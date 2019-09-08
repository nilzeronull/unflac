# unflac

A command line tool for fast frame accurate FLAC image + cue sheet splitting.

This project is started mostly out of frustration over supporting
[split2flac](https://github.com/ftrvxmtrx/split2flac) with all the
external dependencies and their quirks.

## Installation and running

You need [Go](https://golang.org/) installed.

```
go get github.com/ftrvxmtrx/unflac
~/go/bin/unflac -h
```

## Dependencies

 * ffmpeg (ffprobe command is also used, part of ffmpeg package)

Yeah, that's it.

## Some useful ffmpeg options

 * Set a specific sample rate for output files: `unflac -arg-ffmpeg -ar -arg-ffmpeg 44100 ...`

`man ffmpeg` contains a lot more.

## TODO

 * an option to format the output path in a specific way
 * copying cue, log, images over to the destination
 * clean up on errors
 * replay gain
 * support multiple artist/performer/composer?

## DONE

 * basic splitting
 * tagging
 * split to other formats (ogg/vorbis, mp3)
 * parallel extraction
 * support more input formats (wavpack, wave, mac)
 * get the artist/performer/composer right
 * multi-disk cue sheets support
 * detect cue sheet encoding and convert to utf8

## NOTES

 * using id3 v2.3 tags because it seems basically no software reads id3 v2.4
