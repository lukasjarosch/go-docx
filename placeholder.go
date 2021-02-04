package docx

import (
	"fmt"
	"log"
	"regexp"
	"strings"
)

const (
	// OpenDelimiter defines the opening delimiter for the placeholders used inside a docx-document.
	OpenDelimiter rune = '{'
	// CloseDelimiter defines the closing delimiter for the placeholders used inside a docx-document.
	CloseDelimiter rune = '}'
)

var (
	// OpenDelimiterRegex is used to quickly match the opening delimiter and find it'str positions.
	OpenDelimiterRegex = regexp.MustCompile(string(OpenDelimiter))
	// CloseDelimiterRegex is used to quickly match the closing delimiter and find it'str positions.
	CloseDelimiterRegex = regexp.MustCompile(string(CloseDelimiter))
)

// PlaceholderMap is the type used to map the placeholder keys (without delimiters) to the replacement values
type PlaceholderMap map[string]interface{}

// Placeholder is the internal representation of a parsed placeholder from the docx-archive.
// A placeholder usually consists of multiple PlaceholderFragments which specify the relative
// byte-offsets of the fragment inside the underlying byte-data.
type Placeholder struct {
	Fragments []*PlaceholderFragment
}

// Text assembles the placeholder fragments using the given docBytes and returns the full placeholder literal.
func (p Placeholder) Text(docBytes []byte) string {
	str := ""
	for _, fragment := range p.Fragments {
		s := fragment.Run.Text.OpenTag.End
		t := docBytes[s+fragment.Position.Start : s+fragment.Position.End]
		str += string(t)
	}
	return str
}

// StartPos returns the absolute start position of the placeholder.
func (p Placeholder) StartPos() int64 {
	return p.Fragments[0].Run.Text.OpenTag.End + p.Fragments[0].Position.Start
}

// EndPos returns the absolute end position of the placeholder.
func (p Placeholder) EndPos() int64 {
	end := len(p.Fragments) - 1
	return p.Fragments[end].Run.Text.OpenTag.End + p.Fragments[end].Position.End
}

// Valid determines whether the placeholder can be used.
// A placeholder is considered valid, if all fragments are valid.
func (p Placeholder) Valid() bool {
	for _, fragment := range p.Fragments {
		if !fragment.Valid() {
			return false
		}
	}
	return true
}

