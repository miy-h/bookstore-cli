package main

import (
	"fmt"
	"html"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly/v2"
)

type BookSearchResult struct {
	Title     string `json:"title"`
	Author    string `json:"author"`
	Publisher string `json:"publisher"`
	Place     string `json:"place"`
	Pages     int    `json:"pages"`
	Type      string `json:"type"`
	Year      int    `json:"year"`
	StoreID   string `json:"storeId"`
	Price     int    `json:"price"`
}

type BookDetailInfo struct {
	Title     string `json:"title"`
	Author    string `json:"author"`
	Publisher string `json:"publisher"`
	Place     string `json:"place"`
	Pages     int    `json:"pages"`
	Type      string `json:"type"`
	Year      int    `json:"year"`
	Price     int    `json:"price"`
	Isbn      string `json:"isbn"`
	StoreID   string `json:"storeId"`
}

func parsePlace(raw string) string {
	cleaned := strings.Trim(raw, " .,:\t\r\n")
	switch strings.ToUpper(cleaned) {
	case "М":
		return "Москва"
	case "СПБ":
		return "Санкт-Петербург"
	}
	return cleaned
}

func parseType(raw string) string {
	s := strings.ToLower(strings.TrimSpace(raw))
	if strings.Contains(s, "hard") {
		return "hard"
	}
	if strings.Contains(s, "pap") {
		return "paper"
	}
	return s
}

func parseYear(raw string) int {
	reYear := regexp.MustCompile(`(\d{4})\s*年`)
	if yMatch := reYear.FindStringSubmatch(raw); len(yMatch) > 1 {
		year, _ := strconv.Atoi(yMatch[1])
		return year
	}
	return 0
}

func parsePrice(raw string) int {
	rePrice := regexp.MustCompile(`\\([0-9,]+)`)
	if priceMatch := rePrice.FindStringSubmatch(raw); len(priceMatch) > 1 {
		price, _ := strconv.Atoi(strings.ReplaceAll(priceMatch[1], ",", ""))
		return price
	}
	return 0
}

type CommonBookInfo struct {
	Title     string
	Author    string
	Publisher string
	Place     string
	Pages     int
	Type      string
	Price     int
}

func parseCommonBookInfo(s *goquery.Selection) (*CommonBookInfo, error) {
	bookInfo := &CommonBookInfo{}
	bookInfo.Title = strings.TrimSpace(s.Find("h3").Text())

	h4Sel := s.Find("h4")
	if h4Sel.Length() > 0 {
		h4Html, _ := h4Sel.Html()
		reBr := regexp.MustCompile(`(?i)<br\s*/?>`)
		parts := reBr.Split(h4Html, -1)

		if len(parts) > 0 {
			// Author is first line before <br>
			authorDoc, err := goquery.NewDocumentFromReader(strings.NewReader(parts[0]))
			if err == nil {
				bookInfo.Author = strings.TrimSpace(authorDoc.Text())
			} else {
				bookInfo.Author = strings.TrimSpace(parts[0])
			}
		}

		if len(parts) > 1 {
			infoStr := html.UnescapeString(parts[1])

			// Extract Pages and Type location
			rePages := regexp.MustCompile(`(\d+)\s*[cс]\.`)
			loc := rePages.FindStringIndex(infoStr)

			var prefix string
			if loc != nil {
				// Pages
				if pMatch := rePages.FindStringSubmatch(infoStr[loc[0]:loc[1]]); len(pMatch) > 1 {
					bookInfo.Pages, _ = strconv.Atoi(pMatch[1])
				}
				// Type
				typeStr := strings.TrimSpace(infoStr[loc[1]:])
				bookInfo.Type = parseType(typeStr)

				prefix = infoStr[:loc[0]]
			} else {
				prefix = infoStr
			}

			// Extract Place and Publisher from prefix
			prefix = strings.TrimSpace(prefix)

			var rawPlace, rest string
			if idx := strings.Index(prefix, "<"); idx != -1 {
				rawPlace = prefix[:idx]
				rest = prefix[idx:]
			} else if idx := strings.IndexAny(prefix, ",:"); idx != -1 {
				rawPlace = prefix[:idx]
				rest = prefix[idx+1:]
			} else {
				fields := strings.Fields(prefix)
				if len(fields) > 0 {
					rawPlace = fields[0]
					if len(fields) > 1 {
						rest = strings.Join(fields[1:], " ")
					}
				} else {
					rawPlace = prefix
				}
			}

			bookInfo.Place = parsePlace(rawPlace)

			// Clean Publisher
			pub := strings.TrimSpace(rest)
			if strings.HasPrefix(pub, "<") && strings.HasSuffix(pub, ">") {
				pub = strings.Trim(pub, "<> ")
			} else {
				pub = strings.Trim(pub, " .,:\t\r\n")
				if strings.HasPrefix(pub, "<") && strings.Contains(pub, ">") {
					endIdx := strings.Index(pub, ">")
					pub = pub[1:endIdx]
				}
			}
			bookInfo.Publisher = pub
		}
	}

	bookInfo.Price = parsePrice(s.Find("h5").Text())

	return bookInfo, nil
}

