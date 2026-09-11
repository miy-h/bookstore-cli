package main

import (
	"testing"
)

func TestIsValidIsbn(t *testing.T) {
	validIsbnList := []string{"9785986156750", "978-5-98615-675-0"}
	for _, isbn := range validIsbnList {
		if !IsValidIsbn(isbn) {
			t.Errorf("IsValidIsbn failed: %s", isbn)
		}
	}
}

func TestStripIsbnHyphens(t *testing.T) {
	testCases := map[string]string{
		"978-5-98615-675-0": "9785986156750",
	}
	for isbn, expected := range testCases {
		if StripIsbnHyphens(isbn) != expected {
			t.Errorf("StripIsbnHyphens failed: %s", isbn)
		}
	}
}
