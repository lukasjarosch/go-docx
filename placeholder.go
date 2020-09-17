package docx

import (
	"fmt"
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
	end := len(p.Fragments) -1
	return p.Fragments[end].Run.Text.StartTag.End + p.Fragments[end].Position.End
}

func ParsePlaceholders(runs DocumentRuns, docBytes []byte) (placeholders []*Placeholder) {
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
			placeholders = append(placeholders, parseFullPlaceholders(run, openPos, closePos)...)
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
			placeholders = append(placeholders, parseFullPlaceholders(run, openPos[:len(openPos)-1], closePos)...)

			// add the unclosed part of the placeholder to a tmp placeholder var
			unclosedOpenPos := openPos[len(openPos)-1]
			fragment := &PlaceholderFragment{
				Position: Position{
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
			// merge full placeholders in the run, leaving out the last closePos since
			// we know that the one is left over and must be handled separately below
			placeholders = append(placeholders, parseFullPlaceholders(run, openPos, closePos[:len(closePos) - 1])...)

			// there is only a closePos and no open pos
			if len(closePos) == 1 {
				fragment := &PlaceholderFragment{
					Position: Position{
						Start: 0,
						End:   int64(closePos[0])+1,
					},
					Number: len(unclosedPlaceholder.Fragments) + 1,
					Run:    run,
				}
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
				fragment := &PlaceholderFragment{
					Position: Position{
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
	}
	return placeholders
}

func parseFullPlaceholders(run *Run, openPos, closePos []int) (placeholders []*Placeholder){
	for i := 0; i < len(openPos); i++ {
		start := openPos[i]
		end := closePos[i] + 1 // +1 is required to include the closing delimiter in the text
		fragment := &PlaceholderFragment{
			Position: Position{
				Start: int64(start),
				End:   int64(end),
			},
			Number: 0,
			Run:    run,
		}
		p := &Placeholder{Fragments: []*PlaceholderFragment{fragment}}
		placeholders = append(placeholders, p)
	}
	return placeholders
}

type PlaceholderFragment struct {
	Position Position // Position of the actual fragment within the run text. 0 == (Run.Text.StartTag.End + 1)
	Number   int      // numbering fragments for ease of use
	Run      *Run
}

// StartPos returns the absolute start position of the fragment.
func (p PlaceholderFragment) StartPos() int64 {
	return p.Run.Text.StartTag.End + p.Position.Start
}

// EndPos returns the absolute end position of the fragment.
func (p PlaceholderFragment) EndPos() int64 {
	return p.Run.Text.StartTag.End + p.Position.End
}

func (p PlaceholderFragment) Text(docBytes []byte) string {
	return string(docBytes[p.StartPos():p.EndPos()])
}

func (p PlaceholderFragment) TextLength(docBytes []byte) int64 {
	return int64(len(p.Text(docBytes)))
}

func (p PlaceholderFragment) String(docBytes []byte) string {
	format := "fragment in run [%d:%d] '%s' - [%d:%d] '%s'; run-text [%d:%d] '%s' - [%d:%d] '%s'; positions: [%d:%d] '%s'"
	return fmt.Sprintf(format,
		p.Run.StartTag.Start, p.Run.StartTag.End, docBytes[p.Run.StartTag.Start:p.Run.StartTag.End],
		p.Run.EndTag.Start, p.Run.EndTag.End, docBytes[p.Run.EndTag.Start:p.Run.EndTag.End],
		p.Run.Text.StartTag.Start, p.Run.Text.StartTag.End, docBytes[p.Run.Text.StartTag.Start:p.Run.Text.StartTag.End],
		p.Run.Text.EndTag.Start, p.Run.Text.EndTag.End, docBytes[p.Run.Text.EndTag.Start:p.Run.Text.EndTag.End],
		p.Position.Start, p.Position.End, docBytes[p.Run.Text.StartTag.End+p.Position.Start:p.Run.Text.StartTag.End+p.Position.End])
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
