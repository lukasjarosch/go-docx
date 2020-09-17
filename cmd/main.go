package main

import (
	"errors"
	"log"
	"os"
	"time"

	"github.com/lukasjarosch/go-docx"
)

func main() {
	startTime := time.Now()

	fh, err := os.Open("./data/raw.docx")
	if err != nil {
		panic(err)
	}

	archive, err := docx.NewByteArchive(fh)
	if err != nil {
		panic(err)
	}

	doc, err := archive.Parse()
	if err != nil {
		panic(err)
	}

	exampleMapping := docx.PlaceholderMap{
		"key":                         "REPLACE some more",
		"key-with-dash":               "REPLACE",
		"key-with-dashes":             "REPLACE",
		"key with space":              "REPLACE",
		"key_with_underscore":         "REPLACE",
		"multiline":                   "REPLACE",
		"key.with.dots":               "REPLACE",
		"mixed-key.separator_styles#": "REPLACE",
		"yet-another_placeholder":     "REPLACE",
		"foo":                         "bar",
	}

	docBytes := replaceInDocument(doc.Plaintext(doc.DocumentBytes()), doc.DocumentBytes(), exampleMapping)
	headerBytes := make(map[string][]byte, len(doc.Headers()))
	for i, header := range doc.Headers() {
		headerBytes[i] = replaceInDocument(doc.Plaintext(header), header, exampleMapping)
	}
	footerBytes := make(map[string][]byte, len(doc.Footer()))
	for i, footer := range doc.Footer() {
		headerBytes[i] = replaceInDocument(doc.Plaintext(footer), footer, exampleMapping)
	}

	doc.SetDocumentContent(docBytes)
	doc.SetHeaders(headerBytes)
	doc.SetFooters(footerBytes)

	log.Printf("took: %s", time.Since(startTime))

	err = doc.WriteFile("/tmp/test.docx")
	if err != nil {
		panic(err)
	}
}

func replaceInDocument(plaintext string, docBytes []byte, placeholderMap docx.PlaceholderMap) []byte {
	// parse the document, extracting run and text positions
	parser := docx.NewRunParser(docBytes)
	err := parser.Execute()
	if err != nil {
		panic(err)
	}

	// find all placeholders which are present in the document
	foundPlaceholders := docx.FindDelimitedPlaceholders(plaintext, placeholderMap)
	log.Printf("%d placeholders need to be replaced in the document", len(foundPlaceholders))

	// use the parsed runs to find all placeholders
	placeholders := docx.ParsePlaceholders(parser.Runs(), docBytes)
	for _, placeholder := range placeholders {
		log.Printf("placeholder with %d fragments: %s", len(placeholder.Fragments), placeholder.Text(docBytes))
	}

	if len(placeholders) == len(foundPlaceholders) {
		log.Printf("parsed all %d placeholders in document", len(placeholders))
	} else {
		log.Printf("not all placeholders were parsed, have=%d, want=%d", len(placeholders), len(foundPlaceholders))
	}

	replacer := docx.NewReplacer(docBytes, placeholders)

	for key, value := range placeholderMap {
		err := replacer.Replace(key, value.(string))
		if err != nil {
			if errors.Is(err, docx.ErrPlaceholderNotFound) {
				continue
			} else {
				log.Fatalf("!! REPLACE FAILED: %s", err)
			}
		}
	}

	if replacer.ReplaceCount == len(foundPlaceholders) {
		log.Printf("replaced all %d placeholders in document", replacer.ReplaceCount)
	} else {
		log.Printf("not all placeholders have been replaced, the document may be corrupt (have=%d, want=%d)", replacer.ReplaceCount, len(foundPlaceholders))
	}

	return replacer.Bytes()
}