func parseBookFromSearchPage(s *goquery.Selection) (*BookSearchResult, error) {
	book := &BookSearchResult{}

	commonInfo, err := parseCommonBookInfo(s)
	if err != nil {
		return nil, err
	}

	book.Title = commonInfo.Title
	book.Author = commonInfo.Author
	book.Publisher = commonInfo.Publisher
	book.Place = commonInfo.Place
	book.Pages = commonInfo.Pages
	book.Type = commonInfo.Type
	book.Price = commonInfo.Price

	href, exists := s.Find("h3 a").Attr("href")
	if !exists {
		href, _ = s.Find("h2 a").Attr("href")
	}
	storeUrl, err := url.Parse(href)
	if err != nil {
		return nil, err
	}
	id, exist := storeUrl.Query()["id"]
	if !exist || len(id) != 1 {
		return nil, fmt.Errorf("invalid store link: %s", href)
	}
	book.StoreID = id[0]

	book.Year = parseYear(s.Find(".buy_selection_sec .left").Text())

	return book, nil
}

func SearchNauka(host string, query string) ([]*BookSearchResult, error) {
	targetURL := fmt.Sprintf("%s/?orderby=&desc=&q_category1=&q_category2=&q=%s", host, url.QueryEscape(query))

	c := colly.NewCollector(
		colly.UserAgent("bookstore-cli"),
	)

	var result []*BookSearchResult
	var parseErr error

	c.OnHTML("#main_pages_con li", func(e *colly.HTMLElement) {
		res, err := parseBookFromSearchPage(e.DOM)
		if err != nil {
			parseErr = err
			return
		}
		result = append(result, res)
	})

	var reqErr error
	c.OnError(func(r *colly.Response, err error) {
		reqErr = fmt.Errorf("HTTP request failed: %w (status: %d)", err, r.StatusCode)
	})

	err := c.Visit(targetURL)
	if err != nil {
		return nil, fmt.Errorf("failed to visit URL: %w", err)
	}

	if reqErr != nil {
		return nil, reqErr
	}

	if parseErr != nil {
		return nil, parseErr
	}

	return result, nil
}

func parseBookFromDetailPage(s *goquery.Selection) (*BookDetailInfo, error) {
	book := &BookDetailInfo{}

	commonInfo, err := parseCommonBookInfo(s)
	if err != nil {
		return nil, err
	}

	book.Title = commonInfo.Title
	book.Author = commonInfo.Author
	book.Publisher = commonInfo.Publisher
	book.Place = commonInfo.Place
	book.Pages = commonInfo.Pages
	book.Type = commonInfo.Type
	book.Price = commonInfo.Price

	text := s.Find("h4 + p").Text()

	book.Year = parseYear(text)

	reIsbn := regexp.MustCompile(`(?i)ISBN[\s ]+(\d+)`)
	if isbnMatch := reIsbn.FindStringSubmatch(text); len(isbnMatch) > 0 {
		book.Isbn = isbnMatch[1]
	}

	return book, nil
}

func FetchNaukaDetail(host string, storeId string) (*BookDetailInfo, error) {
	targetURL := fmt.Sprintf("%s/detail.php?id=%s", host, storeId)

	c := colly.NewCollector(
		colly.UserAgent("bookstore-cli"),
	)

	var result *BookDetailInfo
	var parseErr error

	c.OnHTML("#main_detail_con li", func(e *colly.HTMLElement) {
		res, err := parseBookFromDetailPage(e.DOM)
		if err != nil {
			parseErr = err
			return
		}
		result = res
	})

	var reqErr error
	c.OnError(func(r *colly.Response, err error) {
		reqErr = fmt.Errorf("HTTP request failed: %w (status: %d)", err, r.StatusCode)
	})

	err := c.Visit(targetURL)
	if err != nil {
		return nil, fmt.Errorf("failed to visit URL: %w", err)
	}

	if reqErr != nil {
		return nil, reqErr
	}

	if parseErr != nil {
		return nil, parseErr
	}

	result.StoreID = storeId

	return result, nil
}
