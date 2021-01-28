package docx

import "fmt"

var (
	runId = 0 // global Run ID counter. Incremented by NewRun()
)

// Run defines a non-block region of text with a common set of properties.
// It is specified with the <w:r> element.
// In our case the run is specified by four byte positions (start and end tag).
type Run struct {
	ID       int
	OpenTag  Position
	CloseTag Position
	Text     TextRun
	HasText  bool
}

// NewEmptyRun returns a new, empty run which has only an ID set.
func NewEmptyRun() *Run {
	runId += 1
	return &Run{
		ID: NewRunID(),
	}
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

// String returns a string representation of the run, given the source bytes.
// It may be helpful in debugging.
func (r *Run) String(bytes []byte) string {
	format := "run %d from offset [%d:%d] '%s' to [%d:%d] '%s; run-text offset from [%d:%d] '%s' to [%d:%d] '%s'"
	return fmt.Sprintf(format, r.ID,
		r.OpenTag.Start, r.OpenTag.End, bytes[r.OpenTag.Start:r.OpenTag.End],
		r.CloseTag.Start, r.CloseTag.End, bytes[r.CloseTag.Start:r.CloseTag.End],
		r.Text.StartTag.Start, r.Text.StartTag.End, bytes[r.Text.StartTag.Start:r.Text.StartTag.End],
		r.Text.EndTag.Start, r.Text.EndTag.End, bytes[r.Text.EndTag.Start:r.Text.EndTag.End],
	)
}

// DocumentRuns is a convenience type used to describe a slice of runs.
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

// NewRunID returns the next Fragment.ID
func NewRunID() int {
	runId += 1
	return runId
}
