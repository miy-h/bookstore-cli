package main

import (
	"regexp"
	"strings"
)

func IsValidIsbn(isbn string) bool {
	return regexp.MustCompile(`^\d{13}$`).MatchString(strings.ReplaceAll(isbn, "-", ""))
}

func StripIsbnHyphens(isbn string) string {
	return strings.ReplaceAll(isbn, "-", "")
}
