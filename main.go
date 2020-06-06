package main

import (
	"flag"
	"fmt"
	"os"
	"sync"

	"github.com/gookit/color"
	"github.com/pimmytrousers/kitphishr/kitphishr"
)

const MAX_DOWNLOAD_SIZE = 104857600 // 100MB

var verbose bool
var downloadKits bool
var concurrency int
var to int
var defaultOutputDir string

func main() {

	flag.IntVar(&concurrency, "c", 50, "set the concurrency level")
	flag.IntVar(&to, "t", 45, "set the connection timeout in seconds (useful to ensure the download of large files)")
	flag.BoolVar(&verbose, "v", false, "get more info on URL attempts")
	flag.BoolVar(&downloadKits, "d", false, "option to download suspected phishing kits")
	flag.StringVar(&defaultOutputDir, "o", "kits", "directory to save output files")

	flag.Parse()

	client := kitphishr.New(to)

	targets := make(chan string)

	// create the output directory, ready to save files to
	if downloadKits {
		err := os.MkdirAll(defaultOutputDir, os.ModePerm)
		if err != nil {
			fmt.Printf("There was an error creating the output directory : %s\n", err)
			os.Exit(1)
		}
	}

	// get input either from user or phishtank
	input, err := kitphishr.GetUserInput()
	if err != nil {
		fmt.Printf("There was an error getting URLS from PhishTank.\n")
		os.Exit(3)
	}

	// generate targets based on user input
	urls := kitphishr.GenerateTargets(input)

	urlWaitGroup := &sync.WaitGroup{}
	responses := kitphishr.GetURLs(urlWaitGroup, client, concurrency, urls)

	processWaitGroup := &sync.WaitGroup{}
	phishKits := kitphishr.ProcessURLs(processWaitGroup, client, concurrency, responses)

	// // send target urls to target channel
	// for url := range urls {
	// 	targets <- url
	// }

	// save group
	var sg sync.WaitGroup

	// give this a few threads to play with
	for i := 0; i < 10; i++ {

		sg.Add(1)

		go func() {
			defer sg.Done()
			for resp := range phishKits {
				fmt.Println("Writing")
				filename, err := SaveResponse(defaultOutputDir, resp)
				if err != nil {
					if verbose {
						color.Red.Printf("There was an error saving %s : %s\n", resp.URL, err)
					}
					continue
				} else if filename != "" {
					if verbose {
						color.Yellow.Printf("Successfully saved %s\n", filename)
					}
				}
			}
		}()
	}

	close(targets)
	urlWaitGroup.Wait()

	close(responses)
	processWaitGroup.Wait()

	close(phishKits)
	sg.Wait()

}
