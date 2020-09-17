package docx

import (
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)


// A run defines a non-block region of text with a common set of properties.
// It is specified with the <w:r> element.
// In our case the run is specified by four byte positions (start and end tag).
type Run struct {
	StartTag Position
	EndTag   Position
	Text     TextRun
	HasText  bool
}

// GetText returns the text of the run, if any.
// If the run does not have a text or the given byte slice is too small, an empty string is returned
func (r *Run) GetText(documentBytes []byte) string {
	if !r.HasText {
		return ""
	}
	startPos := r.Text.StartTag.End
	endPos := r.Text.EndTag.Start

	if int64(len(documentBytes)) < startPos || int64(len(documentBytes)) < endPos {
		return ""
	}

	return string(documentBytes[startPos:endPos])
}

type DocumentRuns []*Run

// WithText returns all runs with the HasText flag set
func (dr DocumentRuns) WithText() DocumentRuns {
	var r DocumentRuns
	for _, run := range dr {
		if run.HasText {
			r = append(r, run)
		}
	}
	return r
}

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
	tmpRun := &Run{}
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
				tmpRun.StartTag = TagPosition(docReader.Pos(), openTag)

				// special case, an empty tag: <w:r/> is also considered to be a start element
				// the tag is 1 byte longer which needs to be addressed
				// since there is no real end tag, the element is marked for the EndElement case to handle it appropriately
				tagStr := string(parser.doc[tmpRun.StartTag.Start:tmpRun.StartTag.End])
				if strings.Contains(tagStr, "/>") {
					tmpRun.StartTag.Start -= 1
					singleElement = true
				}

				//log.Printf("%d: START: %s", docReader.Pos(), doc.DocumentBytes()[startRunTagStartPos:startRunTagEndPos])
			}
			break

		case xml.EndElement:
			if elem.Name.Local == "r" {
				closeTag := "</w:r>"

				// if the run is a single element (<w:r/>), the values of the StartTag are already correct and must
				// be identical.
				if singleElement {
					singleElement = false
					tmpRun.EndTag = tmpRun.StartTag
					parser.runs = append(parser.runs, tmpRun)
					tmpRun = &Run{}
					break
				}
				tmpRun.EndTag = TagPosition(docReader.Pos(), closeTag)
				parser.runs = append(parser.runs, tmpRun)
				tmpRun = &Run{}
			}
			break
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
			if run.StartTag.Start < pos && pos < run.EndTag.End {
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
			break

		case xml.EndElement:
			if elem.Name.Local == "t" {
				run := inRun(docReader.Pos())
				closeTag := "</w:t>"
				run.Text.EndTag = TagPosition(docReader.Pos(), closeTag)
			}
			break
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
