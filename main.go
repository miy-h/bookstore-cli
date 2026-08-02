package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
)

func printUsage() {
	fmt.Fprintf(os.Stderr, "Usage:\n  bookstore-cli [store_name] ...\n\nSupported store names:\n  nauka\n\nExample:\n  bookstore-cli nauka --isbn 9785042420481\n")
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	storeName := os.Args[1]

	switch storeName {
	case "nauka":
		naukaCmd := flag.NewFlagSet("nauka", flag.ExitOnError)
		isbn := naukaCmd.String("isbn", "", "ISBN of the book")

		err := naukaCmd.Parse(os.Args[2:])
		if err != nil || *isbn == "" {
			fmt.Fprintf(os.Stderr, "Error: --isbn flag is required\n")
			printUsage()
			os.Exit(1)
		}

		result, err := SearchNauka("https://www.naukajapan.jp", strings.ReplaceAll(*isbn, "-", ""))
		if err != nil || len(result) != 1 {
			fmt.Fprintf(os.Stderr, "Error scraping book from Nauka Japan: %v\n", err)
			os.Exit(1)
		}

		jsonBytes, err := json.MarshalIndent(result[0], "", "  ")
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
