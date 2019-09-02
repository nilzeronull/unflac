# unflac

A command line tool for frame accurate FLAC image + cue sheet splitting.

WIP.

This project is started mostly out of frustration over supporting
[split2flac](https://github.com/ftrvxmtrx/split2flac) with all the
external dependencies and their quirks.

## Dependencies

 * metaflac
 * ffmpeg

Yeah, that's it.

## TODO

 * parallel extraction
 * an option to format the output path in a specific way
 * detect cue sheet encoding and convert to utf8
 * copying cue, log, images over to the destination
 * clean up on errors
 * support more input formats (wave, wavpack, mac)

## DONE

 * basic splitting
 * tagging
 * split to other formats (ogg/vorbis, mp3)
