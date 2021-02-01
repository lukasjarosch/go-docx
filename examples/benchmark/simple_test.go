package benchmark

import (
	"flag"
	"github.com/lukasjarosch/go-docx"
	"testing"
)

var templatePath, outputPath string

func init() {
	flag.StringVar(&templatePath, "template", "simple.docx", "path to the template docx file")
	flag.StringVar(&outputPath, "out", "simple_replaced.docx", "path to the output docx")
}

func Simple() {
	replaceMap := docx.PlaceholderMap{
		"1": "1",
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

func BenchmarkSimple(b *testing.B) {
	for n := 1; n < b.N; n++ {
		Simple()
	}
}