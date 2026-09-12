package main

import (
	"regexp"
	"strings"
)

func IsValidIsbn(isbn string) bool {
	isbn = StripIsbnHyphens(isbn)
	if !regexp.MustCompile(`^\d{13}$`).MatchString(isbn) {
		return false
	}

	checksum := 0
	for i := range 13 {
		digit := int(isbn[i] - '0')
		if i%2 == 0 {
			checksum += digit
		} else {
			checksum += digit * 3
		}
	}

	return checksum%10 == 0
}

func StripIsbnHyphens(isbn string) string {
	return strings.ReplaceAll(isbn, "-", "")
}
