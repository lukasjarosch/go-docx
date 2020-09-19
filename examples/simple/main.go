package main

import (
	"log"
	"time"

	"github.com/lukasjarosch/go-docx"
)

func main() {
	startTime := time.Now()

	replaceMap := docx.PlaceholderMap{
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

	doc, err := docx.Open("template.docx")
	if err != nil {
	    panic(err)
	}

	log.Printf("open took: %s", time.Since(startTime))

	err = doc.ReplaceAll(replaceMap)
	if err != nil {
	    panic(err)
	}

	log.Printf("replace took: %s", time.Since(startTime))

	err = doc.WriteToFile("replaced.docx")
	if err != nil {
		panic(err)
	}
}

