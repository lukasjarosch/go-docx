package docx

import (
	"container/list"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log"
	"regexp"
)

const (
	// RunElementName is the local name of the XML tag for runs (<w:r>, </w:r> and <w:r/>)
	RunElementName = "r"
	// TextElementName is the local name of the XML tag for text-runs (<w:t> and </w:t>)
	TextElementName = "t"
)

var (
	// RunOpenTagRegex matches all OpenTags for runs, including eventually set attributes
	RunOpenTagRegex = regexp.MustCompile(`(<w:r).*>`)
	// RunCloseTagRegex matches the close tag of runs
	RunCloseTagRegex = regexp.MustCompile(`(</w:r>)`)
	// RunSingletonTagRegex matches a singleton run tag
	RunSingletonTagRegex = regexp.MustCompile(`(<w:r/>)`)
	// TextOpenTagRegex matches all OpenTags for text-runs, including eventually set attributes
	TextOpenTagRegex = regexp.MustCompile(`(<w:t).*>`)
	// TextCloseTagRegex matches the close tag of text-runs
	TextCloseTagRegex = regexp.MustCompile(`(</w:t>)`)
	// ErrTagsInvalid is returned if the parsing failed and the result cannot be used.
	// Typically this means that one or more tag-offsets were not parsed correctly which
	// would cause the document to become corrupted as soon as replacing starts.
	ErrTagsInvalid = errors.New("one or more tags are invalid and will cause the XML to be corrupt")
)

// RunParser can parse a list of Runs from a given byte slice.
type RunParser struct {
	doc      []byte
	runs     DocumentRuns
	runStack list.List
}

// NewRunParser returns an initialized RunParser given the source-bytes.
func NewRunParser(doc []byte) *RunParser {
	return &RunParser{
		doc:  doc,
		runs: DocumentRuns{},
	}
}

// Execute will fire up the parser.
// The parser will do two passes on the given document.
// First, all <w:r> tags are located and marked.
// Then, inside that run tags the <w:t> tags are located.
func (parser *RunParser) Execute() error {
	err := parser.findRuns()
	if err != nil {
		return err
	}
	err = parser.findTextRuns()
	if err != nil {
		return err
	}

	return ValidatePositions(parser.doc, parser.runs)
}

// Runs returns the all runs found by the parser.
func (parser *RunParser) Runs() DocumentRuns {
	return parser.runs
}

// FindRuns will search through the document and return all runs found.
// The text tags are not analyzed at this point, that'str the next step.
func (parser *RunParser) findRuns() error {
	// use a custom reader which saves the current byte position
	docReader := NewReader(string(parser.doc))
	decoder := xml.NewDecoder(docReader)

	tmpRun := NewEmptyRun()
	singleton := false

	// nestCount holds the nesting-level. It is going to be incremented on every OpenTag and decremented
	// on every CloseTag.
	nestCount := 0

	// popRun will pop the last Run from the runStack if there is any on the stack
	popRun := func() *Run {
		r := parser.runStack.Back().Value.(*Run)
		parser.runStack.Remove(parser.runStack.Back())
		return r
	}

	// nextIteration resets the temporary values used inside the for-loop to be ready for the next iteration
	// This is used after a run has been fully analyzed (OpenTag and CloseTag were found).
	// As long as there are runs on the runStack, they will be popped from it.
	// Only when the stack is empty, a new empty Run struct is created.
	nextIteration := func() {
		nestCount -= 1
		if nestCount > 0 {
			tmpRun = popRun()
		} else {
			tmpRun = NewEmptyRun()
		}
		singleton = false
	}

	for {
		tok, err := decoder.Token()
		if tok == nil || err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("error getting token: %s", err)
		}

		switch elem := tok.(type) {
		case xml.StartElement:
			if elem.Name.Local == RunElementName {

				nestCount += 1
				if nestCount > 1 {
					parser.runStack.PushBack(tmpRun)
					tmpRun = NewEmptyRun()
				}

				// tagEndPos points to '>' of the tag
				tagEndPos := docReader.Pos()
				// tagStartPos points to '<' of the tag
				tagStartPos := parser.findOpenBracketPos(tagEndPos - 1)

				tmpRun.OpenTag = Position{
					Start: tagStartPos,
					End:   tagEndPos,
				}

				// special case, a singleton tag: <w:r/> is also considered to be a start element
				// since there is no real end tag, the element is marked for the EndElement case to handle it appropriately
				tagStr := string(parser.doc[tagStartPos:tagEndPos])
				if RunSingletonTagRegex.MatchString(tagStr) {
					singleton = true
				}
			}

		case xml.EndElement:
			if elem.Name.Local == RunElementName {

				// if the run is a singleton tag, it was already identified by the xml.StartElement case
				// in that case, the CloseTag is the same as the openTag and no further work needs to be done
				if singleton {
					tmpRun.CloseTag = tmpRun.OpenTag
					parser.runs = append(parser.runs, tmpRun) // run is finished
					nextIteration()
					break
				}

				// tagEndPos points to '>' of the tag
				tagEndPos := docReader.Pos()
				// tagStartPos points to '<' of the tag
				tagStartPos := parser.findOpenBracketPos(tagEndPos - 1)

				// add CloseTag and finish the run
				tmpRun.CloseTag = Position{
					Start: tagStartPos,
					End:   tagEndPos,
				}
				parser.runs = append(parser.runs, tmpRun)

				nextIteration()
			}
		}
	}

	if nestCount != 0 {
		log.Printf("invalid nestCount, should be 0 but is %d\n", nestCount)
		return ErrTagsInvalid
	}

	return nil
}

