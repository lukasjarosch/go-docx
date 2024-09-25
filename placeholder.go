package docx

import (
	"fmt"
	"log"
	"strings"
)

var (
	// OpenDelimiter defines the opening delimiter for the placeholders used inside a docx-document.
	OpenDelimiter rune = '{'
	// CloseDelimiter defines the closing delimiter for the placeholders used inside a docx-document.
	CloseDelimiter rune = '}'
)

// ChangeOpenCloseDelimiter is used for change the open and close delimiters
func ChangeOpenCloseDelimiter(openDelimiter, closeDelimiter rune) {
	OpenDelimiter = openDelimiter
	CloseDelimiter = closeDelimiter
}

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
	// Use stack to trace the delimiter pair
	stack := []*PlaceholderFragment{}
	for _, run := range runs.WithText() {
		hasDelimiter := false
		runRune := []rune(run.GetText(docBytes))
		for i := 0; i < len(runRune); i++ {
			// There is an open delimiter in the run, thus create a partial placeholder fragment
			if runRune[i] == OpenDelimiter {
				hasDelimiter = true
				stack = append(stack, NewPlaceholderFragment(Position{int64(i), -1}, run))
				continue
			}

			if runRune[i] == CloseDelimiter {
				// There is a close delimiter in the run, 3 scenarios may happen:
				// 1) The stack is empty, no open delimiter can match this close delimiter,
				//    this must be a corrupted placeholder, we log the error and skip
				if len(stack) == 0 {
					log.Printf(
						"detected unmatched close delimiter in run %d \"%s\", index %d, skipping \n",
						run.ID, run.GetText(docBytes), i,
					)
					continue
				}

				// 2) The stack is not empty,
				hasDelimiter = true
				fragment := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				if run == fragment.Run {
					// a) The close delimiter is in the same run as the open delimiter, then we take
					//    the partial fragment from the top of the stack, and complete its end position, to make a
					//    complete placeholder with only 1 fragment.
					// e.g., run like:
					//   foo{bar}baz
					//   foo{bar}baz{qux}bbb
					fragment.Position.End = int64(i) + 1
					placeholders = append(placeholders, &Placeholder{Fragments: []*PlaceholderFragment{fragment}})
				} else {
					// b) There are some span runs between the run of open and close delimiter, then we first
					//    take the partial fragment from the top of the stack, and its end position must be the end of
					//    that run. Then we create span fragments, with its length set to the run length. Finally, we
					//    create the fragment that includes the close delimiter, with its start position set to 0, and
					//    end position set to the position of the close delimiter.
					// e.g., run like (here | is the run boundary):
					//   foo{bar|}baz		   => {bar}
					//   foo{bar|abc|}baz      => {barabc}
					//   foo{bar|abc|def|}baz  => {barabcdef}
					//   foo{bar|{bc|d}ef|}baz => {bar{bcd}ef} {bcd}
					fragment.Position.End = int64(len(fragment.Run.GetText(docBytes)))
					fragments := []*PlaceholderFragment{fragment}
					for _, srun := range fragment.SpanRun {
						fragments = append(
							fragments,
							NewPlaceholderFragment(Position{0, int64(len(srun.GetText(docBytes)))}, srun),
						)
					}
					fragments = append(fragments, NewPlaceholderFragment(Position{0, int64(i) + 1}, run))
					placeholders = append(placeholders, &Placeholder{Fragments: fragments})
				}
				continue
			}
		}
		if !hasDelimiter {
			// If a run has no delimiter, it must be a span run. Thus we add the run to all the partial framents that
			// has not been closed.
			for i := 0; i < len(stack); i++ {
				stack[i].SpanRun = append(stack[i].SpanRun, run)
				continue
			}
		}
	}

	// Warn user there are some unmatched open delimiters (a.k.a corrupted placeholders) left in the stack
	for _, fragment := range stack {
		log.Printf("detected unmatched open delimiter in run %d \"%s\", index %d, skipping \n", fragment.Run.ID, fragment.Run.GetText(docBytes), fragment.Position.Start)
	}

	return placeholders, nil
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
