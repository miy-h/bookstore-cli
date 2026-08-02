package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"
)

func printUsage() {
	fmt.Printf("Usage:\n  bookstore-cli [store_name] ...\n\nSupported store names:\n  nauka\n\nExample:\n  bookstore-cli nauka --isbn 9785042420481\n")
}

func isValidIsbn(isbn string) bool {
	return regexp.MustCompile(`^\d{13}$`).MatchString(strings.ReplaceAll(isbn, "-", ""))
}

func exitWithError(message string) {
	error := struct {
		Error   bool   `json:"error"`
		Message string `json:"message"`
	}{true, message}
	jsonBytes, err := json.Marshal(error)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error formatting JSON output: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(string(jsonBytes))
	os.Exit(1)
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	storeName := os.Args[1]

	switch storeName {
	case "--help":
		printUsage()
	case "nauka":
		naukaCmd := flag.NewFlagSet("nauka", flag.ExitOnError)
		isbn := naukaCmd.String("isbn", "", "ISBN of the book")

		err := naukaCmd.Parse(os.Args[2:])
		if err != nil || *isbn == "" {
			exitWithError("--isbn flag is required")
		}

		if !isValidIsbn(*isbn) {
			exitWithError(fmt.Sprintf("Invalid ISBN: %s", *isbn))
		}

		naukaDomain := "https://www.naukajapan.jp"

		detailInfo, err := FetchNaukaDetailByIsbn(naukaDomain, strings.ReplaceAll(*isbn, "-", ""))

		if err != nil {
			exitWithError(err.Error())
		}

		jsonBytes, err := json.MarshalIndent(detailInfo, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting JSON output: %v\n", err)
			os.Exit(1)
		}

		fmt.Println(string(jsonBytes))

	default:
		fmt.Fprintf(os.Stderr, "Error: unsupported store '%s'\n", storeName)
		printUsage()
		os.Exit(1)
	}
}
