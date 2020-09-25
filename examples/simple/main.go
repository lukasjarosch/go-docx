package main

import (
	"flag"
	"log"
	"time"

	"github.com/lukasjarosch/go-docx"
)

var templatePath, outputPath string
func init() {
	flag.StringVar(&templatePath, "template", "template.docx", "path to the template docx file")
	flag.StringVar(&outputPath, "out", "replaced.docx", "path to the output docx")
}

func main() {
	startTime := time.Now()
	flag.Parse()

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

	doc, err := docx.Open(templatePath)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("open took: %s", time.Since(startTime))

	err = doc.ReplaceAll(replaceMap)
	if err != nil {
	    panic(err)
	}

	log.Printf("replace took: %s", time.Since(startTime))

	err = doc.WriteToFile(outputPath)
	if err != nil {
		panic(err)
	}

	log.Printf("everything took: %s", time.Since(startTime))
}

