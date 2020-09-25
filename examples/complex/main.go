package main

import (
	"log"
	"time"

	"github.com/lukasjarosch/go-docx"
)

func main() {
	startTime := time.Now()

	replaceMap := docx.PlaceholderMap{
		"verwalter.name":                   "Verwalter",
		"verwalter.strasse":                "Strasse",
		"verwalter.hausnummer":             "111",
		"verwalter.plz":                    "8129",
		"verwalter.ort":                    "Irgendwo",
		"verwalter.telefon":                "123 123 123 123",
		"datum.heute": time.Now().Format("02.01.2006"),
	}

	doc, err := docx.Open("./examples/complex/template.docx")
	if err != nil {
	    panic(err)
	}

	log.Printf("open took: %s", time.Since(startTime))

	err = doc.ReplaceAll(replaceMap)
	if err != nil {
	    panic(err)
	}

	log.Printf("replace took: %s", time.Since(startTime))

	err = doc.WriteToFile("./examples/complex/replaced.docx")
	if err != nil {
		panic(err)
	}

	log.Printf("everything took: %s", time.Since(startTime))
}

