package docx

import "fmt"

var (
	fragmentId = 0 // global fragment id counter, incremented on NewPlaceholderFragment
)

// PlaceholderFragment is a part of a placeholder within the document.xml
// If the full placeholder is e.g. '{foo-bar}', the placeholder might be ripped
// apart according to the WordprocessingML spec. So it will most likely occur, that
// the placeholders are split into multiple fragments (e.g. '{foo' and '-bar}').
type PlaceholderFragment struct {
	ID       int      // ID is used to identify the fragments globally.
	Position Position // Position of the actual fragment within the run text. 0 == (Run.Text.OpenTag.End + 1)
	Number   int      // numbering fragments for ease of use. Numbering is scoped to placeholders.
	Run      *Run
}

// NewPlaceholderFragment returns an initialized PlaceholderFragment with a new, auto-incremented, ID.
func NewPlaceholderFragment(number int, pos Position, run *Run) *PlaceholderFragment {
	return &PlaceholderFragment{
		ID:       NewFragmentID(),
		Position: pos,
		Number:   number,
		Run:      run,
	}
}

// ShiftAll will shift all fragment position markers by the given amount.
// The function is used if the underlying byte-data changed and the whole PlaceholderFragment needs to be
// shifted to a new position to be correct again.
//
// For example, 10 bytes were added to the document and this PlaceholderFragment is positioned after that change
// inside the document. In that case one needs to shift the fragment by +10 bytes using ShiftAll(10).
func (p *PlaceholderFragment) ShiftAll(deltaLength int64) {
	p.Run.OpenTag.Start += deltaLength
	p.Run.OpenTag.End += deltaLength
	p.Run.CloseTag.Start += deltaLength
	p.Run.CloseTag.End += deltaLength
	p.Run.Text.OpenTag.Start += deltaLength
	p.Run.Text.OpenTag.End += deltaLength
	p.Run.Text.CloseTag.Start += deltaLength
	p.Run.Text.CloseTag.End += deltaLength
}

// ShiftCut will shift the fragment position markers in such a way that the fragment can be considered empty.
// This is used in order to preserve the correct positions of the tags.
//
// The function is used if the actual value (text-run value) of the fragment has been removed.
// For example the fragment-text was: 'remove-me' (9 bytes)
// If that data was removed from the document, the positions (not all positions) of the fragment need to be adjusted.
// The text positions are set equal (start == end).
func (p *PlaceholderFragment) ShiftCut(cutLength int64) {
	p.Run.Text.CloseTag.Start -= cutLength
	p.Run.Text.CloseTag.End -= cutLength
	p.Run.CloseTag.Start -= cutLength
	p.Run.CloseTag.End -= cutLength
	p.Position.End = p.Position.Start
}

// ShiftReplace is used to adjust the fragment positions after the text value has been replaced.
// The function is used if the text-value of the fragment has been replaced with different bytes.
// For example, the fragment text was 'placeholder' (11 bytes) which is replaced with 'a-super-awesome-value' (21 bytes)
// In that case the deltaLength would be 10. In order to accommodate for the change in bytes you'd need to call ShiftReplace(10)
func (p *PlaceholderFragment) ShiftReplace(deltaLength int64) {
	p.Run.Text.CloseTag.Start += deltaLength
	p.Run.Text.CloseTag.End += deltaLength
	p.Run.CloseTag.Start += deltaLength
	p.Run.CloseTag.End += deltaLength
	p.Position.End += deltaLength
}

// StartPos returns the absolute start position of the fragment.
func (p PlaceholderFragment) StartPos() int64 {
	return p.Run.Text.OpenTag.End + p.Position.Start
}

// EndPos returns the absolute end position of the fragment.
func (p PlaceholderFragment) EndPos() int64 {
	return p.Run.Text.OpenTag.End + p.Position.End
}

// Text returns the actual text of the fragment given the source bytes.
// If the given byte slice is not large enough for the offsets, an empty string is returned.
func (p PlaceholderFragment) Text(docBytes []byte) string {
	if int64(len(docBytes)) < p.StartPos() ||
		int64(len(docBytes)) < p.EndPos() {
		return ""
	}
	return string(docBytes[p.StartPos():p.EndPos()])
}

// TextLength returns the actual length of the fragment given a byte source.
func (p PlaceholderFragment) TextLength(docBytes []byte) int64 {
	return int64(len(p.Text(docBytes)))
}

// String spits out the most important bits and pieces of a fragment and can be used for debugging purposes.
func (p PlaceholderFragment) String(docBytes []byte) string {
	format := "fragment %d in %s with fragment text-positions: [%d:%d] '%s'"
	return fmt.Sprintf(format, p.ID, p.Run.String(docBytes),
		p.Position.Start, p.Position.End, docBytes[p.Run.Text.OpenTag.End+p.Position.Start:p.Run.Text.OpenTag.End+p.Position.End])
}

// Valid returns true if all positions of the fragment are valid.
func (p PlaceholderFragment) Valid() bool {
	return p.Run.OpenTag.Valid() &&
		p.Run.CloseTag.Valid() &&
		p.Run.Text.OpenTag.Valid() &&
		p.Run.Text.CloseTag.Valid() &&
		p.Position.Valid()
}

// NewFragmentID returns the next Fragment.ID
func NewFragmentID() int {
	fragmentId += 1
	return fragmentId
}

// ResetFragmentIdCounter will reset the fragmentId counter to 0
func ResetFragmentIdCounter() {
	fragmentId = 0
}
