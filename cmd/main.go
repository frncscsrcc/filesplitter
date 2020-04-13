package main

import (
	"github.com/frncscsrcc/filesplitter/pkg/filesplitter"
	"flag"
	"fmt"
)

func main() {

	fileName := flag.String("file", "", "File to split")
	blockSize := flag.Int64("blockSize", 1024*500, "Block size")
	workers := flag.Int("workers", 1, "Parallel workers")
	noCheck := flag.Bool("noCheck", false, "Skip digest verification")
	splitFolder := flag.String("folder", "./", "Folder for splitted files")

	flag.Parse()

	if *fileName == "" {
		panic("missing filename")
	}

	fsplit, err := filesplitter.New(filesplitter.NewFileSplit{
		FileName:        *fileName,
		BlockSize:       *blockSize,
		Workers:         *workers,
		SkipCheckDigest: *noCheck,
		SplitFolder:     *splitFolder,
	})
	if err != nil {
		panic(err)
	}

	if err := fsplit.Split(); err != nil {
		panic(err)
	}

	manifest := fsplit.GetManifest()
	fmt.Printf("%s\n", manifest.OriginalFileName)	
	for _, part := range manifest.Parts {
		fmt.Printf("  %s\n", part.FileName)
	}

}
