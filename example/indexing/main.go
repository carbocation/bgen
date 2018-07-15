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
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	path := flag.String("bgen", "", "Filename of the bgen file to process")
	idxPath := flag.String("bgi", "", "Filename of the bgi (index) file to process")
	flag.Parse()

	if strings.HasPrefix(*path, "~/") {
		usr, err := user.Current()
		if err != nil {
			log.Fatalln(pfx.Err(err))
		}
		*path = filepath.Join(usr.HomeDir, (*path)[2:])
	}

	if *idxPath == "" {
		*idxPath = *path + ".bgi"
	}

	if strings.HasPrefix(*idxPath, "~/") {
		usr, err := user.Current()
		if err != nil {
			log.Fatalln(pfx.Err(err))
		}
		*idxPath = filepath.Join(usr.HomeDir, (*idxPath)[2:])
	}

	log.Println("Opening bgen:", *path)
	bg, err := bgen.Open(*path)
	if err != nil {
		log.Fatalln(err)
	}
	defer bg.Close()

	bgi, err := bgen.OpenBGI(*idxPath)
	if err != nil {
		log.Fatalln(err)
	}
	defer bgi.Close()
	bgi.Metadata.FirstThousandBytes = nil

	log.Printf("BGI Metadata: %+v\n", bgi.Metadata)
	log.Printf("BGEN data: %+v\n", bg)

	rows, err := bgi.DB.Queryx("SELECT * FROM Variant ORDER BY Chromosome ASC, Position ASC")
	if err != nil {
		log.Fatalln(err)
	}
	defer rows.Close()
	i := 0
	var row bgen.VariantIndex
	for rows.Next() {
		if err := rows.StructScan(&row); err != nil {
			log.Fatalln(err)
		}
		if i%30 == 0 {
			fmt.Printf("%d) %+v\n", i, row)
		}
		i++
	}
	rows.Close()

	log.Println("Saw indexes for", i, "variants")

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

		if i > 1 {
			continue
		}

		log.Printf("Variant %d) %+v ProbBits: %d\n", i, v, v.Probabilities.NProbabilityBits)
		log.Printf("ProbabilityLayout2: %+v\n", v.Probabilities)
	}

	if vr.Error() != nil {
		log.Println("VR error:", vr.Error())
	}
}
