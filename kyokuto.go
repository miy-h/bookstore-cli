package main

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly/v2"
	"github.com/google/go-querystring/query"
)

type KyokutoSearchOptions struct {
	Keyword string `url:"q"`
	Title   string `url:"title"`
	Author  string `url:"author"`
	Isbn    string `url:"isbn"`
}

type KyokutoSearchResult struct {
	Title   string `json:"title"`
	Isbn    string `json:"isbn"`
	Price   int    `json:"price"`
	StoreID string `json:"storeId"`
}

type KyokutoBookDetailInfo struct {
	Title          string `json:"title"`
	Author         string `json:"author"`
	Publisher      string `json:"publisher"`
	Pages          int    `json:"pages"`
	PublishedMonth string `json:"publishedMonth"`
	Price          int    `json:"price"`
	Language       string `json:"language"`
	Isbn           string `json:"isbn"`
	StoreID        string `json:"storeId"`
}

func SearchKyokuto(options *KyokutoSearchOptions, client *http.Client) ([]*KyokutoSearchResult, error) {
	value, err := query.Values(options)
	if err != nil {
		return nil, fmt.Errorf("failed to encode search options: %w", err)
	}
	targetURL := fmt.Sprintf("https://www.kyokuto-bk.co.jp/books/?%s", value.Encode())

	c := colly.NewCollector(
		colly.UserAgent("bookstore-cli"),
	)
	if client != nil {
		c.SetClient(client)
	}

	result := make([]*KyokutoSearchResult, 0)
	var parseErr error
	c.OnHTML("ul.book-list > li:not(.bl-empty)", func(e *colly.HTMLElement) {
		book, err := parseKyokutoSearchBook(e.DOM)
		if err != nil {
			parseErr = err
			return
		}
		result = append(result, book)
	})

	var requestErr error
	c.OnError(func(r *colly.Response, err error) {
		requestErr = fmt.Errorf("HTTP request failed: %w (status: %d)", err, r.StatusCode)
	})
	if err := c.Visit(targetURL); err != nil {
		return nil, fmt.Errorf("failed to visit URL: %w", err)
	}
	if requestErr != nil {
		return nil, requestErr
	}
	if parseErr != nil {
		return nil, parseErr
	}
	if result == nil {
		return []*KyokutoSearchResult{}, nil
	}
	return result, nil
}

func FetchKyokutoDetail(storeId string, client *http.Client) (*KyokutoBookDetailInfo, error) {
	targetURL := fmt.Sprintf("https://www.kyokuto-bk.co.jp/books/%s/", storeId)

	c := colly.NewCollector(
		colly.UserAgent("bookstore-cli"),
	)
	if client != nil {
		c.SetClient(client)
	}

	var result *KyokutoBookDetailInfo
	var parseErr error
	c.OnHTML("body", func(e *colly.HTMLElement) {
		result, parseErr = parseKyokutoDetail(e.DOM)
	})

	var requestErr error
	c.OnError(func(r *colly.Response, err error) {
		requestErr = fmt.Errorf("HTTP request failed: %w (status: %d)", err, r.StatusCode)
	})
	if err := c.Visit(targetURL); err != nil {
		return nil, fmt.Errorf("failed to visit URL: %w", err)
	}
	if requestErr != nil {
		return nil, requestErr
	}
	if parseErr != nil {
		return nil, parseErr
	}
	if result == nil {
		return nil, fmt.Errorf("book detail not found: %s", storeId)
	}
	result.StoreID = storeId
	return result, nil
}

var kyokutoNumberRE = regexp.MustCompile(`[0-9][0-9,]*`)

func kyokutoNumber(raw string) int {
	match := kyokutoNumberRE.FindString(raw)
	if match == "" {
		return 0
	}
	n, _ := strconv.Atoi(strings.ReplaceAll(match, ",", ""))
	return n
}

func kyokutoISBN(raw string) string {
	raw = strings.ReplaceAll(raw, "-", "")
	match := regexp.MustCompile(`\d{10,13}`).FindString(raw)
	return match
}

