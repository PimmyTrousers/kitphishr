package main

import (
	"crypto/sha1"
	"errors"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/pimmytrousers/kitphishr/kitphishr"
)

func SaveResponse(outputDir string, resp kitphishr.Response) (string, error) {

	checksum := sha1.Sum(resp.Body)
	filename := fmt.Sprintf("%x_%s", checksum[:len(checksum)/2], path.Base(resp.URL))

	if strings.HasPrefix(filename, "da39a3ee5e6b4b0d3255") {
		return "", errors.New("0bytefile")
	}
	// create the output file
	out, err := os.Create(outputDir + "/" + filename)
	if err != nil {
		return filename, err
	}
	defer out.Close()

	// write the body to file
	out.Write(resp.Body)

	return filename, nil
}
