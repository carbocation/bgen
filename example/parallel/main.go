package main

import (
	"flag"
	"fmt"
	"log"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/carbocation/bgen"
	"github.com/carbocation/pfx"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	path := flag.String("bgen", "", "Filename of the bgen file to process")
	idxPath := flag.String("bgi", "", "Filename of the bgi (index) file to process")
	flag.Parse()

	if *path == "" {
		flag.PrintDefaults()
		log.Fatalln("No bgen file found")
	}

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

	// Prep the readers
	offset := make(chan int64)
	output := make(chan AlleleCounter)
	done := make(chan struct{})
	confirmDone := make(chan struct{})

	go func() {
		accumulator := AlleleCounter{}
	MonitorLoop:
		for {
			select {
			case <-done:
				break MonitorLoop
			case o := <-output:
				accumulator.A += o.A
				accumulator.C += o.C
				accumulator.T += o.T
				accumulator.G += o.G
			}
		}
		log.Println("Final accumulated stats")
		log.Printf("%+v\n", accumulator)
		close(confirmDone)
	}()

	// Prep the Workers:
	log.Println("Launching", runtime.NumCPU(), "workers")
	for i := 0; i < runtime.NumCPU(); i++ {
		go Worker(i, *path, offset, output)
	}

	// Load the BGEN Index
	bgi, err := bgen.OpenBGI(*idxPath)
	if err != nil {
		log.Fatalln(err)
	}
	defer bgi.Close()
	bgi.Metadata.FirstThousandBytes = nil
	log.Printf("BGI Metadata: %+v\n", bgi.Metadata)

	rows, err := bgi.DB.Queryx("SELECT * FROM Variant ORDER BY Chromosome ASC, Position ASC")
	if err != nil {
		log.Fatalln(err)
	}
	defer rows.Close()

	var row bgen.VariantIndex
	i := 0
	for rows.Next() {
		if i%1000 == 0 {
			log.Println("Processed", i, "variants")
		}
		if err := rows.StructScan(&row); err != nil {
			log.Fatalln(err)
		}

		offset <- int64(row.FileStartPosition)
		i++
	}
	close(offset)
	time.Sleep(1 * time.Second)
	close(done) // Racey, use a waitgroup instead
	<-confirmDone
	rows.Close()

}

type AlleleCounter struct {
	A, C, T, G float64
}

func (a *AlleleCounter) Add(which string, val float64) error {
	switch which {
	case "A":
		a.A += val
	case "C":
		a.C += val
	case "T":
		a.T += val
	case "G":
		a.G += val
	default:
		return pfx.Err(fmt.Errorf("%s is not recognized", which))
	}

	return nil
}

// Each worker has to maintain its own BGEN since it is not safe for concurrent
// reads
func Worker(workerID int, path string, offset <-chan int64, output chan<- AlleleCounter) {
	b, err := bgen.Open(path)
	if err != nil {
		log.Printf("Worker %d exited:\n", err)
	}
	defer b.Close()
	vr := b.NewVariantReader()

	for {
		select {
		case incoming, ok := <-offset:
			if !ok {
				// Incoming work channel is closed, we're done.
				return
			}
			variant := vr.ReadAt(incoming)
			if vr.Error() != nil {
				log.Fatalln(err)
			}

			// Only unphased for now
			if variant.Probabilities.Phased {
				continue
			}

			// Only biallelic variants for now
			if variant.NAlleles > 2 {
				continue
			}

			names := make(map[int]string)
			m := make(map[int]float64)
			ac := &AlleleCounter{}

			for id, v := range variant.Alleles {
				names[id] = v.String()
				m[id] = 0.0
			}

			for _, prob := range variant.Probabilities.SampleProbabilities {
				// log.Println(prob)
				for i, p := range prob.Probabilities {
					if i == 0 {
						m[0] += 2 * p
					} else if i == 1 {
						m[0] += p
						m[1] += p
					} else {
						m[1] += 2 * p
					}
				}
			}
			for i, name := range names {
				ac.Add(name, m[i])
			}

			output <- *ac
		}
	}
}
