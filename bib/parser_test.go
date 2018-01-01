// (c) 2018 by Carl Kingsford (carlk@cs.cmu.edu). See LICENSE.txt.
package bib

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestParser(t *testing.T) {
	const in = `@Article{moo,
        title = {Chess playing {Masters}},
        data-added = "now",
        author = "Carl Kingsford and Henry Kingsford",
    }
    @PhDThesis{moo2,
        school = "NYU",
        journal = nbt,
        year = 1794
        author = "Art van der Lay",
    }
    @article{Trieu,
  author = {Trieu, Tuan and Cheng, Jianlin},
  date-added = {2015-01-26 18:09:19 +0000},
  date-modified = {2015-01-26 18:09:19 +0000},
  journal = {Nucleic Acids Research},
  number = {7},
  pages = {e52--e52},
  publisher = {Oxford Univ Press},
  title = {{Large-scale reconstruction of 3D structures of human chromosomes from chromosomal contact data}},
  volume = {42},
  year = {2014},
}
    these are ignored
    `
	p := NewParser(strings.NewReader(in))
	db := p.ParseBibTeX()
	if p.NErrors() > 0 {
		p.PrintErrors(os.Stderr)
	}

	db.WriteDatabase(os.Stdout)
	p.PrintErrors(os.Stdout)
}

func TestSplitOnTopLevel(t *testing.T) {
	const in = "now and whyand and noandway {and} {there and what} an{d} test and"
	for i, w := range splitOnTopLevelString(in, "and", true) {
		fmt.Printf("%d [%s]\n", i, w)
	}
}

func TestSplit(t *testing.T) {
	const in = "hi {there word} this is"
	for i, v := range splitOnTopLevel(in) {
		fmt.Println(i, v)
	}
}

func TestVon(t *testing.T) {
	in := []string{"von", "Moo", "{de}", "Moo", "Berry"}
	v, l := parseVon(in)
	fmt.Printf("von = {%s} last = {%s}\n", v, l)
}

func TestParseName(t *testing.T) {
	const in = "Henry von de Moo"
	f, v, l := parseNameParts(in)
	fmt.Printf("f={%s} v={%s} l={%s}\n", f, v, l)
}

func TestNormalizeName(t *testing.T) {
	const in = "von de Moo, Jr., Henry"
	a := NormalizeName(in)
	fmt.Printf("f={%s} v={%s} l={%s} j={%s}\n", a.First, a.Von, a.Last, a.Jr)
}

func TestBraces(t *testing.T) {
	const in = "{hi {there this {is a nested} set}{or sets} of {strings}}"
	//const in = "Hi there is this a normal string"
	b, l := ParseBraceTree(in)
	fmt.Printf("lens = %d read = %d\n", len(in), l)
	b.PrintBraceTree(0)
	fmt.Printf("STRING = \"%s\"\n", b.Flatten())
}

// IsEntireStringBraces returns true iff the string looks like: ^{.......}$
// where those those two braces are matched.  It will return FALSE, for
// example, with this string {foo}~{bar}~{baz} or {foo}{bar}{baz}

func IsEntireStringBracedAlternative(s string) bool {
	nbraces := 0
	for _, r := range s {
		switch r {
		case '{':
			nbraces++
		case '}':
			nbraces--
		default:
			if nbraces == 0 {
				return false
			}
		}
	}
	return true
}

func TestSplitWords(t *testing.T) {
	const in = "hi there   moo man   \t  what"
	s := splitWords(in)
	for i, w := range s {
		fmt.Printf("%d: \"%s\"\n", i, w)
	}
}

func TestFlattenToMinBraces(t *testing.T) {
	//const in = "{hi there {mRNA} methods n\\\"eed {To be} Preserved nOW}"
	const in = "RiboGalaxy: A browser based platform for the alignment, analysis and visualization of ribosome profiling data"
	//const in = "Now is the moo time"
	bn, _ := ParseBraceTree(in)
	bn.PrintBraceTree(0)
	fmt.Println(bn.FlattenToMinBraces())
}

func TestIsStrangeCase(t *testing.T) {
	const in = "nOW"
	fmt.Println(IsStrangeCase(in))
}

func TestBadAuthors(t *testing.T) {
	const in = "Smart J.H."
	a := NormalizeName(in)
	//fmt.Printf("(%s)(%s)(%s)\n", f, v, l)
	fmt.Printf("%v\n", a)
}
