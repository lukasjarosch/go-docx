package docx

import (
	"log"
	"strings"
)

type Replacer struct {
	document     []byte
	placeholders []*Placeholder
	ReplaceCount int
	BytesChanged int64
}

func NewReplacer(docBytes []byte, placeholder []*Placeholder) *Replacer {
	return &Replacer{
		document:     docBytes,
		placeholders: placeholder,
		ReplaceCount: 0,
	}
}

func (r *Replacer) Bytes() []byte {
	return r.document
}

func (r *Replacer) findRunPlaceholders(run *Run) (p []*Placeholder) {
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

func (r *Replacer) replaceFragmentValue(placeholder *Placeholder, value string) {
	// we're doing the replacing only on the first fragment. all other fragments will be emptied
	frag := placeholder.Fragments[0]
	deltaLen := int64(len(value) - len(frag.Text(r.document)))
	/*
		newEndPos := frag.EndPos() + deltaLen
		oldText := frag.Text(r.document)
	*/

	log.Print("PRE 0: ", frag.String(r.document))

	// cut fragment text from bytes
	docBytes := append(r.document[:frag.StartPos()], r.document[frag.EndPos():]...)
	// insert value at the place where the placeholder fragment was
	docBytes = append(r.document[:frag.StartPos()], append([]byte(value), r.document[frag.EndPos():]...)...)

	// ensure the replacement yielded the correct amount of bytes
	if len(docBytes) != len(r.document)+int(deltaLen) {
		log.Fatalf("INCORRECT REPLACE: deltaLen=%d len(r.document)=%d len(docBytes)=%d; len(docBytes) should be %d", deltaLen, len(r.document), len(docBytes), len(r.document)+int(deltaLen))
	}

	// FIXME: this isn't correct
	frag.Position.End += deltaLen
	frag.Run.Text.EndTag.Start += -3
	frag.Run.Text.EndTag.End += -3
	frag.Run.EndTag.Start += -3
	frag.Run.EndTag.End += -3

	log.Print("POST 0: ", frag.String(docBytes))

	// handle all other fragments from this placeholder
	for i := 1; i < len(placeholder.Fragments); i++ {
		frag := placeholder.Fragments[i]
		fragLen := int64(len(frag.Text(docBytes)))

		log.Printf("PRE %d: %s", i, frag.String(docBytes))

		/*Vd
		// adjust the positions, fragLen bytes will be missing
		frag.Position.End -= fragLen
		frag.Run.Text.EndTag.Start = frag.Run.Text.EndTag.Start - fragLen + deltaLen
		frag.Run.Text.EndTag.End = frag.Run.Text.EndTag.End - fragLen + deltaLen
		frag.Run.EndTag.Start = frag.Run.EndTag.Start - fragLen + deltaLen
		frag.Run.EndTag.End = frag.Run.EndTag.End - fragLen + deltaLen

		log.Printf("POST %d: %s", i, frag.String(docBytes))

		// cut fragment text from bytes
		docBytes = append(docBytes[:frag.StartPos()], docBytes[frag.EndPos():]...)

		*/
		_ = fragLen

	}

	/*
		// in the case that the run of the fragment contains multiple placeholders we
		// need to adjust the relative text positions of those as well
		affectedPlaceholders := r.findRunPlaceholders(frag.Run)
		for _, affectedPlaceholder := range affectedPlaceholders {
			// skip the placeholder we've just replaced
			if affectedPlaceholder.Text(docBytes) == value {
				continue
			}

			//log.Print(placeholder.Text(docBytes))
			for _, fragment := range affectedPlaceholder.Fragments {
				frag := fragment

				// special case: there is a placeholder which is only partially inside this run
				// This happens if the placeholder is opened in one run and terminated in another.
				if fragment.Run != frag.Run {
					log.Print(fragment.Text(docBytes))
					frag.Position.Start += deltaLen
					frag.Position.End += deltaLen
					frag.Run.Text.EndTag.Start += deltaLen
					frag.Run.Text.EndTag.End += deltaLen
					frag.Run.EndTag.Start += deltaLen
					frag.Run.EndTag.End += deltaLen
					continue
				}

				frag.Position.Start += deltaLen
				frag.Position.End += deltaLen
				frag.Run.Text.EndTag.Start += deltaLen
				frag.Run.Text.EndTag.End += deltaLen
				frag.Run.EndTag.Start += deltaLen
				frag.Run.EndTag.End += deltaLen
			}
		}

	*/

	r.document = docBytes

	/*
		// handle all following placeholder fragments
		followingFragments := r.findFragmentsFromPos(placeholder.Fragments[len(placeholder.Fragments)-1].EndPos())
		for _, frag := range followingFragments {
			log.Printf("FOLLOW PRE: %s", frag.String(docBytes))
			fragment := frag
			fragment.Run.StartTag.Start += deltaLen
			fragment.Run.StartTag.End += deltaLen

			fragment.Run.EndTag.Start += deltaLen
			fragment.Run.EndTag.End += deltaLen

			fragment.Run.Text.StartTag.Start += deltaLen
			fragment.Run.Text.StartTag.End += deltaLen

			fragment.Run.Text.EndTag.Start += deltaLen
			fragment.Run.Text.EndTag.End += deltaLen

			log.Printf("FOLLOW POST: %s", frag.String(docBytes))
		}

	*/

	/*
		for i, placeholder := range r.placeholders {
			log.Printf("#%.2d [%d:%d]: %s\n", i, placeholder.StartPos(), placeholder.EndPos(), placeholder.Text(r.document))
		}

	*/

	/*
		pos := startPos
		for i := int64(1); i <= int64(len(value)) ; i++ {
			docBytes = append(docBytes, make([]byte, 1)...)
			//fmt.Println(string(docBytes[frag.StartPos():frag.StartPos()+i]))

			copy(docBytes[pos+1:], docBytes[pos:])
			//fmt.Println(string(docBytes[frag.StartPos():frag.StartPos()+i]))

			docBytes[pos] = 'X'
			fmt.Println(string(docBytes[frag.StartPos():frag.StartPos()+1]))
			pos++
		}
		fmt.Println(frag.Text(docBytes))
	*/
}

func (r *Replacer) Replace(placeholderKey string, value string) error {
	if !strings.ContainsRune(placeholderKey, OpenDelimiter) ||
		!strings.ContainsRune(placeholderKey, CloseDelimiter) {
		placeholderKey = AddPlaceholderDelimiter(placeholderKey)
	}

	// deltaLen is the difference between the placeholder literal and the value
	// which needs to be inserted. This is crucial in order to adjust the offsets.
	//deltaLen := int64(len(placeholderKey) - len(value))

	// find the placeholder inside r.placeholders
	for i := 0; i < len(r.placeholders); i++ {
		placeholder := r.placeholders[i]
		if placeholder.Text(r.document) == placeholderKey {
			log.Printf("=== REPLACE #%d %s [%d:%d] ===", i, placeholderKey, placeholder.StartPos(), placeholder.EndPos())

			// The current placeholder can consist of multiple fragments.
			// The value, however, will be placed in the first fragment only.
			// Any remaining fragments need to be cleaned up. If the run's are empty after
			// the cleaning up, they can be removed as well.

			//r.replaceFragmentValue(placeholder, value)

			var deltaLength int64
			for i := 0; i < len(placeholder.Fragments); i++ {
				// first fragment: that's where the value is going to be inserted
				if i == 0 {
					frag := placeholder.Fragments[i]
					docBytes := r.document
					valueLength := int64(len(value))
					fragLength := frag.EndPos() - frag.StartPos()
					deltaLength = valueLength - fragLength

					runStart := frag.Run.StartTag.Start
					runEnd := frag.Run.EndTag.End
					log.Printf("full run: %s", docBytes[runStart:runEnd])

					cutStart := frag.Run.Text.StartTag.End + frag.Position.Start
					cutEnd := frag.Run.Text.StartTag.End + frag.Position.End
					log.Printf("cutting from %c (%d) to %c (%d)", docBytes[cutStart], cutStart, docBytes[cutEnd], cutEnd)
					docBytes = append(docBytes[:cutStart], docBytes[cutEnd:]...)

					docBytes = append(docBytes[:cutStart], append([]byte(value), docBytes[cutStart:]...)...)

					frag.Run.Text.EndTag.Start += deltaLength
					frag.Run.Text.EndTag.End += deltaLength
					frag.Run.EndTag.Start += deltaLength
					frag.Run.EndTag.End += deltaLength
					frag.Position.End += deltaLength

					log.Printf("full run after replace and adjust: %s", docBytes[frag.Run.StartTag.Start:frag.Run.EndTag.End])

					r.document = docBytes
					r.ReplaceCount++
					r.BytesChanged += deltaLength

					// adjust all following fragments
					r.adjustFragments(frag, deltaLength)

					continue
				} else {
					r.cutFragment(placeholder.Fragments[i])
				}

				// adjust for the change of the first fragment
				/*
				frag.Run.StartTag.Start += deltaLength
				frag.Run.StartTag.End += deltaLength
				frag.Run.EndTag.Start += deltaLength
				frag.Run.EndTag.End += deltaLength
				frag.Run.Text.StartTag.Start += deltaLength
				frag.Run.Text.StartTag.End += deltaLength
				frag.Run.Text.EndTag.Start += deltaLength
				frag.Run.Text.EndTag.End += deltaLength

				 */

				/*
				cutStart := frag.Run.Text.StartTag.End + frag.Position.Start
				cutEnd := frag.Run.Text.StartTag.End + frag.Position.End
				cutLength := frag.Position.End - frag.Position.Start
				runStart := frag.Run.StartTag.Start
				runEnd := frag.Run.EndTag.End

				log.Printf("X full run before cut: %s", docBytes[runStart:runEnd])
				log.Printf("X cutting from %c (%d) to %c (%d) - delta: %d", docBytes[cutStart], cutStart, docBytes[cutEnd], cutEnd, cutLength)
				docBytes = append(docBytes[:cutStart], docBytes[cutEnd:]...)

				// adjust fragment after cut
				frag.Run.Text.EndTag.Start -= cutLength
				frag.Run.Text.EndTag.End -= cutLength
				frag.Run.EndTag.Start -= cutLength
				frag.Run.EndTag.End -= cutLength
				frag.Position.End = frag.Position.Start

				 */

				// all other fragments: remove the placeholder text fragment
			}

			// modify all other fragments
			/*
				if len(placeholder.Fragments) > 1 {
					// handle all fragments but the first
					for i := 1; i < len(placeholder.Fragments); i++ {
						log.Printf("fragment #%d to be removed: %s", i, placeholder.Fragments[i].Text(r.document))



					}
				}

			*/

			/*
				// replace placeholder with value and inject or remove bytes as needed
				if deltaLen > 0 {
					valStartPos := placeholder.StartPos()
					newEndPos := placeholder.EndPos() - deltaLen



					// find out which fragments need to be adjusted
					affectedFragments := r.findFragmentsFromPos(placeholder.StartPos())
					_, _, _ = valStartPos, newEndPos, affectedFragments

					/*log.Printf("%d bytes need to be removed from %d: was=%s (len=%d) new=%s (len=%d)",
						deltaLen, valStartPos, placeholderKey, len(placeholderKey), value, len(value))

					log.Printf("value starts at %d and ends at %d (was %d)", valStartPos, newEndPos, placeholder.EndPos())
			*/

			// Splice the value into the document bytes, stripping 'deltaLen' bytes since in this
			// case the actual value is shorter than the placeholder.
			// All following byte offsets (runs, fragments,..) need to be shifted left by that amount.
			//
			// Visualization:  	{the_placeholder_key}
			//				 	|			^-------^ deltaLen
			//				 	|			^-- newEndPos
			//				 	^-- valStartPos

			/*
				pre := r.document[:valStartPos]
				post := r.document[:newEndPos]
				newDocBytes :=  append(pre, append([]byte(value), post...)...)
				newDocBytes :=r.document
				_, _ = valStartPos, newEndPos


				// shift all affected positions left by deltaLen
				for _, fragment := range affectedFragments {
					//log.Printf("old fragment pos: [%d:%d] %s", fragment.StartPos(), fragment.EndPos(), fragment.Text(r.document))

					//log.Printf("%d => %d", fragment.Run.StartTag.Start, fragment.Run.StartTag.Start - deltaLen)
					fragment.Run.StartTag.Start -= deltaLen
					fragment.Run.StartTag.End -= deltaLen
					fragment.Run.EndTag.Start -= deltaLen
					fragment.Run.EndTag.End -= deltaLen
					fragment.Position.Start -= deltaLen
					fragment.Position.End -= deltaLen
					// log.Printf("new fragment pos: [%d:%d] %s", fragment.StartPos(), fragment.EndPos(), fragment.Text(newDocBytes))
				}
				r.document = newDocBytes // replace bytes

			*/

			/*
				} else if deltaLen < 0 {
					// todo: remove bytes
					log.Printf("%d bytes need to be injected from %d: was=%s (len=%d) new=%s (len=%d)",
						deltaLen * -1, placeholder.EndPos(), placeholderKey, len(placeholderKey), value, len(value))
				}

				/*
				i := 40
				r.document = append(r.document[:i], append(make([]byte, 1), r.document[:i]...)...)
			*/

		}
	}

	// find out which fragments need to be removed

	return nil
}

// curFragment will remove the fragment text from the document
func (r *Replacer) cutFragment(fragment *PlaceholderFragment) {
	docBytes := r.document
	cutStart := fragment.Run.Text.StartTag.End + fragment.Position.Start
	cutEnd := fragment.Run.Text.StartTag.End + fragment.Position.End
	cutLength := fragment.Position.End - fragment.Position.Start

	// preserve old pos
	runStart := fragment.Run.StartTag.Start
	runEnd := fragment.Run.EndTag.End

	log.Printf("cutFragment: full run before cut: %s", docBytes[runStart:runEnd])
	docBytes = append(docBytes[:cutStart], docBytes[cutEnd:]...)
	log.Printf("cutFragment: from %c (%d) to %c (%d) - delta: %d", docBytes[cutStart], cutStart, docBytes[cutEnd], cutEnd, cutLength)

	// adjust fragment after cut
	fragment.Run.Text.EndTag.Start -= cutLength
	fragment.Run.Text.EndTag.End -= cutLength
	fragment.Run.EndTag.Start -= cutLength
	fragment.Run.EndTag.End -= cutLength
	fragment.Position.End = fragment.Position.Start

	log.Printf("cutFragment: full run after cut: %s", docBytes[fragment.Run.StartTag.Start:fragment.Run.EndTag.End])
	log.Printf("cutFragment: adjusted %s", fragment.String(docBytes))

	// adjust all following fragments
	r.adjustFragments(fragment, -cutLength)
	r.document = docBytes
	r.BytesChanged -= cutLength
}

func (r *Replacer) findFragmentsInRun(run *Run) (fragments []*PlaceholderFragment) {
	for _, placeholder := range r.findRunPlaceholders(run) {
		for _, fragment := range placeholder.Fragments {
			if fragment.Run == run {
				fragments = append(fragments, fragment)
			}
		}
	}
	return fragments
}

func (r *Replacer) adjustFragments(fromFragment *PlaceholderFragment, deltaLength int64) {

	// handle all fragments which share a run with the given fragment
	sharedRunFragments := r.findFragmentsInRun(fromFragment.Run)
	for _, frag := range sharedRunFragments {
		if frag == fromFragment {
			continue // ignore the fromFragment. It is expected to be correct already.
		}
		fragment := frag

		// If fromFragment is actually after the fragment => skip
		if fromFragment.Position.Start > frag.Position.Start {
			continue
		}

		// The fragment is in the same run as fromFragment.
		// The run of fromFragment is expected to be correct already, so we do not need to shift the run positions.
		fragment.Position.Start += deltaLength
		fragment.Position.End += deltaLength
	}

	followingFragments := r.findFragmentsFromPos(fromFragment.Run.Text.StartTag.End + fromFragment.Position.End)

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

	for _, frag := range followingFragments {
		followingFragment := frag
		followingFragment.Run.StartTag.Start += deltaLength
		followingFragment.Run.StartTag.End += deltaLength
		followingFragment.Run.EndTag.Start += deltaLength
		followingFragment.Run.EndTag.End += deltaLength
		followingFragment.Run.Text.StartTag.Start += deltaLength
		followingFragment.Run.Text.StartTag.End += deltaLength
		followingFragment.Run.Text.EndTag.Start += deltaLength
		followingFragment.Run.Text.EndTag.End += deltaLength
	}
}

func (r *Replacer) findFragmentsFromPos(startingFrom int64) (found []*PlaceholderFragment) {
	for _, placeholder := range r.placeholders {
		for _, fragment := range placeholder.Fragments {
			if fragment.Run.StartTag.Start > startingFrom {
				found = append(found, fragment)
				continue
			}
		}
	}
	return found
}
