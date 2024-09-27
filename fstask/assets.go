package fstask

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

type Asset struct {
	RelativePath string
	Content      []byte
}

func readAssets(rootDirPath string) ([]Asset, error) {
	res := make([]Asset, 0)
	dirPath := filepath.Join(rootDirPath, "assets")
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		return res, nil
	}

	files, err := os.ReadDir(dirPath)
	if err != nil {
		return res, fmt.Errorf("error reading assets directory: %w", err)
	}

	for _, f := range files {
		if f.IsDir() {
			return nil, fmt.Errorf("directories are currently not supported")
		}
		bytes, err := os.ReadFile(filepath.Join(dirPath, f.Name()))
		if err != nil {
			return nil, fmt.Errorf("error reading asset: %w", err)
		}
		res = append(res, Asset{
			RelativePath: f.Name(),
			Content:      bytes,
		})
	}

	return res, nil
}

func readIllstrImgFnameFromPToml(pToml []byte) (string, error) {
	illustrationPath := ""
	tomlStruct := struct {
		IllstrImgFname string `toml:"illustration_image"`
	}{}

	err := toml.Unmarshal(pToml, &tomlStruct)
	if err != nil {
		log.Printf("Failed to unmarshal the task name: %v\n", err)
		return "", fmt.Errorf("failed to unmarshal the task name: %w", err)
	}

	illustrationPath = tomlStruct.IllstrImgFname
	return illustrationPath, nil
}
