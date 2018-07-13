# bgen

**BGEN** is a [BGEN file format](http://www.well.ox.ac.uk/~gav/bgen_format/) reader for golang.

This package supports the most common use-cases for BGEN specifications 1.2 and 1.3. Specifically, it supports bgen with probability formats using 8 bits (the UK Biobank standard), 16 bits, and 32 bits. It does not yet support phased data. This does not support BGEN specification 1.1.

## Installation
```bash
go get github.com/carbocation/bgen
```

## Requirements
For BGEN specification 1.2, this package is immediately usable after `go get`. 

Because ZStandard requires cgo, support for BGEN specification 1.3 is pushed into a separate branch (bgen13). For those using specification 1.2, this greatly simplifies the cross-compile process.

## API
The API is under active development. The main thrust is to convert most functions that return slices into functions that return readers.