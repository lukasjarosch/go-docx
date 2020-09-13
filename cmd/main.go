package main

import (
	docx "dox-replacer-v2"
	"log"
	"os"
)

func main() {
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
		"key":                         "REPLACED",
		"key-with-dash":               "REPLACED",
		"key-with-dashes":             "REPLACED",
		"key with space":              "REPLACED",
		"key_with_underscore":         "REPLACED",
		"multiline":                   "REPLACED",
		"key.with.dots":               "REPLACED",
		"mixed-key.separator_styles#": "REPLACED",
		"yet-another_placeholder":     "REPLACED",
	}

	// find all placeholders which are present in the document
	foundPlaceholders := docx.FindDelimitedPlaceholders(doc.Plaintext(), exampleMapping)
	log.Printf("%d placeholders need to be replaced in the document", len(foundPlaceholders))

	parser := docx.NewRunParser(doc)
	err = parser.Execute()
	if err != nil {
		panic(err)
	}

	/*
		fragNum := -1
		isOpen := true
		curPlaceholder := &docx.Placeholder{}
	*/

	// is set true if an OpenDelimiter but no CloseDelimiter is found and only set to false once the CloseDelimiter is found
	//unclosedPlaceholder := false

	var placeholders []docx.Placeholder
	var unclosedPlaceholder docx.Placeholder
	hasOpenPlaceholder := false

	for _, run := range parser.Runs().WithText() {
		runText := run.GetText(doc.Bytes())

		openDelimPositions := docx.OpenDelimiterRegex.FindAllStringIndex(runText, -1)
		closeDelimPositions := docx.CloseDelimiterRegex.FindAllStringIndex(runText, -1)

		// FindAllStringIndex returns a [][]int whereas the nested []int has only 2 keys (0 and 1)
		// We're only interested in the first key as that one indicates the position of the delimiter
		delimPositions := func(positions [][]int) []int {
			var pos []int
			for _, position := range positions {
				pos = append(pos, position[0])
			}
			return pos
		}

		var openPos, closePos []int
		open := delimPositions(openDelimPositions)
		if open == nil {
			openPos = []int{}
		} else {
			openPos = open
		}
		c := delimPositions(closeDelimPositions)
		if c == nil {
			closePos = []int{}
		} else {
			closePos = c
		}

		// simple case: only full placeholders inside the run
		if (len(openPos) == len(closePos)) && len(openPos) != 0 {
			for i := 0; i < len(openPos); i++ {
				start := openPos[i]
				end := closePos[i] + 1 // +1 is required to include the closing delimiter in the text
				fragment := &docx.PlaceholderFragment{
					Position: docx.Position{
						Start: int64(start),
						End:   int64(end),
					},
					Number: 0,
					Run:    run,
				}
				p := docx.Placeholder{Fragments: []*docx.PlaceholderFragment{fragment}}
				placeholders = append(placeholders, p)
			}
			continue
		}

		// more open than closing delimiters
		// this can only mean that a placeholder is left unclosed after this run
		// For the length this must mean: (len(openPos) + 1) == len(closePos)
		// So we can be sure that the last position in openPos is the opening tag of the
		// unclosed placeholder.
		if len(openPos) > len(closePos) {
			unclosedOpenPos := openPos[len(openPos)-1]

			// merge full placeholders in the run
			for i := 0; i < (len(openPos) - 1); i++ {
				start := openPos[i]
				end := closePos[i] + 1 // +1 is required to include the closing delimiter in the text
				fragment := &docx.PlaceholderFragment{
					Position: docx.Position{
						Start: int64(start),
						End:   int64(end),
					},
					Number: 0,
					Run:    run,
				}
				p := docx.Placeholder{Fragments: []*docx.PlaceholderFragment{fragment}}
				placeholders = append(placeholders, p)
			}

			// add the unclosed part of the placeholder to a tmp placeholder var
			fragment := &docx.PlaceholderFragment{
				Position: docx.Position{
					Start: int64(unclosedOpenPos),
					End:   int64(len(runText)),
				},
				Number: 0,
				Run:    run,
			}
			unclosedPlaceholder.Fragments = append(unclosedPlaceholder.Fragments, fragment)
			hasOpenPlaceholder = true
			continue
		}

		// more closing than opening delimiters
		// this can only mean that there must be an unclosed placeholder which
		// is closed in this run.
		if len(openPos) < len(closePos) {
			// special case: there is only a closePos and no open pos
			if len(closePos) == 1 {
				fragment := &docx.PlaceholderFragment{
					Position: docx.Position{
						Start: 0,
						End:   int64(closePos[0])+1,
					},
					Number: len(unclosedPlaceholder.Fragments) + 1,
					Run:    run,
				}
				unclosedPlaceholder.Fragments = append(unclosedPlaceholder.Fragments, fragment)
				placeholders = append(placeholders, unclosedPlaceholder)
				unclosedPlaceholder = docx.Placeholder{}
				hasOpenPlaceholder = false
				continue
			}

			// merge full placeholders in the run
			for i := 0; i < (len(openPos) - 1); i++ {
				start := openPos[i]
				end := closePos[i] + 1 // +1 is required to include the closing delimiter in the text
				fragment := &docx.PlaceholderFragment{
					Position: docx.Position{
						Start: int64(start),
						End:   int64(end),
					},
					Number: 0,
					Run:    run,
				}
				p := docx.Placeholder{Fragments: []*docx.PlaceholderFragment{fragment}}
				placeholders = append(placeholders, p)
			}
			continue
		}

		// no placeholders at all. The run is only important if there
		// is an unclosed placeholder. That means that the full run belongs to the placeholder.
		if len(openPos) == 0 && len(closePos) == 0 {
			if hasOpenPlaceholder {
				fragment := &docx.PlaceholderFragment{
					Position: docx.Position{
						Start: 0,
						End:   int64(len(runText)),
					},
					Number: len(unclosedPlaceholder.Fragments) + 1,
					Run:    run,
				}
				unclosedPlaceholder.Fragments = append(unclosedPlaceholder.Fragments, fragment)
				continue
			}
		}

		// First, identify complete placeholders within the run.
		// The found placeholder literals will be removed from runText

		// Second, if an OpenDelimiter is found inside the remaining run literal,
		// we can be sure that the rest of the literal belongs to a placeholder.
		// We need to remember the fact that a placeholder tag is now open.

		// Now, until unclosedPlaceholder is set to true, we only need to look for
		// the ClosingDelimiter.
		// Once a ClosingDelimiter has been found, we know for sure that all text up
		// to that point belongs to the placeholder.

		/*
			openPos := -1
			tmpFragment := &docx.PlaceholderFragment{}
			for i, char := range runText {
				if char == docx.OpenDelimiter {
					log.Printf("open delim at: %d", i)
					isOpen = true
					openPos = i
					continue
				}
				if char == docx.CloseDelimiter {
					if !isOpen {
						log.Fatalf("syntax error: closing delimiter without previous opening delimiter")
					}
					log.Printf("c delim at: %d", i)
					isOpen = false
					fragNum += 1

					tmpFragment.Number = fragNum
					tmpFragment.Run = run
					tmpFragment.Position.Start = int64(openPos)
					tmpFragment.Position.End = int64(i)
					curPlaceholder.Fragments = append(curPlaceholder.Fragments, tmpFragment)
					continue
				}
			}
		*/
	}

	for _, placeholder := range placeholders {
		str := ""
		for _, fragment := range placeholder.Fragments {
			s := fragment.Run.Text.StartTag.End
			t := doc.Bytes()[s+fragment.Position.Start : s+fragment.Position.End]
			str += string(t)
		}
		log.Printf("placeholder with %d fragments: %s", len(placeholder.Fragments), str)
	}

	if len(placeholders) == len(foundPlaceholders) {
		log.Printf("parsed all %d placeholders in document", len(placeholders))
	} else {
		log.Printf("not all placeholders were parsed, have=%d, want=%d", len(placeholders), len(foundPlaceholders))
	}
}
