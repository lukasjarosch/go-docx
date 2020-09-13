package docx

import (
	"fmt"
	"log"
	"regexp"
	"strings"
)

const (
	OpenDelimiter  rune = '{'
	CloseDelimiter rune = '}'
)

var (
	OpenDelimiterRegex = regexp.MustCompile(string(OpenDelimiter))
	CloseDelimiterRegex = regexp.MustCompile(string(CloseDelimiter))
)

// PlaceholderMap is the type used to map the placeholder keys (without delimiters) to the replacement values
type PlaceholderMap map[string]interface{}

type Placeholder struct {
	Fragments []*PlaceholderFragment
}

type PlaceholderFragment struct {
	Position Position // Position of the actual fragment within the run text. 0 == (Run.Text.StartTag.End + 1)
	Number   int      // numbering fragments for ease of use
	Run      *Run
}

func ParseRun(run *Run, runText string) []*Placeholder {

	find := func(text string) (*Placeholder,string) {
		openDelimPos := strings.IndexRune(text, OpenDelimiter)
		if openDelimPos > -1 {
			log.Printf("open delimiter at run %d: %s", openDelimPos, text)
		}
		closeDelimPos := strings.IndexRune(text, CloseDelimiter)
		if closeDelimPos > -1 {
			log.Printf("close delimiter at run %d: %s", closeDelimPos, text)
		}
		return nil, ""
	}
	_ = find

	return nil
}

// FindDelimitedPlaceholders will search for the keys of the mapping with the delimiters added.
// All found keys will be added to foundKeys on every occurrence.
func FindDelimitedPlaceholders(plaintext string, placeholderMap PlaceholderMap) []string {
	var foundKeys []string
	for key, _ := range placeholderMap {
		keyVal := AddPlaceholderDelimiter(key)
		if strings.Contains(plaintext, keyVal) {
			count := strings.Count(plaintext, keyVal)
			for i := 0; i < count; i++ {
				foundKeys = append(foundKeys, key)
			}
		}
	}
	return foundKeys
}

func AddPlaceholderDelimiter(s string) string {
	return fmt.Sprintf("%c%s%c", OpenDelimiter, s, CloseDelimiter)
}

func StripPlaceholderDelimiter(s string) string {
	return strings.Trim(s, fmt.Sprintf("%s%s", string(OpenDelimiter), string(CloseDelimiter)))
}