func (parser *RunParser) findTextRuns() error {
	// use a custom reader which saves the current byte position
	docReader := NewReader(string(parser.doc))
	decoder := xml.NewDecoder(docReader)

	// based on the current position, find out in which run we're at
	inRun := func(pos int64) *Run {
		for _, run := range parser.runs {
			if run.OpenTag.Start < pos && pos < run.CloseTag.End {
				return run
			}
		}
		return nil
	}

	for {
		tok, err := decoder.Token()
		if tok == nil || err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("error getting token: %s", err)
		}

		switch elem := tok.(type) {
		case xml.StartElement:
			if elem.Name.Local == TextElementName {

				// tagEndPos points to '>' of the tag
				tagEndPos := docReader.Pos()
				// tagStartPos points to '<' of the tag
				tagStartPos := parser.findOpenBracketPos(tagEndPos - 1)

				currentRun := inRun(docReader.Pos())
				if currentRun == nil {
					return fmt.Errorf("unable to find currentRun for text start-element")
				}
				currentRun.HasText = true
				currentRun.Text.OpenTag = Position{
					Start: tagStartPos,
					End:   tagEndPos,
				}
			}

		case xml.EndElement:
			if elem.Name.Local == TextElementName {

				// tagEndPos points to '>' of the tag
				tagEndPos := docReader.Pos()
				// tagStartPos points to '<' of the tag. -1 is required since Pos() points after the '>'
				tagStartPos := parser.findOpenBracketPos(tagEndPos - 1)

				currentRun := inRun(docReader.Pos())
				if currentRun == nil {
					return fmt.Errorf("unable to find currentRun for text end-element")
				}
				currentRun.Text.CloseTag = Position{
					Start: tagStartPos,
					End:   tagEndPos,
				}
			}
		}
	}

	return nil
}

// findOpenBracketPos searches the matching '<' for a close bracket ('>') given it's position.
func (parser *RunParser) findOpenBracketPos(endBracketPos int64) int64 {
	var found bool
	for i := endBracketPos; !found; i-- {
		if string(parser.doc[i]) == "<" {
			return i
		}
	}
	return 0
}

// ValidatePositions will iterate over all runs and their texts (if any) and ensure that they match
// their respective regex.
// If the validation failed, the replacement will not work since offsets are wrong.
func ValidatePositions(document []byte, runs []*Run) error {
	parsingFailed := false
	for _, run := range runs {

		// singleton tags must not be validated
		if run.OpenTag.Match(RunSingletonTagRegex, document) {
			continue
		}

		if !run.OpenTag.Match(RunOpenTagRegex, document) {
			log.Println("RunOpenTagRegex failed to match", run.String(document))
			parsingFailed = true
		}
		if !run.CloseTag.Match(RunCloseTagRegex, document) {
			log.Println("RunCloseTagRegex failed to match", run.String(document))
			parsingFailed = true
		}

		if run.HasText {
			if !run.Text.OpenTag.Match(TextOpenTagRegex, document) {
				log.Println("TextOpenTagRegex failed to match", run.String(document))
				parsingFailed = true
			}
			if !run.Text.CloseTag.Match(TextCloseTagRegex, document) {
				log.Println("TextCloseTagRegex failed to match", run.String(document))
				parsingFailed = true
			}
		}
	}
	if parsingFailed {
		return ErrTagsInvalid
	}

	return nil
}

// Position is a generic position of a tag, represented by byte offsets
type Position struct {
	Start int64
	End   int64
}

// Match will apply a MatchString using the given regex on the given data and returns true if the position
// matches the regex inside the data.
func (p Position) Match(regexp *regexp.Regexp, data []byte) bool {
	return regexp.MatchString(string(data[p.Start:p.End]))
}

// Valid returns true if Start <= End.
// Only then the position can be used, otherwise there will be a 'slice out of bounds' along the way.
func (p Position) Valid() bool {
	return p.Start <= p.End
}
