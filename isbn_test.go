package main

import (
	"testing"
)

func TestIsValidIsbn(t *testing.T) {
	testCases := map[string]bool{
		"9785986156750":     true,
		"978-5-98615-675-0": true,
		"9785986156751":     false,
		"978598615675":      false,
		"97859861567500":    false,
		"978598615675A":     false,
	}
	for isbn, expected := range testCases {
		if IsValidIsbn(isbn) != expected {
			t.Errorf("IsValidIsbn(%q) = %v, want %v", isbn, !expected, expected)
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
