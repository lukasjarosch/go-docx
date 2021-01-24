package docx

import (
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

// RunParser can parse a list of Runs from a given byte slice.
type RunParser struct {
	doc []byte
	runs DocumentRuns
}

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
	return nil
}

func (parser *RunParser) Runs() DocumentRuns {
	return parser.runs
}

// FindRuns will search through the document and return all runs found.
// The text tags are not analyzed at this point, that's the next step.
func (parser *RunParser) findRuns() error {
	// use a custom reader which saves the current byte position
	docReader := NewReader(string(parser.doc))
	decoder := xml.NewDecoder(docReader)

	// find all runs in document.xml
	tmpRun := NewEmptyRun()
	singleElement := false

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
			if elem.Name.Local == "r" {
				openTag := "<w:r>"
				// docReader.Pos() points to the '>' of the opening 'r' element (<w:r>)
				tmpRun.OpenTag = TagPosition(docReader.Pos(), openTag)

				// special case, an empty tag: <w:r/> is also considered to be a start element
				// the tag is 1 byte longer which needs to be addressed
				// since there is no real end tag, the element is marked for the EndElement case to handle it appropriately
				tagStr := string(parser.doc[tmpRun.OpenTag.Start:tmpRun.OpenTag.End])
				if strings.Contains(tagStr, "/>") {
					tmpRun.OpenTag.Start -= 1
					singleElement = true
				}
			}

		case xml.EndElement:
			if elem.Name.Local == "r" {
				closeTag := "</w:r>"

				// if the run is a single element (<w:r/>), the values of the OpenTag are already correct and must
				// be identical.
				if singleElement {
					singleElement = false
					tmpRun.CloseTag = tmpRun.OpenTag
					parser.runs = append(parser.runs, tmpRun)
					tmpRun = NewEmptyRun()
					break
				}
				tmpRun.CloseTag = TagPosition(docReader.Pos(), closeTag)
				parser.runs = append(parser.runs, tmpRun)
				tmpRun = NewEmptyRun()
			}
		}
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
			if elem.Name.Local == "t" {
				run := inRun(docReader.Pos())
				openTag := "<w:t>"
				run.HasText = true
				run.Text.StartTag = TagPosition(docReader.Pos(), openTag)
			}

		case xml.EndElement:
			if elem.Name.Local == "t" {
				run := inRun(docReader.Pos())
				closeTag := "</w:t>"
				run.Text.EndTag = TagPosition(docReader.Pos(), closeTag)
			}
		}
	}

	return nil
}

// TagPosition returns a filled Position struct given the end position and the tag itself.
func TagPosition(endPos int64, tag string) (tp Position) {
	tp.End = endPos
	tp.Start = endPos - int64(len(tag))
	return tp
}


// TextRun defines the <w:t> element which contains the actual literal text data.
// A TextRun is always a child of a Execute.
type TextRun struct {
	StartTag Position
	EndTag   Position
}

// Position is a generic position of a tag, represented by byte offsets
type Position struct {
	Start int64
	End   int64
}
