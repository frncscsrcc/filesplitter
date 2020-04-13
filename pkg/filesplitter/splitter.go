package filesplitter

import (
	"crypto/md5"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"sync"
)

type part struct {
	order    int
	fileName string
	hash     string
}

type FileSplit struct {
	splitFolder string
	fileName    string
	workers     int
	size        int64
	blockSize   int64
	hash        string

	byteCounter  []int64
	blockFileIDs []int
	fileHandlers []*os.File

	parts []part
}

type NewFileSplit struct {
	FileName        string
	Workers         int
	BlockSize       int64
	SkipCheckDigest bool
	SplitFolder     string
}

func New(options NewFileSplit) (*FileSplit, error) {
	fsplit := &FileSplit{}

	fileName := options.FileName
	fsplit.fileName = fileName

	workers := 1
	if options.Workers > 1 {
		workers = options.Workers
	}

	fsplit.blockSize = 1024 * 512
	if options.BlockSize > 1 {
		fsplit.blockSize = options.BlockSize
	}

	fsplit.splitFolder = "./"
	if options.SplitFolder != "" {
		fsplit.splitFolder = options.SplitFolder
	}

	// Check if file exits
	fileInfo, err := os.Stat(fileName)
	if err != nil {
		return nil, errors.New("file " + fileName + " not found")
	}

	fsplit.size = fileInfo.Size()
	fsplit.workers = workers
	fsplit.fileHandlers = make([]*os.File, workers)
	fsplit.byteCounter = make([]int64, workers)
	fsplit.blockFileIDs = make([]int, workers)

	// Save the digest of the original file
	if !options.SkipCheckDigest {
		hash, err := checkDigest(fileName)
		if err != nil {
			return nil, err
		}
		fsplit.hash = hash
	}

	return fsplit, nil
}

func (fsplit *FileSplit) Split() error {
	workers := fsplit.workers
	bytesPerWorkesSection := int64(fsplit.size / int64(workers))

	pos := int64(0)
	for workerId := 0; workerId < workers; workerId++ {
		fh, err := openFile(fsplit.fileName)
		if err != nil {
			return err
		}
		fsplit.fileHandlers[workerId] = fh
		defer fsplit.fileHandlers[workerId].Close()

		// Add the offset readNextBlock
		fsplit.setOffset(workerId, pos)
		fsplit.byteCounter[workerId] = bytesPerWorkesSection
		pos += bytesPerWorkesSection
	}
	fsplit.byteCounter[workers-1] += int64(fsplit.size % int64(workers))

	var splitWG sync.WaitGroup
	splitWG.Add(workers)
	localParts := make([][]part, workers)
	for workedId := 0; workedId < workers; workedId++ {
		workerPartList := &(localParts[workedId])
		go fsplit.split(workedId, workerPartList, &splitWG)
	}
	splitWG.Wait()

	order := 0
	for workerId := 0; workerId < workers; workerId++ {
		for _, part := range localParts[workerId] {
			part.order = order
			fsplit.parts = append(fsplit.parts, part)
			order++
		}
	}

	// If the original file hash was saved, verified the split joining the content of all the parts
	if fsplit.hash != "" {
		if err := fsplit.verifySplit(); err != nil {
			return err
		}
	}

	fsplit.writeManifest()

	return nil
}

func (fsplit *FileSplit) split(processId int, workerPartList *[]part, wg *sync.WaitGroup) {
	defer wg.Done()

	*workerPartList = make([]part, 0)
	for true {
		byteSlice, nBytes, err := fsplit.readNextBlock(processId)
		if err != nil && err.Error() != "EOP" {
			break
		}
		fsplit.blockFileIDs[processId]++

		partFileName, hash, _ := fsplit.savePartialLocal(processId, fsplit.blockFileIDs[processId], byteSlice, nBytes)

		*workerPartList = append(*workerPartList, part{
			fileName: partFileName,
			hash:     hash,
		})

		if err != nil && err.Error() == "EOP" {
			break
		}
	}
}

func openFile(fileName string) (*os.File, error) {
	fh, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	return fh, err
}

func (fsplit *FileSplit) setOffset(workerId int, offset int64) error {
	whence := 0 // Set offset from the beginning of the file
	_, err := fsplit.fileHandlers[workerId].Seek(offset, whence)
	if err != nil {
		return err
	}
	return nil
}

func (fsplit *FileSplit) readNextBlock(workerId int) ([]byte, int, error) {
	var byteSlice []byte

	if fsplit.byteCounter[workerId] > fsplit.blockSize {
		byteSlice = make([]byte, fsplit.blockSize)
	} else {
		byteSlice = make([]byte, fsplit.byteCounter[workerId])
	}

	fsplit.byteCounter[workerId] -= int64(len(byteSlice))

	readBytes, err := fsplit.fileHandlers[workerId].Read(byteSlice)
	if fsplit.byteCounter[workerId] == 0 {
		err = errors.New("EOP")
	}

	return byteSlice, readBytes, err
}

func (fsplit *FileSplit) savePartialLocal(workerId int, blockId int, content []byte, nBytes int) (string, string, error) {
	digest := digest(&content)
	partFileName := fsplit.fileName +
		"." +
		strconv.Itoa(workerId) +
		"." +
		strconv.Itoa(blockId) +
		"." +
		digest +
		".part"

	file, err := os.OpenFile(
		fsplit.splitFolder+partFileName,
		os.O_WRONLY|os.O_TRUNC|os.O_CREATE,
		0666,
	)
	if err != nil {
		return "", "", err
	}
	defer file.Close()

	// Write bytes to file
	bytesWritten, err := file.Write(content)
	if err != nil {
		return "", "", err
	}
	if bytesWritten != nBytes {
		return "", "", errors.New("not all bytes was saved for " + partFileName)
	}

	return partFileName, digest, nil
}

func digest(bytes *[]byte) string {
	return fmt.Sprintf("%x", md5.Sum(*bytes))
}

func checkDigest(fileName string) (string, error) {
	contentPtr, err := readAll(fileName)
	if err != nil {
		return "", err
	}
	return digest(contentPtr), nil
}

func readAll(fileName string) (*[]byte, error) {
	file, err := os.Open(fileName)
	defer file.Close()
	if err != nil {
		return &[]byte{}, err
	}

	data, err := ioutil.ReadAll(file)
	if err != nil {
		return &[]byte{}, err
	}
	return &data, nil
}

func (fsplit *FileSplit) verifySplit() error {
	content := []byte{}
	for _, part := range fsplit.parts {
		bytesPtr, err := readAll(part.fileName)
		if err != nil {
			return err
		}
		content = append(content, *bytesPtr...)
	}
	if fsplit.hash != digest(&content) {
		return errors.New("split verification failed")
	}
	return nil
}
