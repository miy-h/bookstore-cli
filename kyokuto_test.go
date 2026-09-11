package main

import (
	"encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"sort"
	"testing"

	"github.com/google/go-querystring/query"
)

func parseFixtureForKyokuto(path string) map[string]string {
	fixtures := make(map[string]string)
	if data, err := os.ReadFile(path); err == nil && len(data) > 0 {
		_ = json.Unmarshal(data, &fixtures)
	}
	return fixtures
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

func getMockedClientForKyokuto(t *testing.T) *http.Client {
	mux := http.NewServeMux()
	searchFixtures := parseFixtureForKyokuto("fixtures/kyokuto/search.json")
	detailFixtures := parseFixtureForKyokuto("fixtures/kyokuto/detail.json")

	mux.HandleFunc("GET www.kyokuto-bk.co.jp/books/", func(w http.ResponseWriter, r *http.Request) {
		params := make(map[string]any, len(r.URL.Query()))
		for key, values := range r.URL.Query() {
			if values[0] != "" {
				params[key] = values[0]
			}
		}

		paramsInJSON, err := marshalSortedParams(params)
		if err != nil {
			http.Error(w, "failed to encode request parameters", http.StatusInternalServerError)
			return
		}
		val, hasKey := searchFixtures[string(paramsInJSON)]
		if !hasKey {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(val))
	})

	mux.HandleFunc("GET www.kyokuto-bk.co.jp/books/{id}/", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		val, hasKey := detailFixtures[id]
		if !hasKey {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(val))
	})

	server := httptest.NewTestServer(t, mux)
	t.Cleanup(func() { server.Close() })
	return server.Client()
}

func TestSearchKyokuto(t *testing.T) {
	client := getMockedClientForKyokuto(t)
	testCases := map[KyokutoSearchOptions][]*KyokutoSearchResult{
		{Isbn: "9798887196589"}: {
			{
				Title:   "Dreams of Emancipation : A Transnational History of Revolutionary Russia.",
				Isbn:    "9798887196589",
				Price:   27881,
				StoreID: "1749432665-891410",
			},
		},
		{Isbn: "9782251458786"}: {
			{
				Title:   "Actes des empereurs carolingiens et ottoniens : une anthologie.",
				Isbn:    "9782251458786",
				Price:   6922,
				StoreID: "1788144867-f8c5ab",
			},
		},
		{Isbn: "9783406838088"}: {
			{
				Title:   "Woerterbuch der antiken Philosophie.  3., ueberarb. Aufl.",
				Isbn:    "9783406838088",
				Price:   6415,
				StoreID: "1751242301-115805",
			},
		},
		{Isbn: "9784910672786"}: {
			{
				Title:   "明治大正期帝国信用録：　第ＩＩＩ期第４回配本.  全４巻.",
				Isbn:    "9784910672786",
				Price:   143000,
				StoreID: "1781833816-350504",
			},
		},
		{Isbn: "9785986156750"}: {},
		// ISBN with hyphen
		{Isbn: "979-8-88719-658-9"}: {
			{
				Title:   "Dreams of Emancipation : A Transnational History of Revolutionary Russia.",
				Isbn:    "9798887196589",
				Price:   27881,
				StoreID: "1749432665-891410",
			},
		},
	}
	for options, expected := range testCases {
		result, err := SearchKyokuto(&options, client)
		if err != nil || !reflect.DeepEqual(result, expected) {
			q, err := query.Values(options)
			if err != nil {
				t.Errorf("book search failed")
			}
			t.Errorf("book search failed: %s", q.Encode())
		}
	}
}

