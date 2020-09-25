package docx

import (
	"errors"
	"strings"
	"sync"
)

var (
	ErrPlaceholderNotFound = errors.New("placeholder not found in document")
)

type DocumentFile interface {
	DocumentBytes() []byte
	HeaderBytes() [][]byte
	FooterBytes() [][]byte
}

// Replacer is the key struct which works on the parsed DOCX document.
type Replacer struct {
	document     []byte
	placeholders []*Placeholder
	ReplaceCount int
	BytesChanged int64
	mu sync.Mutex
}

// NewReplacer returns a new Replacer.
func NewReplacer(docBytes []byte, placeholder []*Placeholder) *Replacer {
	return &Replacer{
		document:     docBytes,
		placeholders: placeholder,
		ReplaceCount: 0,
	}
}


// Replace will replace all occurrences of the placeholderKey with the given value.
// The function is synced with a mutex as it is not concurrency safe.
func (r *Replacer) Replace(placeholderKey string, value string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !strings.ContainsRune(placeholderKey, OpenDelimiter) ||
		!strings.ContainsRune(placeholderKey, CloseDelimiter) {
		placeholderKey = AddPlaceholderDelimiter(placeholderKey)
	}

	// find all occurrences of the placeholderKey inside r.placeholders
	found := false
	for i := 0; i < len(r.placeholders); i++ {
		placeholder := r.placeholders[i]
		if placeholder.Text(r.document) == placeholderKey {
			found = true
			// replace text of the placeholder's first fragment with the actual value
			// the other fragments of the placeholder are cut, leaving only the value inside the document.
			r.replaceFragmentValue(placeholder.Fragments[0], value)
			for i := 1; i < len(placeholder.Fragments); i++ {
				r.cutFragment(placeholder.Fragments[i])
			}
		}
	}
	if !found {
		return ErrPlaceholderNotFound
	}
	return nil
}

// DocumentBytes returns the document bytes.
// If called after Replace(), the bytes will be modified.
func (r *Replacer) Bytes() []byte {
	return r.document
}

// replaceFragmentValue will replace the fragment text with the given value, adjusting all following
// fragments afterwards.
func (r *Replacer) replaceFragmentValue(fragment *PlaceholderFragment, value string) {
	var deltaLength int64

	docBytes := r.document
	valueLength := int64(len(value))
	fragLength := fragment.EndPos() - fragment.StartPos()
	deltaLength = valueLength - fragLength

	// cut out the fragment text literal
	cutStart := fragment.Run.Text.StartTag.End + fragment.Position.Start
	cutEnd := fragment.Run.Text.StartTag.End + fragment.Position.End
	docBytes = append(docBytes[:cutStart], docBytes[cutEnd:]...)

	// insert the value from the cut start position
	docBytes = append(docBytes[:cutStart], append([]byte(value), docBytes[cutStart:]...)...)

	// shift everything which is after the replaced value for this fragment
	fragment.Run.Text.EndTag.Start += deltaLength
	fragment.Run.Text.EndTag.End += deltaLength
	fragment.Run.CloseTag.Start += deltaLength
	fragment.Run.CloseTag.End += deltaLength
	fragment.Position.End += deltaLength

	r.document = docBytes
	r.ReplaceCount++
	r.BytesChanged += deltaLength
	r.shiftFollowingFragments(fragment, deltaLength)
}

// curFragment will remove the fragment text from the document bytes.
// Afterwards, all following fragments will be adjusted.
func (r *Replacer) cutFragment(fragment *PlaceholderFragment) {
	docBytes := r.document
	cutStart := fragment.Run.Text.StartTag.End + fragment.Position.Start
	cutEnd := fragment.Run.Text.StartTag.End + fragment.Position.End
	cutLength := fragment.Position.End - fragment.Position.Start

	docBytes = append(docBytes[:cutStart], docBytes[cutEnd:]...)

	// adjust fragment after cut
	fragment.Run.Text.EndTag.Start -= cutLength
	fragment.Run.Text.EndTag.End -= cutLength
	fragment.Run.CloseTag.Start -= cutLength
	fragment.Run.CloseTag.End -= cutLength
	fragment.Position.End = fragment.Position.Start

	r.document = docBytes
	r.BytesChanged -= cutLength
	r.shiftFollowingFragments(fragment, -cutLength)
}


