package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

func getPdfInfo(pdfFile string) (map[string]string, error) {
	info := make(map[string]string)

	cmd := exec.Command("/usr/bin/pdfinfo", pdfFile)
	output, err := cmd.Output()
	if err != nil {
		return nil, errors.New("unable to get pdf information")
	}

	myString := string(output)
	lines := strings.Split(myString, "\n")
	m1 := regexp.MustCompile(`/(^:)|(:$)/`)

	for _, line := range lines {
		cleanLine := m1.ReplaceAllString(line, "")
		cols := strings.Split(cleanLine, ":")
		if len(cols) == 2 {
			info[strings.ToLower(strings.Replace(cols[0], " ", "_", -1))] = strings.TrimSpace(cols[1])
		}
	}

	return info, nil
}

func createPreviews(pdfFile string, outputDir string) (string, error) {
	baseName := fileNameWithoutExtension(filepath.Base(pdfFile))
	var cmdArgs []string
	cmdArgs = append(cmdArgs, "-jpeg")
	cmdArgs = append(cmdArgs, "-scale-to")
	cmdArgs = append(cmdArgs, "1024")
	cmdArgs = append(cmdArgs, pdfFile)
	cmdArgs = append(cmdArgs, filepath.Join(outputDir, baseName))
	cmd := exec.Command("/usr/bin/pdftocairo", cmdArgs...)
	if err := cmd.Run(); err != nil {
		return "", errors.New("unable to produce pdf image previews")
	}

	return baseName, nil
}

func combinePdfs(inputFiles []string, options *ServerOptions) (*os.File, error) {
	// Merge the PDF files
	combinedFile, err := os.CreateTemp(*options.RootDirectory+"/files/pdfs/", "*-combined.pdf")
	if err != nil {
		return nil, errors.New("unable to create combined pdf output files")
	}

	cmdArgs := inputFiles
	cmdArgs = append(cmdArgs, combinedFile.Name())
	cmd := exec.Command("/usr/bin/pdfunite", cmdArgs...)
	if err := cmd.Run(); err != nil {
		return nil, errors.New("unable to combine pdfs")
	}

	return combinedFile, nil
}
