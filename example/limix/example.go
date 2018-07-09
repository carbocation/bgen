package main

import (
	"log"

	"bitbucket.org/kathiresanlab/bgen"
)

func main() {
	path := "example.bgen"

	bg, err := bgen.Open(path)
	if err != nil {
		log.Fatalln(err)
	}

	log.Printf("%+v\n", bg)
}
