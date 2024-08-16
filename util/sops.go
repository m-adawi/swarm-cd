package util

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/getsops/sops/v3/decrypt"
)

func DecryptFile(filepath string) (err error) {
	format := getFileFormat(filepath)
	textBytes, err := decrypt.File(filepath, format)
	if err != nil {
		return fmt.Errorf("could not decrypt the file %s: %w", filepath, err)
	}
	file, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("could not open the file %s: %w", filepath, err)
	}
	defer file.Close()
	_, err = file.Write(textBytes)
	if err != nil {
		return fmt.Errorf("could not write the file %s: %w", filepath, err)
	}
	return
}

func getFileFormat(filename string) string {
	extension := filepath.Ext(filename)
	if extension == ".yaml" || extension == ".yml" {
		return "yaml"
	} else if extension == ".json" {
		return "json"
	} else if extension == ".ini" {
		return "ini"
	} else {
		return "binary"
	}
}
