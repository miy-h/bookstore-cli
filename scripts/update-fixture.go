package main

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type commandConfig struct {
	fixturePath  string
	urlFunc      func(param string) string
	selectorFunc func(doc *goquery.Document) *goquery.Selection
}

func main() {
	if len(os.Args) < 3 {
		printUsage()
		os.Exit(1)
	}

	store := os.Args[1]
	subcommand := os.Args[2]
	args := os.Args[3:]

	var config commandConfig

	if store != "nauka" {
		printUsage()
		os.Exit(1)
	}

	switch subcommand {
	case "search":
		config = commandConfig{
			fixturePath: "fixtures/nauka/search.json",
			urlFunc: func(query string) string {
				return fmt.Sprintf("https://www.naukajapan.jp/?orderby=registered&desc=1&q_category1=&q_category2=&q=%s", url.QueryEscape(query))
			},
			selectorFunc: func(doc *goquery.Document) *goquery.Selection {
				return doc.Find("#main_pages_con")
			},
		}
	case "detail":
		config = commandConfig{
			fixturePath: "fixtures/nauka/detail.json",
			urlFunc: func(id string) string {
				return fmt.Sprintf("https://www.naukajapan.jp/detail.php?id=%s", url.QueryEscape(id))
			},
			selectorFunc: func(doc *goquery.Document) *goquery.Selection {
				sel := doc.Find("#main_detail_con")
				if sel.Length() == 0 {
					sel = doc.Find("#main_pages_con")
				}
				return sel
			},
		}
	default:
		printUsage()
		os.Exit(1)
	}

	fixtures := make(map[string]string)
	if data, err := os.ReadFile(config.fixturePath); err == nil && len(data) > 0 {
		_ = json.Unmarshal(data, &fixtures)
	}

	client := &http.Client{}

	for i, arg := range args {
		if i > 0 {
			time.Sleep(1 * time.Second)
		}
		targetURL := config.urlFunc(arg)
		req, err := http.NewRequest("GET", targetURL, nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create request for %q: %v\n", arg, err)
			os.Exit(1)
		}
		req.Header.Set("User-Agent", "bookstore-cli")

		resp, err := client.Do(req)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to fetch URL for %q: %v\n", arg, err)
			os.Exit(1)
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			fmt.Fprintf(os.Stderr, "HTTP request for %q returned status %d\n", arg, resp.StatusCode)
			os.Exit(1)
		}

		doc, err := goquery.NewDocumentFromReader(resp.Body)
		resp.Body.Close()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to parse HTML for %q: %v\n", arg, err)
			os.Exit(1)
		}

		sel := config.selectorFunc(doc)
		content, err := goquery.OuterHtml(sel)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to extract content for %q: %v\n", arg, err)
			os.Exit(1)
		}

		fixtures[arg] = fmt.Sprintf("<html><body>%s</body></html>", content)
	}

	if err := os.MkdirAll(filepath.Dir(config.fixturePath), 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create directory for fixture: %v\n", err)
		os.Exit(1)
	}

	jsonBytes, err := json.Marshal(fixtures, json.Deterministic(true), jsontext.WithIndent("    "))
	jsonBytes = append(jsonBytes, '\n')
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to encode fixtures JSON: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(config.fixturePath, jsonBytes, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write fixture file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully updated %s with %d entries.\n", config.fixturePath, len(args))
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "Usage:\n")
	fmt.Fprintf(os.Stderr, "  go run scripts/update-fixture.go nauka search \"query words\" [\"query words 2\" ...]\n")
	fmt.Fprintf(os.Stderr, "  go run scripts/update-fixture.go nauka detail \"storeID1\" [\"storeID2\" ...]\n")
}
