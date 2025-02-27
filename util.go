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
	for _, path := range [4]string{"pdfs", "sources", "previews", "pngs"} {
		if !pathExists(*options.RootDirectory + "/files/" + path) {
			err := os.MkdirAll(*options.RootDirectory+"/files/"+path, 0755)
			if err != nil {
				panic("Unable to create " + path + " directory: " + err.Error())
			}
		}
	}

	if options.CertDirectory != nil && !pathExists(*options.CertDirectory) {
		err := os.MkdirAll(*options.CertDirectory, 0755)
		if err != nil {
			panic("Unable to create cert directory: " + err.Error())
		}
	}
}
