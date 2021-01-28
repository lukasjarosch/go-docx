package main

import (
	"flag"
	"time"

	"github.com/lukasjarosch/go-docx"
)

var templatePath, outputPath string

func init() {
	flag.StringVar(&templatePath, "template", "template.docx", "path to the template docx file")
	flag.StringVar(&outputPath, "out", "replaced.docx", "path to the output docx")
}

func main() {
	flag.Parse()

	replaceMap := docx.PlaceholderMap{
		"verwalter.name":       "Verwalter",
		"verwalter.strasse":    "Strasse",
		"verwalter.hausnummer": "111",
		"verwalter.plz":        "8129",
		"verwalter.ort":        "Irgendwo",
		"verwalter.telefon":    "123 123 123 123",
		"datum.heute":          time.Now().Format("02.01.2006"),
		"vermietung.brutto":    "CHF 123245.00",
		"vermietung.netto":     "CHF 2342.00",
	}

	doc, err := docx.Open(templatePath)
	if err != nil {
		panic(err)
	}

	err = doc.ReplaceAll(replaceMap)
	if err != nil {
		panic(err)
	}

	err = doc.WriteToFile(outputPath)
	if err != nil {
		panic(err)
	}
}
