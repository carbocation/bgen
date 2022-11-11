# bgen

**BGEN** is a BGEN parser for golang. It can read files in the [.bgen format](http://www.well.ox.ac.uk/~gav/bgen_format/), but it cannot write them.

This package supports the most common use-cases for BGEN specifications 1.1, 1.2, and 1.3. It does not yet support phased data.

## Installation
```bash
go get github.com/carbocation/bgen
```

## Requirements
For BGEN specifications 1.1, 1.2, and 1.3 this package is immediately usable after `go get`. Only unphased samples are correctly supported currently.

## API
The API is under active development and the public API may change for now.

For the current API, please see the [BGEN Godoc](https://godoc.org/github.com/carbocation/bgen)
