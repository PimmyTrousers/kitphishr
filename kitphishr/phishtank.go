package kitphishr

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

func getPhishTankURLs() ([]PhishUrls, error) {

	pturl := "http://data.phishtank.com/data/online-valid.json"

	// if the user has their own phishtank api key, use it
	apiKey := os.Getenv("PT_API_KEY")
	if apiKey != "" {
		pturl = fmt.Sprintf("http://data.phishtank.com/data/%s/online-valid.json", apiKey)
	}

	resp, err := http.Get(pturl)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	var urls []PhishUrls
	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	respByte := buf.Bytes()
	if err := json.Unmarshal(respByte, &urls); err != nil {
		return nil, err
	}
	return urls, nil
}
