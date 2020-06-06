package kitphishr

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"

	termutil "github.com/andrew-d/go-termutil"
	"github.com/gookit/color"
)

const (
	UA              = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_2) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/80.0.3987.149 Safari/537.36"
	maxDownloadSize = 104857600 // 100 MB
)

func GetURLs(wg *sync.WaitGroup, cli *http.Client, concurrency int, targets chan string) chan Response {
	// worker group to fetch the urls from targets channel
	// send the output to responses channel for further processing
	responses := make(chan Response)

	for i := 0; i < concurrency; i++ {

		wg.Add(1)

		go func() {

			defer wg.Done()

			for url := range targets {
				res, err := AttemptTarget(cli, url)
				if err != nil {
					continue
				}

				responses <- res
			}

		}()

	}

	return responses
}

func ProcessURLs(rg *sync.WaitGroup, cli *http.Client, concurrency int, responses chan Response) chan Response {
	// response group
	// determines if we've found a zip from a url folder
	// or if we've found an open directory and looks for a zip within
	tosave := make(chan Response)

	for i := 0; i < concurrency/2; i++ {

		rg.Add(1)

		go func() {

			defer rg.Done()

			for resp := range responses {

				if resp.StatusCode != http.StatusOK {
					continue
				}

				requrl := resp.URL

				// if we found a zip from a URL path
				if strings.HasSuffix(requrl, ".zip") {

					// make sure it's a valid zip
					if resp.ContentLength > 0 && resp.ContentLength < maxDownloadSize && strings.Contains(resp.ContentType, "zip") {
						fmt.Printf("Found valid zip: %s\n", requrl)
						tosave <- resp
					}
				}

				// check if we've found an open dir containing a zip
				// todo - walk an open dir for zips in other folders
				href, err := zipFromDir(resp)
				if err != nil {
					continue
				}
				if href != "" {
					hurl := ""
					if strings.HasSuffix(requrl, "/") {
						hurl = requrl + href
					} else {
						hurl = requrl + "/" + href
					}
					fmt.Printf("found open dir containing zip: %s\n", hurl)

					resp, err := AttemptTarget(cli, hurl)
					if err != nil {
						color.Red.Printf("There was an error downloading %s\n", hurl)
						continue
					}
					tosave <- resp
					continue

				}
			}
		}()
	}

	return tosave
}

/*
	get a list of urls either from the user
	piping into this program, or fetch the latest
	phishing urls from phishtank
*/
func GetUserInput() ([]PhishUrls, error) {

	var urls []PhishUrls

	// if nothing on stdin, default getting input from phishtank
	if termutil.Isatty(os.Stdin.Fd()) {

		pturls, err := getPhishTankURLs()
		if err != nil {
			return urls, err
		}
		urls = pturls

	} else {

		// if we do have stdin input, process that instead
		sc := bufio.NewScanner(os.Stdin)
		for sc.Scan() {
			urls = append(urls, PhishUrls{URL: sc.Text()})
		}
	}

	return urls, nil

}

/*
   iterate through the paths of each url to generate
   a target list...e.g.
     http://example.com/foo/bar
     http://example.com/foo/bar.zip
     http://example.com/foo/
     http://example.com/foo.zip
     http://example.com/
*/
func GenerateTargets(urls []PhishUrls) chan string {

	_urls := make(chan string, 1)

	go func() {

		seen := make(map[string]bool)

		for _, row := range urls {
			myurl := row.URL

			// parse the url
			u, err := url.Parse(myurl)
			if err != nil {
				continue
			}
			// split the paths from the parsed url
			paths := strings.Split(u.Path, "/")

			// iterate over the paths slice to traverse and send to urls channel
			for i := 0; i < len(paths); i++ {
				_path := paths[:len(paths)-i]
				tmpUrl := fmt.Sprintf(u.Scheme + "://" + u.Host + strings.Join(_path, "/"))

				// if we've seen the url already, keep moving
				if _, ok := seen[tmpUrl]; ok {
					continue
				}

				// add to seen
				seen[tmpUrl] = true

				// feed the _urls channels
				_urls <- tmpUrl

				// guess zip path and send to targets
				zipurl := tmpUrl + ".zip"

				// ignore http://example.com/.zip and http://example.com.zip
				if strings.HasSuffix(zipurl, "/.zip") || strings.Count(zipurl, "/") < 3 {
					continue
				}

				// add this one to seen too
				seen[zipurl] = true

				// feed the _urls channels
				_urls <- zipurl
			}
		}
		close(_urls)
	}()

	return _urls
}

// AttemptTarget peforms a GET against the target URL and returns a custo response type
func AttemptTarget(client *http.Client, url string) (Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return Response{}, err
	}

	req.Header.Set("User-Agent", UA)
	req.Header.Add("Connection", "close")
	req.Close = true

	httpresp, err := client.Do(req)
	if err != nil {
		return Response{}, err
	}

	defer httpresp.Body.Close()

	var resp Response

	resp.StatusCode = int64(httpresp.StatusCode)

	if respbody, err := ioutil.ReadAll(httpresp.Body); err == nil {
		resp.Body = respbody
	}

	resp.URL = url

	resp.ContentLength = httpresp.ContentLength
	resp.ContentType = httpresp.Header.Get("Content-Type")

	return resp, nil

}
