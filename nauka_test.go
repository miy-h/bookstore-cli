package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"
)

func ParseFixture(path string) map[string]string {
	fixtures := make(map[string]string)
	if data, err := os.ReadFile(path); err == nil && len(data) > 0 {
		_ = json.Unmarshal(data, &fixtures)
	}
	return fixtures
}

func StartTestServer() *httptest.Server {
	mux := http.NewServeMux()
	detailFixtures := ParseFixture("fixtures/nauka/detail.json")
	searchFixtures := ParseFixture("fixtures/nauka/search.json")

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		queryWord := r.URL.Query().Get("q")
		val, hasKey := searchFixtures[queryWord]
		if hasKey {
			w.Header().Add("Content-Type", "text/html")
			w.WriteHeader(200)
			w.Write([]byte(val))
		} else {
			w.WriteHeader(404)
			w.Write([]byte{})
		}
	})

	mux.HandleFunc("/detail.php", func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("id")
		val, hasKey := detailFixtures[id]
		if hasKey {
			w.WriteHeader(200)
			w.Write([]byte(val))
		} else {
			w.WriteHeader(404)
			w.Write([]byte{})
		}
	})

	return httptest.NewServer(mux)
}

func TestSearchNaukaIsbn(t *testing.T) {
	server := StartTestServer()
	defer server.Close()

	testCases := map[string]BookSearchResult{
		// Moscow, hardback
		"9785002223046": {
			Title:     "Мы на Западе и на Востоке. Культурно-исторические основы русской государственности. (Сила мысли)",
			Author:    "Иванов В.Н.",
			Publisher: "Родина",
			Place:     "Москва",
			Pages:     336,
			Type:      "hard",
			Year:      2024,
			StoreID:   "256516",
			Price:     5280,
		},
		// Saint Petersburg, paperback
		"9785002561001": {
			Title:     "Документы Свирлага из фонда колонии \"Свирьстрой\". 1930-1937 гг.: По материалам фонда Р-2557 Ленинградского областного государственного архива гор. Выборга.",
			Author:    "Муравьева М.В.",
			Publisher: "Реноме",
			Place:     "Санкт-Петербург",
			Pages:     168,
			Type:      "paper",
			Year:      2025,
			StoreID:   "273925",
			Price:     13860,
		},
		// the publisher doesn't include `<>`
		"9785605096283": {
			Title:     "Культуры и цивилизации Центральной Азии от неолита до средневековья. Материалы международной научной конференции, посвященной 120-летию А. М. Беленицкого, 95-летию В. М. Массона・・・",
			Author:    "Никоноров В.П., Стоянов Е.О. (ed.)",
			Publisher: "Ин-т истории материальной культуры РАН",
			Place:     "Санкт-Петербург",
			Pages:     419,
			Type:      "paper",
			Year:      2024,
			StoreID:   "269203",
			Price:     27940,
		},
		// the place of publication contains space
		"9785947062588": {
			Title:     "От Александра III до Горбачева. Жизнь композитора Александра Касьянова. (Нижегородские были)",
			Author:    "Колесников В.С.",
			Publisher: "Книги",
			Place:     "Нижний Новгород",
			Pages:     336,
			Type:      "hard",
			Year:      2022,
			StoreID:   "247513",
			Price:     10010,
		},
		// the author is empty, the publisher contains semicolon, the place of publication contains hyphens
		"9785986156750": {
			Title:     "Историки Ростовского университета./ 3-е изд., испр. и доп.",
			Author:    "",
			Publisher: "Мини Тайп; ЮФУ",
			Place:     "Ростов-на-Дону",
			Pages:     347,
			Type:      "hard",
			Year:      2025,
			StoreID:   "276081",
			Price:     15400,
		},
		// the place of publication is irregular
		"9785449936851": {
			Title:     "Даша севастопольская. Первая сестра милосердия. (Люди. Судьбы. Эпохи)",
			Author:    "Лукашевич К.В.",
			Publisher: "Директмедиа Паблишинг",
			Place:     "М.-Берлин",
			Pages:     44,
			Type:      "paper",
			Year:      2023,
			StoreID:   "247047",
			Price:     3520,
		},
	}

	for isbn, expected := range testCases {
		result, err := SearchNauka(server.URL, isbn)
		if err != nil || len(result) != 1 || !reflect.DeepEqual(result[0], &expected) {
			t.Errorf("search by ISBN failed: %s", isbn)
		}
	}
}