func TestFetchKyokutoDetail(t *testing.T) {
	client := getMockedClientForKyokuto(t)

	testCases := map[string]KyokutoBookDetailInfo{
		"1749432665-891410": {
			Title:          "Dreams of Emancipation : A Transnational History of Revolutionary Russia.",
			Author:         "Naganawa, Norihiro (ed.),",
			Publisher:      "Academic Studies Pr., US",
			Pages:          290,
			PublishedMonth: "2025-04",
			Price:          27881,
			Language:       "en",
			Isbn:           "9798887196589",
			StoreID:        "1749432665-891410",
		},
		"1751242301-115805": {
			Title:          "Woerterbuch der antiken Philosophie.  3., ueberarb. Aufl.",
			Author:         "Horn, Christoph / Rapp, Christof (Hrsg.),",
			Publisher:      "Beck, GW",
			Pages:          528,
			PublishedMonth: "2027-01",
			Price:          6415,
			Language:       "de",
			Isbn:           "9783406838088",
			StoreID:        "1751242301-115805",
		},
		"1781833816-350504": {
			Title:          "明治大正期帝国信用録：　第ＩＩＩ期第４回配本.  全４巻.",
			Author:         "",
			Publisher:      "クロスカルチャー出版, JA",
			Pages:          2400,
			PublishedMonth: "2026",
			Price:          143000,
			Language:       "ja",
			Isbn:           "9784910672786",
			StoreID:        "1781833816-350504",
		},
		"1788144867-f8c5ab": {
			Title:          "Actes des empereurs carolingiens et ottoniens : une anthologie.",
			Author:         "Depreux, Philippe (textes introduits, traduits et commentés),",
			Publisher:      "Belles lettres, FR",
			Pages:          250,
			PublishedMonth: "2026-06",
			Price:          6922,
			Language:       "fr",
			Isbn:           "9782251458786",
			StoreID:        "1788144867-f8c5ab",
		},
	}
	for storeId, expected := range testCases {
		result, err := FetchKyokutoDetail(storeId, client)
		if err != nil || !reflect.DeepEqual(result, &expected) {
			t.Errorf("fetch book detail failed: %s", storeId)
		}
	}
}

func TestFetchKyokutoDetailByIsbn(t *testing.T) {
	client := getMockedClientForKyokuto(t)

	testCases := map[string]KyokutoBookDetailInfo{
		"9798887196589": {
			Title:          "Dreams of Emancipation : A Transnational History of Revolutionary Russia.",
			Author:         "Naganawa, Norihiro (ed.),",
			Publisher:      "Academic Studies Pr., US",
			Pages:          290,
			PublishedMonth: "2025-04",
			Price:          27881,
			Language:       "en",
			Isbn:           "9798887196589",
			StoreID:        "1749432665-891410",
		},
		"9782251458786": {
			Title:          "Actes des empereurs carolingiens et ottoniens : une anthologie.",
			Author:         "Depreux, Philippe (textes introduits, traduits et commentés),",
			Publisher:      "Belles lettres, FR",
			Pages:          250,
			PublishedMonth: "2026-06",
			Price:          6922,
			Language:       "fr",
			Isbn:           "9782251458786",
			StoreID:        "1788144867-f8c5ab",
		},
		"9783406838088": {
			Title:          "Woerterbuch der antiken Philosophie.  3., ueberarb. Aufl.",
			Author:         "Horn, Christoph / Rapp, Christof (Hrsg.),",
			Publisher:      "Beck, GW",
			Pages:          528,
			PublishedMonth: "2027-01",
			Price:          6415,
			Language:       "de",
			Isbn:           "9783406838088",
			StoreID:        "1751242301-115805",
		},
		"9784910672786": {
			Title:          "明治大正期帝国信用録：　第ＩＩＩ期第４回配本.  全４巻.",
			Author:         "",
			Publisher:      "クロスカルチャー出版, JA",
			Pages:          2400,
			PublishedMonth: "2026",
			Price:          143000,
			Language:       "ja",
			Isbn:           "9784910672786",
			StoreID:        "1781833816-350504",
		},
		// ISBN with hyphen
		"979-8-88719-658-9": {
			Title:          "Dreams of Emancipation : A Transnational History of Revolutionary Russia.",
			Author:         "Naganawa, Norihiro (ed.),",
			Publisher:      "Academic Studies Pr., US",
			Pages:          290,
			PublishedMonth: "2025-04",
			Price:          27881,
			Language:       "en",
			Isbn:           "9798887196589",
			StoreID:        "1749432665-891410",
		},
	}
	for isbn, expected := range testCases {
		result, err := FetchKyokutoDetailByIsbn(isbn, client)
		if err != nil || !reflect.DeepEqual(result, &expected) {
			t.Errorf("fetch book detail by ISBN failed: %s", isbn)
		}
	}
}
