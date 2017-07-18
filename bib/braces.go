package bib

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

type BraceNode struct {
	Children []*BraceNode
	Leaf     string
}

func ParseBraceTree(s string) (*BraceNode, int) {

	// walk thru runes, if

	me := &BraceNode{
		Children: make([]*BraceNode, 0),
	}
	accum := ""
	saveAccum := func() {
		if len(accum) > 0 {
			me.Children = append(me.Children, &BraceNode{Leaf: accum})
			accum = ""
		}
	}
	i := 0
	for i < len(s) {
		r, iskip := utf8.DecodeRuneInString(s[i:])
		switch r {
		case '{':
			saveAccum()
			c, nread := ParseBraceTree(s[i+iskip:])
			me.Children = append(me.Children, c)
			iskip += nread
		case '}':
			saveAccum()
			return me, i + iskip
		default:
			accum += string(r)
		}
		i += iskip
	}
	saveAccum()
	return me, i
}

func printIndent(indent int) {
	for indent > 0 {
		fmt.Print(" ")
		indent--
	}
}

func (bn *BraceNode) IsLeaf() bool {
	return len(bn.Children) == 0
}

func (bn *BraceNode) IsEntireStringBraced() bool {
	return len(bn.Children) == 1 && !bn.Children[0].IsLeaf()
}

func (bn *BraceNode) Flatten() string {
	return bn.flatten(true)
}

func (bn *BraceNode) flatten(isroot bool) string {
	if bn.IsLeaf() {
		return bn.Leaf
	} else {
		words := make([]string, 0)
		for _, c := range bn.Children {
			words = append(words, c.flatten(false))
		}
		if isroot {
			return strings.Join(words, "")
		} else {
			return "{" + strings.Join(words, "") + "}"
		}
	}
}

func PrintBraceTree(b *BraceNode, indent int) {
	printIndent(indent)
	if b.Leaf != "" {
		fmt.Printf("LEAF \"%s\"\n", b.Leaf)
	} else {
		fmt.Println("NODE")
		for _, c := range b.Children {
			PrintBraceTree(c, indent+2)
		}
	}
}

func splitWords(s string) []string {
	words := make([]string, 0)
	word := ""
	c, _ := utf8.DecodeRuneInString(s)
	inspace := unicode.IsSpace(c)
	for _, r := range s {
		if inspace && unicode.IsSpace(r) {
			word += string(r)
		} else if !inspace && !unicode.IsSpace(r) {
			word += string(r)
		} else {
			inspace = unicode.IsSpace(r)
			if word != "" {
				words = append(words, word)
			}
			word = string(r)
		}
	}
	if word != "" {
		words = append(words, word)
	}
	return words
}

func (bn *BraceNode) ContainsNoBraces() bool {
    return len(bn.Children) == 1 && bn.Children[0].IsLeaf()
}

func (bn *BraceNode) FlattenToMinBraces() string {
	// only modify if it looks like the user didn't really think about
	// the braces (the most common case)
    var children []*BraceNode = nil
	if bn.IsEntireStringBraced() {
        children = bn.Children[0].Children
    } else if bn.ContainsNoBraces() {
        children = bn.Children
    }

    if children != nil {
		words := make([]string, 0)

		for _, c := range children {
			// for leaf children, we iterate through the words
			if c.IsLeaf() {
				for _, w := range splitWords(c.Leaf) {
					if IsStrangeCase(w) || HasEscape(w) {
						words = append(words, "{"+w+"}")
					} else {
						words = append(words, w)
					}
				}
				// for non-leaf children, we just flatten as normal
			} else {
				words = append(words, c.flatten(false))
            }
			
		}
		return strings.Join(words, "")

	} else {
		return bn.Flatten()
	}
}

// split s on occurances of sep that are not contained in a { } block. If whitespace is
// true, the it requires that the string be surrounded by whitespace (or the string
// boundaries)
func splitOnTopLevelString(s, sep string, whitespace bool) []string {
	nbraces := 0
	lastend := 0
	split := make([]string, 0)

	following := ' '
	prevchar := ' '

	for i, r := range s {
		switch r {
		case '{':
			nbraces++
		case '}':
			nbraces--
		}
		// if we're not in a nested brace and we have a match
		if nbraces == 0 && i >= lastend && i < len(s)-len(sep)+1 && strings.ToLower(s[i:i+len(sep)]) == sep {
			// get the run following the end of the match
			following = ' '
			if i+len(sep) < len(s)-1 {
				following, _ = utf8.DecodeRuneInString(s[i+len(sep):])
			}

			// either we dont' care about whitespace or
			// the match is surrounded by whitespace on each side
			if !whitespace || (unicode.IsSpace(prevchar) && unicode.IsSpace(following)) {
				// then save the split
				split = append(split, s[lastend:i])
				lastend = i + len(sep)
			}

		}
		prevchar = r
	}
	last := s[lastend:]
	split = append(split, last)

	return split
}

// splitOnTopLevel splits s on whitespace separated words, but treats {}-deliminated substrings
// as a unit
func splitOnTopLevel(s string) []string {
	nbraces := 0
	word := ""
	words := make([]string, 0)
	s = strings.TrimSpace(s)
	for _, r := range s {
		switch r {
		case '{':
			nbraces++
		case '}':
			nbraces--
		}
		// if we are at a top-level space
		if nbraces == 0 && unicode.IsSpace(r) {
			// and there is a current word, save it
			if len(word) > 0 {
				words = append(words, word)
				word = ""
			}
			// if we are in {} or non-space, add to current word
		} else {
			word += string(r)
		}
	}
	if len(word) > 0 {
		words = append(words, word)
	}
	return words
}