func TestFetchNaukaDetail(t *testing.T) {
	server := StartTestServer()
	defer server.Close()

	testCases := map[string]BookDetailInfo{
		// Moscow, hardback
		"256516": {
			Title:     "Мы на Западе и на Востоке. Культурно-исторические основы русской государственности. (Сила мысли)",
			Author:    "Иванов В.Н.",
			Publisher: "Родина",
			Place:     "Москва",
			Pages:     336,
			Type:      "hard",
			Year:      2024,
			Isbn:      "9785002223046",
			Price:     5280,
			StoreID:   "256516",
		},
		// Saint Petersburg, paperback
		"273925": {
			Title:     "Документы Свирлага из фонда колонии \"Свирьстрой\". 1930-1937 гг.: По материалам фонда Р-2557 Ленинградского областного государственного архива гор. Выборга.",
			Author:    "Муравьева М.В.",
			Publisher: "Реноме",
			Place:     "Санкт-Петербург",
			Pages:     168,
			Type:      "paper",
			Year:      2025,
			Isbn:      "9785002561001",
			Price:     13860,
			StoreID:   "273925",
		},
		// the publisher doesn't include `<>`
		"269203": {
			Title:     "Культуры и цивилизации Центральной Азии от неолита до средневековья. Материалы международной научной конференции, посвященной 120-летию А. М. Беленицкого, 95-летию В. М. Массона и 100-летию Ю. А. Заднепровского, Санкт-Петербург, 5-8 ноября 2024 г.",
			Author:    "Никоноров В.П., Стоянов Е.О. (ed.)",
			Publisher: "Ин-т истории материальной культуры РАН",
			Place:     "Санкт-Петербург",
			Pages:     419,
			Type:      "paper",
			Year:      2024,
			Isbn:      "9785605096283",
			Price:     27940,
			StoreID:   "269203",
		},
		// the place of publication contains space
		"247513": {
			Title:     "От Александра III до Горбачева. Жизнь композитора Александра Касьянова. (Нижегородские были)",
			Author:    "Колесников В.С.",
			Publisher: "Книги",
			Place:     "Нижний Новгород",
			Pages:     336,
			Type:      "hard",
			Year:      2022,
			Price:     10010,
			Isbn:      "9785947062588",
			StoreID:   "247513",
		},
		// the author is empty, the publisher contains semicolon, the place of publication contains hyphens
		"276081": {
			Title:     "Историки Ростовского университета./ 3-е изд., испр. и доп.",
			Author:    "",
			Publisher: "Мини Тайп; ЮФУ",
			Place:     "Ростов-на-Дону",
			Pages:     347,
			Type:      "hard",
			Year:      2025,
			Price:     15400,
			Isbn:      "9785986156750",
			StoreID:   "276081",
		},
		// the place of publication is irregular
		"247047": {
			Title:     "Даша севастопольская. Первая сестра милосердия. (Люди. Судьбы. Эпохи)",
			Author:    "Лукашевич К.В.",
			Publisher: "Директмедиа Паблишинг",
			Place:     "М.-Берлин",
			Pages:     44,
			Type:      "paper",
			Year:      2023,
			Isbn:      "9785449936851",
			Price:     3520,
			StoreID:   "247047",
		},
	}

	for storeId, expected := range testCases {
		result, err := FetchNaukaDetail(server.URL, storeId)
		if err != nil || !reflect.DeepEqual(result, &expected) {
			t.Errorf("fetch book detail failed: %s", storeId)
		}
	}
}
