package filesplitter

import(
	"encoding/json"
	"io/ioutil"
)

type ManifestPart struct {
	Hash     string `json:"hash"`
	FileName string `json:"fileName"`
	Order    int    `json:"order"`
}

type Manifest struct {
	OriginalFileName string         `json:"originalFileName"`
	OriginalFileHash string         `json:"originalFileHash"`
	BlockSize        int64          `json:"blockSize"`
	Parts            []ManifestPart `json:"parts"`
}

func (fsplit *FileSplit) GetManifest() Manifest {
	m := Manifest{
		OriginalFileName: fsplit.fileName,
		OriginalFileHash: fsplit.hash,
		BlockSize:        fsplit.blockSize,
		Parts:            make([]ManifestPart, len(fsplit.parts)),
	}
	for i, part := range fsplit.parts {
		m.Parts[i] = ManifestPart{
			Hash:     part.hash,
			FileName: part.fileName,
			Order:    part.order,
		}
	}
	return m
}

func (fsplit *FileSplit) writeManifest() error {
	manifest := fsplit.GetManifest()
	jsonContent, err := json.MarshalIndent(manifest, "", " ")
	if err != nil {
		return err
	}
	jsonByteSlice := []byte(jsonContent)
	hash := digest(&jsonByteSlice)
	manifestFileName := fsplit.splitFolder + fsplit.fileName + "." + hash + ".manifest.json"
	if err := ioutil.WriteFile(manifestFileName, jsonContent, 0644); err != nil {
		return err
	}
	return nil
}
