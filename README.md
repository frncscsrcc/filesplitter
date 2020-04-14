[![Go Report Card](https://goreportcard.com/badge/github.com/frncscsrcc/filesplitter)](https://goreportcard.com/report/github.com/frncscsrcc/filesplitter)

File Splitter
===

Splits a file in several parts, compares the hash digests and produces a manifest file with the fragment details.

From command line:
```
Usage of bin/fileSplitter_linux64:
  -blockSize int
    	Block size (default 512000)
  -file string
    	File to split
  -folder string
    	Folder for splitted files (default "./")
  -noCheck
    	Skip digest verification
  -workers int
    	Parallel workers (default 1)

```
