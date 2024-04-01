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

var (
    // URL regex
    UrlRegexStr string = `\bhttps?://\S+`
    UrlRegex *regexp.Regexp = regexp.MustCompile(UrlRegexStr)

    // HTTP Client with timeout
    HttpClient *http.Client = &http.Client{
        Timeout: 10 * time.Second,
    }
)

func main() {
    filename := flag.String("f", "", "file path")
    flag.Parse()

    if *filename == "" {
        fmt.Println("No file provided")
        return
    }

    // Open file passed as argument
    file, err := os.Open(*filename)
    if err != nil {
        fmt.Println("Error while reading file")
        return
    }
    defer file.Close()

    // Read file content line by line
    scanner := bufio.NewScanner(file)
    scanner.Split(bufio.ScanLines)

    var lines []string
    for scanner.Scan() {
        matches, err := getUrlMatches(scanner.Text())
        if err != nil {
            continue
        }

        lines = append(lines, matches...)
    }

    var (
        results []string
        wg sync.WaitGroup
    )

    results = make([]string, len(lines))
    wg.Add(len(lines))

    for i, line := range lines {
        go checkUrlMatch(&wg, line, results, i)
    }

    wg.Wait()

    // Print results with corresponding line
    for i, result := range results {
        if result != "" {
            printColoredUrl(i, result)
        }
    }
}

// Get all regex matches.
// If there are no matches found, return an error.
func getUrlMatches(line string) ([]string, error) {
    matches := UrlRegex.FindAllString(line, -1)
    if len(matches) == 0 {
        return nil, errors.New("no matches")
    }
    return matches, nil
}

// Check if an URL match is alive.
// If URL is alive, saves entry within the corresponding line index.
func checkUrlMatch(wg *sync.WaitGroup, match string, results []string, index int) {
    defer wg.Done()

    if isUrlAlive(match) {
        results[index] = match
    }
}

// Verifies if an URL is working or not.
// Send head request to the url, and wait for 200 OK response.
func isUrlAlive(url string) bool {
    req, err := http.NewRequest("HEAD", url, nil)
    if err != nil {
        return false
    }

    res, err := HttpClient.Do(req)
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