// shiftFollowingFragments is responsible of shifting all fragments following the given fragment by some amount.
func (r *Replacer) shiftFollowingFragments(fromFragment *PlaceholderFragment, deltaLength int64) {
	// handle all fragments which share a run with the given fragment.
	// this happens for example if there are multiple placeholders in the same line.
	sharedRunFragments := r.fragmentsInRun(fromFragment.Run)
	for _, frag := range sharedRunFragments {
		if frag == fromFragment {
			continue // ignore the fromFragment. It is expected to be correct already.
		}
		fragment := frag

		// If fromFragment is actually after the fragment there is nothing to do as the bytes
		// did not shift for those.
		// Example: (fromFragment == {foo}): {key}{key}{foo}
		if fromFragment.Position.Start > frag.Position.Start {
			continue
		}

		// fragment in the same run is after fromFragment and thus only the position needs to be adjusted
		fragment.Position.Start += deltaLength
		fragment.Position.End += deltaLength
	}

	followingFragments := r.fragmentsFromPosition(fromFragment.Run.Text.StartTag.End + fromFragment.Position.End)

	// remove fragments which have been adjusted already above
	for i, fragment := range followingFragments {
		alreadyHandled := func(fragment *PlaceholderFragment) bool {
			for _, runFragment := range sharedRunFragments {
				if fragment == runFragment {
					return true
				}
			}
			return false
		}
		if alreadyHandled(fragment) {
			followingFragments = append(followingFragments[:i], followingFragments[i+1:]...)
		}
	}

	// we need to keep track of which runs were already modified.
	// This is important since there may be following fragments which share a run
	var modifiedRuns []*Run
	isAlreadyModified := func(r *Run) bool {
		for _, run := range modifiedRuns {
			if run == r {
				return true
			}
		}
		return false
	}

	// shift all fragments following fromFragment and are in a different Run
	for _, frag := range followingFragments {
		if isAlreadyModified(frag.Run) {
			continue
		}
		followingFragment := frag
		followingFragment.Run.OpenTag.Start += deltaLength
		followingFragment.Run.OpenTag.End += deltaLength
		followingFragment.Run.CloseTag.Start += deltaLength
		followingFragment.Run.CloseTag.End += deltaLength
		followingFragment.Run.Text.StartTag.Start += deltaLength
		followingFragment.Run.Text.StartTag.End += deltaLength
		followingFragment.Run.Text.EndTag.Start += deltaLength
		followingFragment.Run.Text.EndTag.End += deltaLength
		modifiedRuns = append(modifiedRuns, followingFragment.Run)
	}
}

// fragmentsFromPosition will return all fragments where: fragment.Run.OpenTag.Start > startingFrom
func (r *Replacer) fragmentsFromPosition(startingFrom int64) (found []*PlaceholderFragment) {
	for _, placeholder := range r.placeholders {
		for _, fragment := range placeholder.Fragments {
			if fragment.Run.OpenTag.Start > startingFrom {
				found = append(found, fragment)
				continue
			}
		}
	}
	return found
}

// fragmentsInRun returns all fragment which are in the given Run.
func (r *Replacer) fragmentsInRun(run *Run) (fragments []*PlaceholderFragment) {
	for _, placeholder := range r.placeholdersInRun(run) {
		for _, fragment := range placeholder.Fragments {
			if fragment.Run == run {
				fragments = append(fragments, fragment)
			}
		}
	}
	return fragments
}

// placeholdersInRun returns all placeholders which belong to the given Run
func (r *Replacer) placeholdersInRun(run *Run) (p []*Placeholder) {
	for _, placeholder := range r.placeholders {
		for _, fragment := range placeholder.Fragments {
			if fragment.Run == run {
				p = append(p, placeholder)
				continue
			}
		}
	}
	return p
}

