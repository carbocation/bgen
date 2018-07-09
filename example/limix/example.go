package main

import (
	"flag"
	"log"

	"bitbucket.org/kathiresanlab/bgen"
)

func main() {
	path := flag.String("filename", "example.bgen", "Filename of the bgen file to process")
	// path := "example.bgen"

	bg, err := bgen.Open(*path)
	if err != nil {
		log.Fatalln(err)
	}
	defer bg.Close()

	log.Printf("%+v\n", bg)
}
