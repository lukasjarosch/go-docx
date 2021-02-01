package docx

import (
	"fmt"
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
	CloseDelimiterRegex  = regexp.MustCompile(string(CloseDelimiter))
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
		s := fragment.Run.Text.StartTag.End
		t := docBytes[s+fragment.Position.Start : s+fragment.Position.End]
		str += string(t)
	}
	return str
}

// StartPos returns the absolute start position of the placeholder.
func (p Placeholder) StartPos() int64 {
	return p.Fragments[0].Run.Text.StartTag.End + p.Fragments[0].Position.Start
}

// EndPos returns the absolute end position of the placeholder.
func (p Placeholder) EndPos() int64 {
	end := len(p.Fragments) - 1
	return p.Fragments[end].Run.Text.StartTag.End + p.Fragments[end].Position.End
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

		// simple case: only full placeholders inside the run
		if (len(openPos) == len(closePos)) && len(openPos) != 0 {
			placeholders = append(placeholders, assembleFullPlaceholders(run, openPos, closePos)...)
			continue
		}

		// more open than closing delimiters
		// this can only mean that a placeholder is left unclosed after this run
		// For the length this must mean: (len(openPos) + 1) == len(closePos)
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

		// more closing than opening delimiters
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

		// no placeholders at all. The run is only important if there
		// is an unclosed placeholder. That means that the full run belongs to the placeholder.
		if len(openPos) == 0 && len(closePos) == 0 {
			if hasOpenPlaceholder {
				fragment := NewPlaceholderFragment(0, Position{0, int64(len(runText))}, run)
				unclosedPlaceholder.Fragments = append(unclosedPlaceholder.Fragments, fragment)
				continue
			}
		}
	}

	// in order to catch false positives, ensure that all placeholders have BOTH delimiters
	// if a placeholder only has one, remove it since it cannot be right.
	for i, placeholder := range placeholders {
		text := placeholder.Text(docBytes)
		if !strings.ContainsRune(text, OpenDelimiter) ||
			!strings.ContainsRune(text, CloseDelimiter) {
			placeholders = append(placeholders[:i], placeholders[i+1:]...)
		}
	}

	return placeholders, nil
}

// assembleFullPlaceholders will extract all complete placeholders inside the run given a open and close position.
// The open and close positions are the positions of the Delimiters which must already be known at this point.
// openPos and closePos are expected to be symmetrical (e.g. same length).
// Example: openPos := []int{10,20,30}; closePos := []int{13, 23, 33}
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
