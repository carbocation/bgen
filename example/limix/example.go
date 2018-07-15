package main

import (
	"flag"
	"fmt"
	"log"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/carbocation/bgen"
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

	samples, err := bgen.ReadSamples(bg)
	if err != nil {
		log.Println(err)
	} else {

		i := 0
		for _, sample := range samples {
			fmt.Println(i, sample.SampleID)
			i++

			if i > 10 {
				break
			}
		}
		if i > 0 {
			log.Println("Saw up to", samples[i-1].SampleID)
		}

		log.Println("Iterated over", i, "samples")
	}

	vr := bg.NewVariantReader()
	for i := 1; ; i++ {
		v := vr.Read()
		if v == nil {
			break
		}

		for j, pb := range v.Probabilities.SampleProbabilities {
			if j > 10 {
				continue
			}

			if i > 10 {
				continue
			}

			if pb.Missing {
				log.Printf("\tProb %d) %s\n", j, "is missing")
			} else {
				log.Printf("\tProb %d) %+v\n", j, pb.Probabilities)
			}
		}

		if i > 10 {
			continue
		}

		log.Printf("Variant %d) %+v ProbBits: %d\n", i, v, v.Probabilities.NProbabilityBits)
		log.Printf("ProbabilityLayout2: %+v\n", v.Probabilities)
	}

	if vr.Error() != nil {
		log.Println("VR error:", vr.Error())
	}
}
