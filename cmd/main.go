package main

import (
	docx "dox-replacer-v2"
	"log"
	"os"
	"time"
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
	}

	// find all placeholders which are present in the document
	foundPlaceholders := docx.FindDelimitedPlaceholders(doc.Plaintext(), exampleMapping)
	log.Printf("%d placeholders need to be replaced in the document", len(foundPlaceholders))

	// parse the document, extracting run and text positions
	parser := docx.NewRunParser(doc)
	err = parser.Execute()
	if err != nil {
		panic(err)
	}

	docBytes := doc.Bytes()

	// use the parsed runs to find all placeholders
	placeholders := docx.ParsePlaceholders(parser.Runs(), doc.Bytes())
	for _, placeholder := range placeholders {
		log.Printf("placeholder with %d fragments: %s", len(placeholder.Fragments), placeholder.Text(doc.Bytes()))
	}


	if len(placeholders) == len(foundPlaceholders) {
		log.Printf("parsed all %d placeholders in document", len(placeholders))
	} else {
		log.Printf("not all placeholders were parsed, have=%d, want=%d", len(placeholders), len(foundPlaceholders))
	}

	replacer := docx.NewReplacer(docBytes, placeholders)

	for key, value := range exampleMapping {
		err := replacer.Replace(key, value.(string))
		if err != nil {
		    panic(err)
		}
	}

	if replacer.ReplaceCount == len(foundPlaceholders) {
		log.Printf("replaced all %d placeholders in document", replacer.ReplaceCount)
	} else {
		log.Printf("not all placeholders have been replaced, the document may be corrupt (have=%d, want=%d)", replacer.ReplaceCount, len(foundPlaceholders))
	}

	log.Printf("took: %s", time.Since(startTime))

	doc.SetDocumentContent(replacer.Bytes())
	err = doc.WriteFile("/tmp/test.docx")
	if err != nil {
		panic(err)
	}
}

