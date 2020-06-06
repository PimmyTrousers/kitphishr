package kitphishr

import (
	"bytes"
	"crypto/sha1"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

func SaveResponse(outputDir string, resp Response) (string, error) {

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

// New makes an http client that allows redirects, skips SSL warnings and sets timeouts
func New(timeout int) *http.Client {

	proxyURL := http.ProxyFromEnvironment

	var tr = &http.Transport{
		Proxy:             proxyURL,
		MaxConnsPerHost:   50,
		DisableKeepAlives: true,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
			Renegotiation:      tls.RenegotiateOnceAsClient,
		},
		DialContext: (&net.Dialer{
			Timeout:   time.Second * time.Duration(timeout),
			DualStack: true,
		}).DialContext,
	}

	client := &http.Client{
		Transport: tr,
		Timeout:   time.Second * time.Duration(timeout),
	}

	return client

}

/*
	parse the response to see if we've hit an open dir
	if we have, then look for hrefs that are zips
*/
func zipFromDir(resp Response) (string, error) {

	ziphref := ""

	// read body for hrefs
	data := bytes.NewReader(resp.Body)
	doc, err := goquery.NewDocumentFromReader(data)
	if err != nil {
		return ziphref, err
	}

	title := doc.Find("title").Text()

	if strings.Contains(title, "Index of /") {
		doc.Find("a").Each(func(i int, s *goquery.Selection) {
			if strings.Contains(s.Text(), ".zip") {
				ziphref = s.Text()
			}
		})
	}

	return ziphref, nil
}
