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
	lineMatches [][]string
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
	flag.Parse()

	if *filename == "" {
		fmt.Println("no file provided")
		return
	}

	// Open file passed as argument
	file, err := os.Open(*filename)
	if err != nil {
		fmt.Println("error while reading file")
		return
	}
	defer file.Close()

	// Read file content line by line
	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)

	var (
		lines    []*string
		numLines int
	)

	for numLines = 0; scanner.Scan(); numLines++ {
		text := scanner.Text()
		lines = append(lines, &text)
	}

	var wg sync.WaitGroup
	result := Result{
		lineMatches: make([][]string, numLines),
	}

	for lineNum, line := range lines {
		wg.Add(1)

		go func(lineNum int, line string) {
			defer wg.Done()

			matches, err := getUrlMatches(line)
			if err != nil {
				return
			}

			for _, match := range matches {
				if !isUrlAlive(match) {
					result.AddMatch(lineNum, match)
				}
			}
		}(lineNum, *line)
	}

	wg.Wait()

	for line, matches := range result.lineMatches {
		for _, match := range matches {
			printColoredUrl(line, match)
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