// ParsePlaceholders will, given the document run positions and the bytes, parse out all placeholders including
// their fragments.
func ParsePlaceholders(runs DocumentRuns, docBytes []byte) (placeholders []*Placeholder, err error) {
	// tmp vars used to preserve state across iterations
	unclosedPlaceholder := new(Placeholder)
	hasOpenPlaceholder := false

	for _, run := range runs.WithText() {
		runText := run.GetText(docBytes)

		openDelimPositions := OpenDelimiterRegex.FindAllStringIndex(runText, -1)
		closeDelimPositions := CloseDelimiterRegex.FindAllStringIndex(runText, -1)

		// FindAllStringIndex returns a [][]int whereas the nested []int has only 2 keys (0 and 1)
		// We're only interested in the first key as that one indicates the position of the delimiter
		delimPositions := func(positions [][]int) []int {
			var pos []int
			for _, position := range positions {
				pos = append(pos, position[0])
			}
			return pos
		}

		// index all delimiters
		openPos := delimPositions(openDelimPositions)
		closePos := delimPositions(closeDelimPositions)


		// In case there are the same amount of open and close delimiters.
		// Here we will have three three different sub-cases.
		// Case 1 (default):
		//			'{foo}{bar}' which is the simplest case to handle
		//
		// Case 2 (special):
		//			'}foo{bar}foo{' which can easily be detected by checking if 'openPos > endPos'.
		//			That case can only be valid if there is an unclosed placeholder in a previous run.
		//			If there is no unclosed placeholder, then there is some form of user error (e.g. '{baz}}foo{bar}').
		//			We can also be sure that the first close and the last open delimiters are wrong, all the other ones
		//			in between will be correct, given the len(openPos)==len(closePos) premise.
		//			We're ignoring the case in which the user might've entered '}foo}bar{foo{' and went full derp-mode.
		//
		// Case 3 (nested):
		//			'{foo{bar}foo}' aka placeholder-nesting, which is acatually not going to be supported
		//			but needs to be detected and handled anyway. TODO handle nestings
		if (len(openPos) == len(closePos)) && len(openPos) != 0 {

			// isSpecialCase checks if, for all found delimiters, startPos > endPos is true (case 2)
			isSpecialCase := func() bool {
				for i := 0; i < len(openPos); i++ {
					start := openPos[i]
					end := closePos[i] + 1 // +1 is required to include the closing delimiter in the text
					if start > end {
						return true
					}
				}
				return false
			}

			// isNestedCase checks if, there are >1 OpenDelimiters before the first CloseDelimiter
			// if there is only 1 openPos, this cannot be true (we already know that it's not 0
			isNestedCase := func() bool {
				if len(openPos) == 1 {
					return false
				}
				if openPos[0] < closePos [0] &&
					openPos[1] < closePos[0] {
					return true
				}
				return false
			}

			// handle case 2
			if isSpecialCase() {

				// handle the easy part (everything between the the culprit first '}' and last '{' in the example of '}foo{bar}foo{'
				validOpenPos := openPos[:len(openPos)-1]
				validClosePos := closePos[1:]
				placeholders = append(placeholders, assembleFullPlaceholders(run, validOpenPos, validClosePos)...)

				// extract the first open and last close delimiter positions as they are the one causing issues.
				lastOpenPos := openPos[len(openPos)-1]
				firstClosePos := closePos[0]

				// we MUST be having an unclosedPlaceholder or the user made a typo like double-closing ('{foo}}{bar')
				if !hasOpenPlaceholder {
					return nil, fmt.Errorf("unexpected %c in run %d \"%s\"), missing preceeding %c", CloseDelimiter, run.ID, run.GetText(docBytes), OpenDelimiter)
				}

				// everything up to firstClosePos belongs to the currently open placeholder
				fragment := NewPlaceholderFragment(0, Position{0, int64(firstClosePos)+1}, run)
				unclosedPlaceholder.Fragments = append(unclosedPlaceholder.Fragments, fragment)
				placeholders = append(placeholders, unclosedPlaceholder)

				// a new, unclosed, placeholder starts at lastOpenPos
				fragment = NewPlaceholderFragment(0, Position{int64(lastOpenPos), int64(len(runText))}, run)
				unclosedPlaceholder = new(Placeholder)
				unclosedPlaceholder.Fragments = append(unclosedPlaceholder.Fragments, fragment)
				hasOpenPlaceholder = true

				continue
			}

			// there are multiple ways to handle this
			//	- error
			//	- cut out
			// 	- skip the run (that's what we do because we're lazy bums)
			if isNestedCase() {
				log.Printf("detected nested placeholder in run %d \"%s\", skipping \n", run.ID, run.GetText(docBytes))
				continue
			}

			// case 1, assemble and continue
			placeholders = append(placeholders, assembleFullPlaceholders(run, openPos, closePos)...)
			continue
		}

		// More open than closing delimiters, e.g. '{foo}{bar'
		// this can only mean that a placeholder is left unclosed after this run
		// For the length this means that (len(openPos) + 1) == len(closePos)
		// So we can be sure that the last position in openPos is the opening tag of the
		// unclosed placeholder.
		if len(openPos) > len(closePos) {
			// merge full placeholders in the run, leaving out the last openPos since
			// we know that the one is left over and must be handled separately below
			placeholders = append(placeholders, assembleFullPlaceholders(run, openPos[:len(openPos)-1], closePos)...)

			// add the unclosed part of the placeholder to a tmp placeholder var
			unclosedOpenPos := openPos[len(openPos)-1]
			fragment := NewPlaceholderFragment(0, Position{int64(unclosedOpenPos), int64(len(runText))}, run)
			unclosedPlaceholder.Fragments = append(unclosedPlaceholder.Fragments, fragment)
			hasOpenPlaceholder = true
			continue
		}

		// More closing than opening delimiters, e.g. '}{foo}'
		// this can only mean that there must be an unclosed placeholder which
		// is closed in this run.
		if len(openPos) < len(closePos) {
			// merge full placeholders in the run, leaving out the last closePos since
			// we know that the one is left over and must be handled separately below
			placeholders = append(placeholders, assembleFullPlaceholders(run, openPos, closePos[:len(closePos)-1])...)

			// there is only a closePos and no open pos
			if len(closePos) == 1 {
				fragment := NewPlaceholderFragment(0, Position{0, int64(int64(closePos[0]) + 1)}, run)
				unclosedPlaceholder.Fragments = append(unclosedPlaceholder.Fragments, fragment)
				placeholders = append(placeholders, unclosedPlaceholder)
				unclosedPlaceholder = new(Placeholder)
				hasOpenPlaceholder = false
				continue
			}
			continue
		}

		// No placeholders at all.
		// The run is only relevant if there is an unclosed placeholder from a previous run.
		// In that case it means that the full run-text belongs to the placeholder.
		// For example, if a placeholder has three fragments in total, this represents fragment 2 (see below)
		//	1) '{foo'
		//	2) 'bar-'
		//	3) '-baz}
		if len(openPos) == 0 && len(closePos) == 0 {
			if hasOpenPlaceholder {
				fragment := NewPlaceholderFragment(0, Position{0, int64(len(runText))}, run)
				unclosedPlaceholder.Fragments = append(unclosedPlaceholder.Fragments, fragment)
				continue
			}
		}
	}

	// Make sure that we're dealing with valid and proper placeholders only.
	// Everything else may cause issues like out of bounds errors or any other sort of weird things.
	// Here we will also assemble the final list of placeholders and return only the valid ones.
	var validPlaceholders []*Placeholder
	for _, placeholder := range placeholders {
		if !placeholder.Valid() {
			continue
		}

		// in order to catch false positives, ensure that all placeholders have BOTH delimiters
		text := placeholder.Text(docBytes)
		if !strings.ContainsRune(text, OpenDelimiter) ||
			!strings.ContainsRune(text, CloseDelimiter) {
			continue
		}

		// placeholder is valid
		validPlaceholders = append(validPlaceholders, placeholder)
	}
	return validPlaceholders, nil
}