func kyokutoPrice(s *goquery.Selection) int {
	priceRE := regexp.MustCompile(`[￥¥]\s*([0-9,]+)`)
	var price int
	s.Find("*").AddBack().EachWithBreak(func(_ int, item *goquery.Selection) bool {
		text := strings.TrimSpace(item.Text())
		if match := priceRE.FindStringSubmatch(text); len(match) > 1 {
			price, _ = strconv.Atoi(strings.ReplaceAll(match[1], ",", ""))
			return false
		}
		return true
	})
	return price
}

func parseKyokutoSearchBook(s *goquery.Selection) (*KyokutoSearchResult, error) {
	book := &KyokutoSearchResult{}
	book.Title = strings.TrimSpace(s.Find(".bl-bib > a").First().Text())
	if book.Title == "" {
		book.Title = strings.TrimSpace(s.Find("a.cover").First().AttrOr("title", ""))
	}
	book.Isbn = kyokutoISBN(s.Find(".bl-ed-info strong").First().Text())
	book.Price = kyokutoPrice(s.Find(".bl-ed-info").First())
	book.StoreID = strings.TrimSpace(s.AttrOr("data-rid", ""))
	if book.StoreID == "" {
		href := s.Find("a[href*='/books/']").First().AttrOr("href", "")
		if parsed, err := url.Parse(href); err == nil {
			parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
			if len(parts) >= 2 && parts[0] == "books" {
				book.StoreID = parts[1]
			}
		}
	}
	if book.StoreID == "" {
		return nil, fmt.Errorf("invalid Kyokuto book result: missing store ID")
	}
	return book, nil
}

func parseKyokutoDetail(s *goquery.Selection) (*KyokutoBookDetailInfo, error) {
	book := &KyokutoBookDetailInfo{}
	book.Title = strings.TrimSpace(s.Find(".bd-titles .en").First().Text())
	if book.Title == "" {
		book.Title = strings.TrimSpace(s.Find(".bd-titles").First().Text())
	}

	s.Find(".bd-table tr").Each(func(_ int, row *goquery.Selection) {
		label := strings.TrimSpace(row.Find("th").Text())
		value := strings.TrimSpace(row.Find("td").Text())
		switch label {
		case "著者・編者":
			book.Author = value
		case "出版社":
			book.Publisher = value
			if strings.HasPrefix(book.Publisher, "(") && strings.HasSuffix(book.Publisher, ")") {
				book.Publisher = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(book.Publisher, "("), ")"))
			}
		case "出版年月":
			book.PublishedMonth = strings.Replace(strings.TrimSpace(value), ".", "-", 1)
		case "ページ数":
			book.Pages = kyokutoNumber(value)
		case "言語":
			book.Language = strings.ToLower(value)
			if book.Language == "eng" {
				book.Language = "en"
			}
			if book.Language == "ger" {
				book.Language = "de"
			}
			if book.Language == "fre" {
				book.Language = "fr"
			}
			if book.Language == "jpn" {
				book.Language = "ja"
			}
		}
	})

	infoText := s.Find(".dtl-tabs").Text()
	book.Isbn = kyokutoISBN(infoText)
	book.Price = kyokutoPrice(s.Find(".dtl-tabs").First())
	if book.Isbn == "" {
		book.Isbn = kyokutoISBN(s.Text())
	}
	if book.Title == "" {
		return nil, fmt.Errorf("invalid Kyokuto book detail: missing title")
	}
	return book, nil
}

func FetchKyokutoDetailByIsbn(isbn string, client *http.Client) (*KyokutoBookDetailInfo, error) {
	searchResults, err := SearchKyokuto(&KyokutoSearchOptions{Isbn: strings.ReplaceAll(isbn, "-", "")}, client)
	if err != nil {
		return nil, err
	}

	if len(searchResults) == 0 {
		return nil, fmt.Errorf("Book not found: %s", isbn)
	}
	if len(searchResults) > 1 {
		var ids []string
		for _, result := range searchResults {
			ids = append(ids, result.StoreID)
		}
		return nil, fmt.Errorf("Multiple books found for the same ISBN: id=%s", strings.Join(ids, ", "))
	}

	detailInfo, err := FetchKyokutoDetail(searchResults[0].StoreID, client)

	if err != nil {
		return nil, err
	}
	return detailInfo, nil
}
