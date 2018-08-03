# bgen

**BGEN** is a [BGEN file format](http://www.well.ox.ac.uk/~gav/bgen_format/) reader for golang.

This package supports the most common use-cases for BGEN specifications 1.1, 1.2, and 1.3. It does not yet support phased data.

## Installation
```bash
go get github.com/carbocation/bgen
```

## Requirements
For BGEN specifications 1.1 and 1.2, this package is immediately usable after `go get`. 

Because ZStandard requires cgo, support for BGEN specification 1.3 is pushed into a separate branch (bgen13). For those using specification 1.2, this greatly simplifies the cross-compile process.

## API
The API is under active development and the public API may change for now.

## Gotchas
The SampleProbabilities.Probabilities all share the same underlying slice. If you need to manipulate 
these (e.g., expand the slice, etc), the values should first be copied into a new slice. The original 
slices should not be modified.

For the current API, please see the [BGEN Godoc](https://godoc.org/github.com/carbocation/bgen)
