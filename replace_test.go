package docx

import (
	"encoding/xml"
	"os"
	"testing"
)

func TestReplacer_Replace(t *testing.T) {
	replaceMap := PlaceholderMap{
		"key":                         "key",
		"key-with-dash":               "key-with-dash",
		"key-with-dashes":             "key-with-dashes",
		"key with space":              "key with space",
		"key_with_underscore":         "key_with_underscore",
		"multiline":                   "multiline",
		"key.with.dots":               "key.with.dots",
		"mixed-key.separator_styles#": "mixed-key.separator_styles#",
		"yet-another_placeholder":     "yet-another_placeholder",
		"foo":                         "foo",
		"nested":                      "nested",
	}

	doc, err := Open("./test/template.docx")
	if err != nil {
		t.Error(err)
		return
	}

	err = doc.ReplaceAll(replaceMap)
	if err != nil {
		t.Error("replacing failed", err)
		return
	}

	bs, err := os.ReadFile("./test/cameraman.jpg")
	if err != nil {
		t.Error(err)
		return
	}

	err = doc.SetFile("word/media/image1.jpg", bs)
	if err != nil {
		t.Error("replacing image failed", err)
		return
	}

	err = doc.WriteToFile("./test/out.docx")
	if err != nil {
		t.Error("unable to write", err)
		return
	}

	document, err := Open("./test/out.docx")
	if err != nil {
		t.Error("failed to open docx")
		return
	}

	documentXml := document.files[DocumentXml]

	err = xml.Unmarshal(documentXml, new(interface{}))
	if err != nil {
		t.Error("failed to unmarshal xml, replacing failed")
		return
	}

	// cleanup
	_ = os.Remove("./test/out.docx")
}
