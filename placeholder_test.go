package docx

import "testing"

var (
	textMapping = PlaceholderMap{
		"single":                  "Replaced text",
		"fragmented_placeholder":  "Replaced text",
		"yet-another-placeholder": "Replaced text",
		"some_placeholder":        "Replaced text",
		"foo_bar":                 "BAR BAZ",
	}
)

func TestParsePlaceholders(t *testing.T) {
	docBytes := readFile(t, "./test/placeholder.xml")
	expectedPlaceholderCount := 6

	parser := NewRunParser(docBytes)
	err := parser.Execute()
	if err != nil {
		t.Errorf("parser.Execute failed: %s", err)
	}

	placeholders, err := ParsePlaceholders(parser.Runs().WithText(), docBytes)
	if err != nil {
		t.Error(err)
		return
	}
	if len(placeholders) != expectedPlaceholderCount {
		t.Errorf("did not parse all placeholders, want=%d, have=%d", expectedPlaceholderCount, len(placeholders))
	}

	for key := range textMapping {
		expectedKey := AddPlaceholderDelimiter(key)

		valid := false
		for _, placeholder := range placeholders {
			if expectedKey == placeholder.Text(docBytes) {
				valid = true
				continue
			}
		}
		if !valid {
			t.Errorf("did not find expected placeholder %s", expectedKey)
		}
	}
}
