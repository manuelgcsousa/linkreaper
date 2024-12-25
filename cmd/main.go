package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"sync"
	"time"

	"github.com/fatih/color"
)

type Result struct {
	mut         sync.Mutex
	lineMatches map[int][]string
}

func (res *Result) AddMatch(lineNum int, match string) {
	res.mut.Lock()
	defer res.mut.Unlock()

	res.lineMatches[lineNum] = append(res.lineMatches[lineNum], match)
}

func (res *Result) GetMatches(lineNum int) []string {
	res.mut.Lock()
	defer res.mut.Unlock()

	return res.lineMatches[lineNum]
}

type Line struct {
	Number int
	Text   string
}

var (
	// URL regex
	urlRegexStr string         = `\bhttps?://\S+`
	urlRegex    *regexp.Regexp = regexp.MustCompile(urlRegexStr)

	// HTTP Client with timeout
	httpClient *http.Client = &http.Client{
		Transport: &http.Transport{DisableKeepAlives: true}, // do not leave connections open
		Timeout:   10 * time.Second,
	}
)

func main() {
	filename := flag.String("f", "", "file path")
	workers := flag.Int("w", 10, "number of concurrent workers")
	flag.Parse()

	if *filename == "" {
		fmt.Println("no file provided")
		return
	}

	if *workers <= 0 {
		fmt.Println("number of concurrent workers must be greater than 0")
		return
	}

	// Open file passed as argument
	file, err := os.Open(*filename)
	if err != nil {
		fmt.Println("error while reading file")
		return
	}
	defer file.Close()

	// Init:
	// wait group \
	// buffered channel to process lines \
	// result line matches
	var (
		wg     sync.WaitGroup
		lines  = make(chan Line, *workers)
		result = Result{lineMatches: make(map[int][]string)}
	)

	// Give work to the workers
	for i := 0; i < *workers; i++ {
		wg.Add(1)
		go processLines(lines, &result, &wg)
	}

	// Scan file and start sending lines to buffered channel
	scanner := bufio.NewScanner(file)
	for i := 0; scanner.Scan(); i++ {
		lines <- Line{Number: i, Text: scanner.Text()}
	}
	close(lines)

	wg.Wait()

	for line, matches := range result.lineMatches {
		for _, match := range matches {
			printColoredUrl(line, match)
		}
	}
}

func processLines(lines <-chan Line, result *Result, wg *sync.WaitGroup) {
	defer wg.Done()

	for line := range lines {
		matches, err := getUrlMatches(line.Text)
		if err != nil {
			continue
		}

		for _, match := range matches {
			if !isUrlAlive(match) {
				if err == nil {
					result.AddMatch(line.Number, match)
				}
			}
		}
	}
}

// Get all regex matches.
// If there are no matches found, return an error.
func getUrlMatches(line string) ([]string, error) {
	matches := urlRegex.FindAllString(line, -1)
	if len(matches) == 0 {
		return nil, errors.New("no URL matches")
	}
	return matches, nil
}

// Verifies if an URL is working or not.
// Send head request to the url, and wait for 200 OK response.
func isUrlAlive(url string) bool {
	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return false
	}

	res, err := httpClient.Do(req)
	if res != nil {
		defer res.Body.Close()
	}
	if err != nil {
		return false
	}

	return (res.StatusCode == http.StatusOK)
}

// Prints both line number and URL with color.
func printColoredUrl(lineCounter int, url string) {
	yellow := color.New(color.FgYellow).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()

	fmt.Printf("%s > %s\n", yellow(lineCounter), green(url))
}
