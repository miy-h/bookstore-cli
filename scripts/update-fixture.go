package main

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
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
	var params []fixtureParam

	switch subcommand {
	case "search":
		if store == "nauka" {
			config = commandConfig{
				fixturePath: "fixtures/nauka/search.json",
				urlFunc: func(query string) string {
					return fmt.Sprintf("https://www.naukajapan.jp/?orderby=registered&desc=1&q_category1=&q_category2=&q=%s", url.QueryEscape(query))
				},
				selectorFunc: func(doc *goquery.Document) *goquery.Selection {
					return doc.Find("#main_pages_con")
				},
			}
			for _, arg := range args {
				params = append(params, fixtureParam{key: arg, requestURL: config.urlFunc(arg)})
			}
		} else if store == "kyokuto" {
			config = commandConfig{
				fixturePath: "fixtures/kyokuto/search.json",
				selectorFunc: func(doc *goquery.Document) *goquery.Selection {
					return sanitizeSelection(doc.Find(".book-list").First())
				},
			}
			parsedParams, err := parseKyokutoParams(args)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
				os.Exit(1)
			}
			for _, parsed := range parsedParams {
				params = append(params, fixtureParam{
					key:        parsed.key,
					requestURL: "https://www.kyokuto-bk.co.jp/books/?" + parsed.query,
				})
			}
		} else {
			printUsage()
			os.Exit(1)
		}
	case "detail":
		if store == "kyokuto" {
			config = commandConfig{
				fixturePath: "fixtures/kyokuto/detail.json",
				urlFunc: func(id string) string {
					return "https://www.kyokuto-bk.co.jp/books/" + url.PathEscape(id)
				},
				selectorFunc: func(doc *goquery.Document) *goquery.Selection {
					return sanitizeSelection(doc.Find(".bd-section .container").First())
				},
			}
		} else if store == "nauka" {
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
		} else {
			printUsage()
			os.Exit(1)
		}
		params = make([]fixtureParam, 0, len(args))
		for _, arg := range args {
			params = append(params, fixtureParam{key: arg, requestURL: config.urlFunc(arg)})
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

	for i, param := range params {
		if i > 0 {
			time.Sleep(1 * time.Second)
		}
		targetURL := param.requestURL
		req, err := http.NewRequest("GET", targetURL, nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create request for %q: %v\n", param.key, err)
			os.Exit(1)
		}
		req.Header.Set("User-Agent", "bookstore-cli")

		resp, err := client.Do(req)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to fetch URL for %q: %v\n", param.key, err)
			os.Exit(1)
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			fmt.Fprintf(os.Stderr, "HTTP request for %q returned status %d\n", param.key, resp.StatusCode)
			os.Exit(1)
		}

		doc, err := goquery.NewDocumentFromReader(resp.Body)
		resp.Body.Close()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to parse HTML for %q: %v\n", param.key, err)
			os.Exit(1)
		}

		sel := config.selectorFunc(doc)
		content, err := goquery.OuterHtml(sel)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to extract content for %q: %v\n", param.key, err)
			os.Exit(1)
		}

		fixtures[param.key] = fmt.Sprintf("<html><body>%s</body></html>", content)
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

	fmt.Printf("Successfully updated %s with %d entries.\n", config.fixturePath, len(params))
}

type fixtureParam struct {
	key        string
	requestURL string
}

func sanitizeSelection(sel *goquery.Selection) *goquery.Selection {
	sel.Find("script, style").Remove()
	sel.Find("*").AddBack().Each(func(_ int, selection *goquery.Selection) {
		selection.RemoveAttr("style")
	})
	return sel
}

type kyokutoParam struct {
	key   string
	query string
}

func parseKyokutoParams(args []string) ([]kyokutoParam, error) {
	flags := flag.NewFlagSet("kyokuto search", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	paramsJSON := flags.String("params", "", "JSON input")
	isbn := flags.String("isbn", "", "ISBN alias for --params")
	if err := flags.Parse(args); err != nil {
		return nil, err
	}
	if len(flags.Args()) != 0 {
		return nil, fmt.Errorf("unexpected arguments: %s", strings.Join(flags.Args(), " "))
	}
	if *paramsJSON != "" && *isbn != "" {
		return nil, fmt.Errorf("--params and --isbn cannot be used together")
	}
	if *isbn != "" {
		*paramsJSON = `{"isbn":"` + strings.ReplaceAll(*isbn, `"`, `\"`) + `"}`
	}
	if *paramsJSON == "" {
		return nil, fmt.Errorf("--params or --isbn is required")
	}

	var params map[string]any
	if err := json.Unmarshal([]byte(*paramsJSON), &params); err != nil {
		return nil, fmt.Errorf("invalid --params JSON: %v", err)
	}
	if params == nil {
		return nil, fmt.Errorf("--params must be a JSON object")
	}

	query, err := queryString(params)
	if err != nil {
		return nil, fmt.Errorf("invalid --params: %v", err)
	}
	keyBytes, err := marshalSortedParams(params)
	if err != nil {
		return nil, fmt.Errorf("failed to encode params: %v", err)
	}
	return []kyokutoParam{{key: string(keyBytes), query: query}}, nil
}

func marshalSortedParams(params map[string]any) ([]byte, error) {
	keys := make([]string, 0, len(params))
	for key := range params {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	sortedParams := make(map[string]any)
	for _, key := range keys {
		sortedParams[key] = params[key]
	}
	bytes, err := json.Marshal(sortedParams, json.Deterministic(true))
	if err != nil {
		return nil, err
	}
	return []byte(bytes), nil
}

func queryString(params map[string]any) (string, error) {
	values := url.Values{}
	for key, raw := range params {
		switch value := raw.(type) {
		case nil:
			values.Add(key, "")
		case string:
			values.Add(key, value)
		case bool:
			values.Add(key, strconv.FormatBool(value))
		case float64:
			values.Add(key, strconv.FormatFloat(value, 'f', -1, 64))
		case []any:
			for _, item := range value {
				text, ok := item.(string)
				if !ok {
					return "", fmt.Errorf("parameter %q contains a non-string array value", key)
				}
				values.Add(key, text)
			}
		default:
			return "", fmt.Errorf("parameter %q has unsupported value", key)
		}
	}
	return values.Encode(), nil
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "Usage:\n")
	fmt.Fprintf(os.Stderr, "  go run scripts/update-fixture.go nauka search \"query words\" [\"query words 2\" ...]\n")
	fmt.Fprintf(os.Stderr, "  go run scripts/update-fixture.go nauka detail \"storeID1\" [\"storeID2\" ...]\n")
	fmt.Fprintf(os.Stderr, "  go run scripts/update-fixture.go kyokuto search --params '{\"isbn\":\"9798887196589\"}'\n")
	fmt.Fprintf(os.Stderr, "  go run scripts/update-fixture.go kyokuto search --isbn 9798887196589\n")
	fmt.Fprintf(os.Stderr, "  go run scripts/update-fixture.go kyokuto detail \"idstring\"\n")
}
