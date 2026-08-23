package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
)

func printUsage() {
	fmt.Printf("Usage:\n  bookstore-cli [store_name] ...\n\nSupported store names:\n  nauka\n\nExample:\n  bookstore-cli nauka --params '{\"isbn\":[\"9785605096283\",\"9785947062588\"]}'\n")
}

func isValidIsbn(isbn string) bool {
	return regexp.MustCompile(`^\d{13}$`).MatchString(strings.ReplaceAll(isbn, "-", ""))
}

type ErrorObject struct {
	Error   bool   `json:"error"`
	Message string `json:"message"`
}

func createErrorObject(message string) ErrorObject {
	errorObject := ErrorObject{
		Error: true, Message: message}
	return errorObject
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

type NaukaParams struct {
	Isbn []string `json:"isbn"`
}

func parseNaukaParams(paramsJson string) (*NaukaParams, error) {
	dec := json.NewDecoder(strings.NewReader(paramsJson))
	dec.DisallowUnknownFields()
	var parsed NaukaParams
	err := dec.Decode(&parsed)
	if err != nil {
		return nil, err
	}
	if len(parsed.Isbn) == 0 {
		return nil, errors.New("ISBN list is empty")
	}
	return &parsed, nil
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
		paramsStr := naukaCmd.String("params", "", "JSON input")

		err := naukaCmd.Parse(os.Args[2:])
		if err != nil || *paramsStr == "" {
			exitWithError("--params flag is required")
		}

		params, err := parseNaukaParams(*paramsStr)
		if err != nil {
			exitWithError(fmt.Sprintf("Invalid params: %v", err))
		}

		for _, isbn := range params.Isbn {
			if !isValidIsbn(isbn) {
				exitWithError(fmt.Sprintf("Invalid ISBN: %s", isbn))
			}
		}

		result := make(map[string]any)

		for i, isbn := range params.Isbn {
			if i != 0 {
				time.Sleep(1 * time.Second)
			}

			detailInfo, err := FetchNaukaDetailByIsbn(strings.ReplaceAll(isbn, "-", ""), nil)
			if err != nil {
				result[isbn] = createErrorObject(err.Error())
			} else {
				result[isbn] = detailInfo
			}
		}

		jsonBytes, err := json.MarshalIndent(result, "", "  ")
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
