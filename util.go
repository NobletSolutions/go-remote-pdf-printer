package main

import (
	"os"
	"strings"
)

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func fileNameWithoutExtension(fileName string) string {
	if pos := strings.LastIndexByte(fileName, '.'); pos != -1 {
		return fileName[:pos]
	}
	return fileName
}

func createDirectories(options *ServerOptions) {
	for _, path := range [3]string{"pdfs", "sources", "previews"} {
		if !pathExists(*options.RootDirectory + "/files/" + path) {
			err := os.MkdirAll(*options.RootDirectory+"/files/"+path, 0755)
			if err != nil {
				panic("Unable to create pdf directory: " + err.Error())
			}
		}
	}
}