// assembleFullPlaceholders will extract all complete placeholders inside the run given a open and close position.
// The open and close positions are the positions of the Delimiters which must already be known at this point.
// openPos and closePos are expected to be symmetrical (e.g. same length).
// Example: openPos := []int{10,20,30}; closePos := []int{13, 23, 33} resulting in 3 fragments (10,13),(20,23),(30,33)
// The n-th elements inside openPos and closePos must be matching delimiter positions.
func assembleFullPlaceholders(run *Run, openPos, closePos []int) (placeholders []*Placeholder) {
	for i := 0; i < len(openPos); i++ {
		start := openPos[i]
		end := closePos[i] + 1 // +1 is required to include the closing delimiter in the text
		fragment := NewPlaceholderFragment(0, Position{int64(start), int64(end)}, run)
		p := &Placeholder{Fragments: []*PlaceholderFragment{fragment}}
		placeholders = append(placeholders, p)
	}
	return placeholders
}

// AddPlaceholderDelimiter will wrap the given string with OpenDelimiter and CloseDelimiter.
// If the given string is already a delimited placeholder, it is returned unchanged.
func AddPlaceholderDelimiter(s string) string {
	if IsDelimitedPlaceholder(s) {
		return s
	}
	return fmt.Sprintf("%c%s%c", OpenDelimiter, s, CloseDelimiter)
}

// RemovePlaceholderDelimiter removes OpenDelimiter and CloseDelimiter from the given text.
// If the given text is not a delimited placeholder, it is returned unchanged.
func RemovePlaceholderDelimiter(s string) string {
	if !IsDelimitedPlaceholder(s) {
		return s
	}
	return strings.Trim(s, fmt.Sprintf("%s%s", string(OpenDelimiter), string(CloseDelimiter)))
}

// IsDelimitedPlaceholder returns true if the given string is a delimited placeholder.
// It checks whether the first and last rune in the string is the OpenDelimiter and CloseDelimiter respectively.
// If the string is empty, false is returned.
func IsDelimitedPlaceholder(s string) bool {
	if len(s) < 1 {
		return false
	}
	first := s[0]
	last := s[len(s)-1]
	if rune(first) == OpenDelimiter && rune(last) == CloseDelimiter {
		return true
	}
	return false
}
