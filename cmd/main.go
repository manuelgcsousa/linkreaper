package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	urlUtils "net/url"
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

	// Init program state
	state := initState()

	// Parse URL and check if it's valid
	// If so, build the base URI
	parseUrl, err := urlUtils.Parse(*url)
	if err != nil {
		fmt.Printf("Error while extracting base URI from '%s'\n", *url)
		os.Exit(1)
	} else {
		state.urlHost = fmt.Sprintf("%s://%s", parseUrl.Scheme, parseUrl.Host)
	}

	// Fetch provided URL
	res, err := state.httpClient.Get(*url)
	if err != nil {
		fmt.Printf("Error while fetching '%s'\n", *url)
		os.Exit(1)
	}

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

	// Iterate and process HTML tree nodes
	for node := range doc.Descendants() {
		if node.Type != html.ElementNode || node.DataAtom != atom.A {
			continue
		}

		state.wg.Add(1)
		go processAnchorNodeAttributes(node.Attr, &state)
	}

	state.wg.Wait()
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
		parsedUrl, err := urlUtils.Parse(url)
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
	res, err := httpClient.Head(url)
	if err == nil {
		return res.StatusCode == http.StatusOK, res.StatusCode
	}

	// If there is some response data, but an error occur --> attempt a GET request
	// Guardrail for some servers which do not allow HEAD method
	if res != nil {
		res, err = httpClient.Get(url)
		if err == nil {
			return false, res.StatusCode
		}

		defer res.Body.Close()
	}

	return false, -1
}
