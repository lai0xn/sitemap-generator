package main

import (
	"encoding/xml"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/charmbracelet/log"
)

var maxLinks int64 // Maximum number of links to crawl

// URLSet represents the sitemap XML structure
type URLSet struct {
	XMLName xml.Name `xml:"urlset"`
	XMLNs   string   `xml:"xmlns,attr"`
	URLs    []URL    `xml:"url"`
}

// URL represents a URL entry in the sitemap
type URL struct {
	Loc string `xml:"loc"`
}

type Crawler struct {
	logger      log.Logger          // Logger for output
	baseUrl     string              // Base URL to start crawling from
	mu          sync.Mutex          // Mutex to synchronize access to shared data
	urlCount    int64               // Count of URLs crawled, using atomic operations
	seen        map[string]struct{} // Set of seen URLs to avoid duplicates
	wg          sync.WaitGroup      // WaitGroup to wait for all goroutines to finish
	stopOnce    sync.Once           // Ensures that the stop channel is closed only once
	stopChan    chan struct{}       // Channel to signal stopping of crawling
	sitemapPath string              // Path to the sitemap XML file
}

func NewCrawler(url, sitemapPath string) *Crawler {
	logger := log.NewWithOptions(os.Stderr, log.Options{
		ReportCaller:    true,
		ReportTimestamp: true,
		TimeFormat:      time.Kitchen,
		Prefix:          "Crawler 🕸️",
	})

	return &Crawler{
		logger:      *logger,
		baseUrl:     url,
		seen:        make(map[string]struct{}),
		stopChan:    make(chan struct{}),
		sitemapPath: sitemapPath,
	}
}

func (c *Crawler) ExtractLinks(url string) {
	defer c.wg.Done() // Decrement the WaitGroup counter when this function completes

	// Check if the maximum number of links has been reached
	if atomic.LoadInt64(&c.urlCount) >= maxLinks {
		c.stopOnce.Do(func() {
			close(c.stopChan) // Close stopChan only once to signal termination
		})
		return
	}

	// Create a new HTTP request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		c.logger.Warn(err) // Log the error
		return
	}

	// Perform the HTTP request
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		c.logger.Warn(err) // Log the error
		return
	}
	defer res.Body.Close() // Ensure the response body is closed after use

	// Check if the response status code is 200 OK
	if res.StatusCode != http.StatusOK {
		c.logger.Warn("Non-200 response status code:", res.StatusCode)
		return
	}

	// Parse the HTML document
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		c.logger.Warn(err) // Log the error
		return
	}

	// Find and process all anchor tags
	doc.Find("a").Each(func(i int, item *goquery.Selection) {
		href, exists := item.Attr("href") // Extract the href attribute
		if exists && strings.HasPrefix(href, c.baseUrl) {
			c.mu.Lock() // Lock the mutex to safely update shared data
			if _, seen := c.seen[href]; !seen {
				c.seen[href] = struct{}{} // Mark the URL as seen
				if atomic.AddInt64(&c.urlCount, 1) <= maxLinks {
					c.logger.Info("Link Found: ", href)
					c.wg.Add(1)             // Increment the WaitGroup counter
					go c.ExtractLinks(href) // Crawl the found link in a new goroutine
				}
			}
			c.mu.Unlock()
		}
	})
}

// writeSitemap writes all URLs to the sitemap XML file
func (c *Crawler) WriteSitemap() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Open the sitemap file for writing
	file, err := os.Create(c.sitemapPath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write the XML header
	_, err = file.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	if err != nil {
		return err
	}

	// Create URLSet for the sitemap
	urlSet := URLSet{
		XMLNs: "http://www.sitemaps.org/schemas/sitemap/0.9",
	}

	// Populate the URLSet with URLs from the seen map
	for url := range c.seen {
		urlSet.URLs = append(urlSet.URLs, URL{Loc: url})
	}

	// Encode the URLSet to XML
	encoder := xml.NewEncoder(file)
	encoder.Indent("", "  ")
	if err := encoder.Encode(urlSet); err != nil {
		return err
	}

	return nil
}

// Close cleans up resources
func (c *Crawler) Close() {
	// Optionally, you can perform cleanup tasks here
}

func main() {
	// Logger for argument parsing
	ArgsLogger := log.NewWithOptions(os.Stderr, log.Options{
		ReportCaller:    true,
		ReportTimestamp: true,
		TimeFormat:      time.Kitchen,
		Prefix:          "Args 🔑",
	})

	// Argument parsing
	outputPath := flag.String("o", "./sitemap.xml", "set the output file")
	flag.Int64Var(&maxLinks, "n", 100, "number of links to crawl")
	url := flag.String("t", "", "target url")

	ArgsLogger.Info("Parsing Args")

	flag.Parse()
	if *url == "" {
		ArgsLogger.Error("You must specify a target url")
		return
	}

	crwl := NewCrawler(*url, *outputPath)

	crwl.wg.Add(1)             // Add one to the WaitGroup for the initial goroutine
	go crwl.ExtractLinks(*url) // Start the crawling process

	// Wait for all goroutines to finish
	crwl.wg.Wait()

	// Write the sitemap XML file
	if err := crwl.WriteSitemap(); err != nil {
		ArgsLogger.Error("Failed to write sitemap:", err)
		return
	}

	// Log completion message
	fmt.Print("\n\n\n")
	log.Info("Scraping Completed")
}
