package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	urlutils "net/url"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

type State struct {
	httpClient *http.Client
	wg         sync.WaitGroup
	urlHost    string
}

func initState() State {
	return State{
		httpClient: &http.Client{
			Transport: &http.Transport{DisableKeepAlives: true}, // don't leave connections open
			Timeout:   10 * time.Second,
		},
		urlHost: "",
	}
}

func main() {
	url := flag.String("url", "", "url to fetch")
	flag.Parse()

	if *url == "" {
		fmt.Println("No URL provided. Exiting...")
		os.Exit(1)
	}

	// Init state and fetch base URL
	state := initState()
	state.urlHost = extractBaseUrl(*url)

	// Fetch HTML document
	htmlDoc := fetchHtmlDocument(*url, state.httpClient)

	// Process links and wait until all work is done
	processLinks(htmlDoc, &state)
	state.wg.Wait()
}

func extractBaseUrl(rawUrl string) string {
	// Parse URL and check if it's valid
	// If so, build the base URI
	parsed, err := urlutils.Parse(rawUrl)
	if err != nil {
		fmt.Printf("Error while extracting base URI from '%s'\n", rawUrl)
		os.Exit(1)
	}

	return fmt.Sprintf("%s://%s", parsed.Scheme, parsed.Host)
}

func fetchHtmlDocument(url string, client *http.Client) *html.Node {
	// Fetch provided URL
	res, err := client.Get(url)
	if err != nil {
		fmt.Printf("Error while fetching '%s'\n", url)
		os.Exit(1)
	}
	defer res.Body.Close()

	// Extract data from HTTP response
	bytes, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Println("Error while reading response body data")
		os.Exit(1)
	}

	// Get the parse tree from the extracted HTML data
	doc, err := html.Parse(strings.NewReader(string(bytes)))
	if err != nil {
		fmt.Println("Error while parsing HTML tree from response body data")
		os.Exit(1)
	}

	return doc
}

func processLinks(doc *html.Node, state *State) {
	// Iterate and process HTML tree nodes
	for node := range doc.Descendants() {
		if node.Type != html.ElementNode || node.DataAtom != atom.A {
			continue
		}

		state.wg.Add(1)
		go processAnchorNodeAttributes(node.Attr, state)
	}
}

func processAnchorNodeAttributes(attributes []html.Attribute, state *State) {
	defer state.wg.Done()

	for _, val := range attributes {
		// Skip if no 'href' attribute
		if val.Key != "href" || val.Val == "" {
			continue
		}

		url := val.Val

		// Parse extracted URL
		//
		// If there isn't any scheme or host, it's probably a relative link,
		// so we join with the host URL.
		parsedUrl, err := urlutils.Parse(url)
		if err != nil {
			continue
		} else {
			if parsedUrl.Scheme == "" && parsedUrl.Host == "" {
				url = fmt.Sprintf("%s%s", state.urlHost, url)
			}
		}

		if status, code := isUrlAlive(url, state.httpClient); status {
			fmt.Printf("['%s' => %d OK]\n", url, code)
		} else {
			fmt.Printf("['%s' => %d RIP]\n", url, code)
		}
	}
}

func isUrlAlive(url string, httpClient *http.Client) (bool, int) {
	methods := []func(string) (*http.Response, error){
		httpClient.Head,
		httpClient.Get,
	}

	for _, method := range methods {
		res, err := method(url)
		if res != nil {
			defer res.Body.Close()
		}
		if err == nil {
			return res.StatusCode == http.StatusOK, res.StatusCode
		}
	}

	return false, -1
}
