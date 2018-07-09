package main

import (
	"flag"
	"log"
	"os/user"
	"path/filepath"
	"strings"

	"bitbucket.org/kathiresanlab/bgen"
	"github.com/carbocation/pfx"
)

func main() {
	path := flag.String("filename", "example.bgen", "Filename of the bgen file to process")
	flag.Parse()

	if strings.HasPrefix(*path, "~/") {
		usr, err := user.Current()
		if err != nil {
			log.Fatalln(pfx.Err(err))
		}
		*path = filepath.Join(usr.HomeDir, (*path)[2:])
	}

	bg, err := bgen.Open(*path)
	if err != nil {
		log.Fatalln(err)
	}
	defer bg.Close()

	log.Printf("%+v\n", bg)
}
