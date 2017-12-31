package bib

import (
	"fmt"
	"io"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

var titleLowerWords = []string{"the", "a", "an", "but", "for", "and", "or",
	"nor", "to", "from", "on", "in", "of", "at", "by"}

// FieldType represents the type of a data entry
type FieldType int

const (
	StringType FieldType = iota
	NumberType
	SymbolType
)

// Value is the value of an item in an entry
type Value struct {
	T FieldType
	S string
	I int
}

// BibTeXError holds an error found in a bibtex file
type BibTeXError struct {
	BadEntry *Entry
	Tag      string
	Msg      string
}

// addError adds an error to the list of reported errors
func (db *Database) addError(e *Entry, tag string, msg string) {
	db.Errors = append(db.Errors, &BibTeXError{
		BadEntry: e,
		Tag:      tag,
		Msg:      msg,
	})
}

// PrintErrors writes all the saved errors to the `w` stream.
func (db *Database) PrintErrors(w io.Writer) {
	byKey := make(map[string][]string)
	keys := make([]string, 0)

	for _, er := range db.Errors {
		key := "<none>"
		line := 0
		if er.BadEntry != nil {
			key = er.BadEntry.Key
			line = er.BadEntry.LineNo
		}
		if _, ok := byKey[key]; !ok {
			byKey[key] = make([]string, 0)
			keys = append(keys, key)
		}

		var msg string
		if er.Tag != "" {
			msg = fmt.Sprintf("%d:%s: %s", line, er.Tag, er.Msg)
		} else {
			msg = fmt.Sprintf("%d: %s", line, er.Msg)
		}
		byKey[key] = append(byKey[key], msg)
	}

	sort.Strings(keys)
	for _, k := range keys {
		fmt.Fprintf(w, "Key %q:\n", k)
		for _, msg := range byKey[k] {
			fmt.Fprintf(w, "  %s\n", msg)
		}
		fmt.Fprintln(w)
	}

}

// Expands symbol definitions to their defined values; if an expansion expands
// to a symbol itself, then that symbol will be expanded and so on, up to
// `depth` recursions.  If an undefined symbol is found, we return the
// unexpanded symbol (at that point). If we exceed the depth, we return where
// we got to for that depth. These rules mean that the function is a no-op for
// non-Symbols and undefined Symbols.
func (db *Database) SymbolValue(symb *Value, depth int) *Value {
	i := 0
	// repeat until either we hit a non-symbol, or exceed our recursion depth
	for i < depth && symb.T == SymbolType {
		// if this is a user-defined symbol, do the substituion
		s := strings.ToLower(symb.S)
		if v, ok := db.Symbols[s]; ok {
			symb = v
			i++

			// if we are a predefined symbol, we're done
		} else if v, ok := predefinedSymbols[s]; ok {
			return &Value{T: StringType, S: v}
		} else {
			return symb
		}
	}
	return symb
}

// returns true if v1 < v2
func (db *Database) Less(v1 *Value, v2 *Value) bool {

	// expand the symbols, if appropriate (nop otherwise)
	v1 = db.SymbolValue(v1, 10)
	v2 = db.SymbolValue(v2, 10)

	if (v1.T == StringType || v1.T == SymbolType) && (v2.T == StringType || v2.T == SymbolType) {
		bt1, _ := ParseBraceTree(v1.S)
		bt2, _ := ParseBraceTree(v2.S)
		return bt1.FlattenForSorting() < bt2.FlattenForSorting()
	}

	if v1.T == NumberType && v2.T == NumberType {
		return v1.I < v2.I
	}

	vs1 := v1.S
	if v1.T == NumberType {
		vs1 = strconv.Itoa(v1.I)
	}

	vs2 := v2.S
	if v2.T == NumberType {
		vs2 = strconv.Itoa(v2.I)

	}

	return vs1 < vs2
}

// Equals returns true if v1 == v2
func (v1 *Value) Equals(v2 *Value) bool {
	if v1.T == v2.T {
		switch v1.T {
		case StringType, SymbolType:
			return v1.S == v2.S
		case NumberType:
			return v1.I == v2.I
		}
	}
	return false
}

// Author represents an Author and the various named parts
type Author struct {
	Others bool
	First  string
	Last   string
	Von    string
	Jr     string
}

// Returns true iff an author structure is exactly equal to another
func (a *Author) Equals(b *Author) bool {
	return *a == *b
}

// Entry represents some publication or entry in the database
type Entry struct {
	Kind        EntryKind
	EntryString string
	Key         string
	Fields      map[string]*Value
	AuthorList  []*Author
	LineNo      int
}

// IsSubset returns true if this entnry is a subset of the given one. An e1 is
// a subset of e2 if they (a) have the same type, and (b) every field in e1
// appears in e2 with the exact same value
func (e1 *Entry) IsSubset(e2 *Entry) bool {
	if e1.Kind != e2.Kind || strings.ToLower(e1.EntryString) != strings.ToLower(e2.EntryString) {
		return false
	}

	for tag, value1 := range e1.Fields {
		if value2, ok := e2.Fields[tag]; !ok || !value1.Equals(value2) {
			return false
		}
	}
	return true
}

// Equals returns true if the two entries are equal. Entries e1 and e2 are
// equal if they are the same Kind, have the exact same set of tags, and have
// exactly the same values for each tag. Note that the entrie's KEY does not
// play a role in Equality testing, nor does the parsed author list -- two
// entries would be unequal if they had the same author list encoded
// differently.
func (e1 *Entry) Equals(e2 *Entry) bool {
	return e1.IsSubset(e2) && e2.IsSubset(e1)
}

// newEntry creates a new empty entry
func newEntry() *Entry {
	return &Entry{
		Fields: make(map[string]*Value),
	}
}

// Tags() returns the list of tag names of the entry's fields
func (e *Entry) Tags() []string {
	tags := make([]string, len(e.Fields))
	i := 0
	for k := range e.Fields {
		tags[i] = k
		i++
	}
	sort.Strings(tags)
	return tags
}

// Database is a collection of entries plus symbols and preambles
type Database struct {
	Pubs     []*Entry
	Symbols  map[string]*Value
	Preamble []string
	Errors   []*BibTeXError
}

// NewDatabase creates a new empty database
func NewDatabase() *Database {
	return &Database{
		Pubs:     make([]*Entry, 0),
		Symbols:  make(map[string]*Value),
		Preamble: make([]string, 0),
		Errors:   make([]*BibTeXError, 0),
	}
}

/*===============================================================================
 * Sorting
 *==============================================================================*/

type pubSorter struct {
	pubs []*Entry
	by   func(*Entry, *Entry) bool
}

func (ps *pubSorter) Len() int {
	return len(ps.pubs)
}

func (ps *pubSorter) Less(i, j int) bool {
	return ps.by(ps.pubs[i], ps.pubs[j])
}

func (ps *pubSorter) Swap(i, j int) {
	ps.pubs[i], ps.pubs[j] = ps.pubs[j], ps.pubs[i]
}

// SortByField sorts the database by the given field. Missing fields will come
// before non-missing fields. Within the missing fields, entries will be sorted
// by Key. Uses the value.Less function to determine the order. If reverse is
// true, this order will be reversed.
func (db *Database) SortByField(field string, reverse bool) {
	ps := &pubSorter{
		pubs: db.Pubs,
		by: func(e1, e2 *Entry) bool {
			v1, ok1 := e1.Fields[field]
			v2, ok2 := e2.Fields[field]
			ans := false
			switch {
			case !ok1 && ok2:
				ans = true
			case ok1 && !ok2:
				ans = false
			case !ok1 && !ok2:
				ans = (e1.Key < e2.Key)
			case ok1 && ok1:
				ans = db.Less(v1, v2)
			}

			if reverse {
				ans = !ans
			}
			return ans
		},
	}
	sort.Sort(ps)
}

// NormalizeAuthors parses every listed author and puts them into normal form.
// It also populates the Authors field of each entry with the list of *Authors.
// Call this function before working with the Authors field.
func (db *Database) NormalizeAuthors() {
	for _, e := range db.Pubs {
		// if there is an author field that is a string
		if authors, ok := e.Fields["author"]; ok && authors.T == StringType {
			// normalize each name
			e.AuthorList = make([]*Author, 0)
			names := make([]string, 0)

			a := splitOnTopLevelString(authors.S, "and", true)
			for _, name := range a {
				auth := NormalizeName(name)
				if auth != nil {
					e.AuthorList = append(e.AuthorList, auth)
					names = append(names, auth.String())
				}
			}

			e.Fields["author"].S = strings.Join(names, " and ")
		}
	}
}

// TransformEachField is a helper function that applies the parameter trans
// to every tag/value pair in the database
func (db *Database) TransformEachField(trans func(string, *Value) *Value) {
	for _, e := range db.Pubs {
		for tag, value := range e.Fields {
			e.Fields[tag] = trans(tag, value)
		}
	}
}

// TransformField applies the given transformation to each field named "tag"
func (db *Database) TransformField(tag string, trans func(string, *Value) *Value) {
	for _, e := range db.Pubs {
		if value, ok := e.Fields[tag]; ok {
			e.Fields[tag] = trans(tag, value)
		}
	}
}

// ConvertIntStringsToInt looks for values that are marked as strings but that
// are really integers and converts them to ints. This happens when a bibtex
// file has, e.g., volume = {9} instead of volume = 9
func (db *Database) ConvertIntStringsToInt() {
	db.TransformEachField(
		func(tag string, value *Value) *Value {
			if value.T == StringType {
				if i, err := strconv.Atoi(value.S); err == nil {
					value.T = NumberType
					value.I = i
				}
			}
			return value
		})
}

// NoramalizeWhitespace replaces errant whitespace with " " characters
func (db *Database) NormalizeWhitespace() {
	spaces := regexp.MustCompile(" +")
	db.TransformEachField(
		func(tag string, value *Value) *Value {
			if value.T == StringType {
				value.S = strings.Replace(value.S, "\n", " ", -1)
				value.S = strings.Replace(value.S, "\t", " ", -1)
				value.S = strings.Replace(value.S, "\x0D", " ", -1)
				value.S = strings.TrimSpace(value.S)
				value.S = spaces.ReplaceAllString(value.S, " ")
			}
			return value
		})
}

// ReplaceSymbols tries to find strings that are uniquely represented by a symbol and
// replaces the use of those strings with the symbol. It requires (a) that the
// strings match exactly and (b) there is only 1 symbol that matches the string.
func (db *Database) ReplaceSymbols() {
	symbols := make(map[string]string)
	for k, v := range predefinedSymbols {
		symbols[k] = v
	}
	for k, v := range db.Symbols {
		if v.T == StringType {
			symbols[k] = v.S
		}
	}

	inverted := make(map[string]string)
	for sym, val := range symbols {
		// if we define it twice, we can't invert
		if _, ok := inverted[val]; ok {
			inverted[val] = ""
		} else {
			inverted[val] = sym
		}
	}
	db.TransformEachField(
		func(tag string, value *Value) *Value {
			if value.T == StringType {
				if sym, ok := inverted[value.S]; ok && sym != "" {
					value.T = SymbolType
					value.S = sym
				}
			}
			return value
		})
}

// Find month fields that contain abbreviated text or numbers
// Note that we don't need to handle fields that list the full month name since
// those are handled by the pre-defined symbols
func (db *Database) ReplaceAbbrMonths() {
	months := map[string]string{
		"jan":  "jan",
		"feb":  "feb",
		"mar":  "mar",
		"apr":  "apr",
		"may":  "may",
		"jun":  "jun",
		"jul":  "jul",
		"aug":  "aug",
		"sep":  "sep",
		"sept": "sep",
		"oct":  "oct",
		"nov":  "nov",
		"dec":  "dec",
	}
	monthsnum := []string{"jan", "feb", "mar", "apr", "may", "jun", "jul", "aug", "sep", "oct", "nov", "dec"}
	db.TransformField("month",
		func(tag string, value *Value) *Value {
			if value.T == StringType {
				if sym, ok := months[strings.ToLower(value.S)]; ok {
					value.T = SymbolType
					value.S = sym
				}
			} else if value.T == NumberType {
				if 1 <= value.I && value.I <= 12 {
					value.T = SymbolType
					value.S = monthsnum[value.I-1]
				}
			}
			return value
		})
}

// RemoveNonBlessedFields removes any field that isn't 'blessed'. The blessed
// fields are (a) any fields in the required or optional global variables, any
// that are in the blessed global variable, plus any fields listed in the
// additional parameter.
func (db *Database) RemoveNonBlessedFields(additional []string) {
	blessedFields := make(map[string]bool, 0)

	for _, f := range required {
		for _, r := range f {
			for _, s := range strings.Split(r, "/") {
				blessedFields[s] = true
			}
		}
	}

	for _, f := range optional {
		for _, r := range f {
			blessedFields[r] = true
		}
	}

	for _, f := range blessed {
		blessedFields[f] = true
	}

	for _, f := range additional {
		blessedFields[f] = true
	}

	// Remove the fields that are not in the blessed map
	for _, e := range db.Pubs {
		for tag := range e.Fields {
			if _, ok := blessedFields[tag]; !ok {
				delete(e.Fields, tag)
			}
		}
	}
}

// RemoveEmptyField removes string fields whose value is the empty string
func (db *Database) RemoveEmptyFields() {
	for _, e := range db.Pubs {
		for tag, value := range e.Fields {
			if value.T == StringType && value.S == "" {
				delete(e.Fields, tag)
			}
		}
	}
}

/*
foo moo{bar}moo  -> foo {moo{bar}moo} 

{moo bar} -> {moo bar}

{m{oo} bar} -> {{mo{oo}} bar}

{m{oo bar}}

moo-CHILD-moo-CHILD-moo -> 

*/



// BraceQuotes replaces any word foo"bar with {foo"bar"}. the most common
// situation is foo\"{e}bar. Note that word here is defined as a whitespace
// separated string of chars. We do *not* take into account the {} structure
// so: {hi\"{e} there} because {{hi\"{e}} there}
func (db *Database) CanonicalBrace() {
    db.TransformEachField(
        func(tag string, v *Value) *Value {
            if v.T == StringType && tag != "author" {
                v.S = canonicalBrace(v.S)
            }
            return v
        })
}

// RemoveWholeFieldBraces removes the braces from fields that look like:
// {{foo bar baz}}
func (db *Database) RemoveWholeFieldBraces() {
	db.TransformEachField(
		func(tag string, v *Value) *Value {
			// we only transform non-author, string-type fields
			if v.T == StringType && tag != "author" {
				if bn, size := ParseBraceTree(v.S); size == len(v.S) {
                    if bn.IsEntireStringBraced() {
                        v.S = bn.Children[0].Flatten()
                    } else {
					    v.S = bn.Flatten()
                    }
				}
			}
			return v
		})
}

// ConvertTitlesToMinBraces makes sure that all strange-case words are in {}
// {{foo bar baz}}
func (db *Database) ConvertTitlesToMinBraces() {
	db.TransformEachField(
		func(tag string, v *Value) *Value {
			// we only transform non-author, string-type fields
			if v.T == StringType && (tag == "title" || tag == "booktitle") {
				if bn, size := ParseBraceTree(v.S); size == len(v.S) {
					v.S = bn.FlattenToMinBraces()
				}
			}
			return v
		})
}

// Removes unneeded "." from the end of the titles. The . must be the last character
// and it must be preceeded by a lowercase letter
func (db *Database) RemovePeriodFromTitles() {
	pend := regexp.MustCompile(`([[:lower:]])\.$`)
	db.TransformField("title",
		func(tag string, v *Value) *Value {
			if v.T == StringType {
				v.S = pend.ReplaceAllString(v.S, "$1")
			}
			return v
		})
}

// FixHyphensInPages will replace pages fields that look like NUMBER - NUMBER or
// NUMBER -- NUMBER with NUMBER--NUMBER
func (db *Database) FixHyphensInPages() {
	dash := regexp.MustCompile(`([[:digit:]])\s*-{1,2}\s*([[:digit:]])`)
	db.TransformField("pages",
		func(tag string, v *Value) *Value {
			if v.T == StringType {
				bt, s := ParseBraceTree(v.S)
				if s == len(v.S) && bt.ContainsNoBraces() {
					v.S = dash.ReplaceAllString(v.S, "$1--$2")
				}
			}
			return v
		})
}

// for page fields that match aaaa--bb, and where aaaa and bb are integers,
// replace with aaaa-aabb
func (db *Database) FixTruncatedPageNumbers() {
	pages := regexp.MustCompile(`^(\d+)--(\d+)$`)
	db.TransformField("pages",
		func(tag string, v *Value) *Value {
			if v.T == StringType && pages.MatchString(v.S) {
				ab := pages.FindStringSubmatch(v.S)
				if len(ab) == 3 {
					if len(ab[1]) > len(ab[2]) {
						v.S = fmt.Sprintf("%s--%s", ab[1], ab[1][:len(ab[1])-len(ab[2])]+ab[2])
					}
				}
			}
			return v
		})
}

// toGoodTitle converts a word to title case, meaning the first letter is capitalized
// unless the word is a "small" word
func toGoodTitle(w string) string {

	tlw := make(map[string]bool)
	for _, w := range titleLowerWords {
		tlw[w] = true
	}

	if _, ok := tlw[w]; !ok {
		r, size := utf8.DecodeRuneInString(w)
		w = string(unicode.ToTitle(r)) + w[size:]
	}
	return w
}

// TitleCaseJournalNames converts the journal name so that big words are capitalized
func (db *Database) TitleCaseJournalNames() {
	db.TransformField("journal",
		func(tag string, v *Value) *Value {
			if v.T == StringType {
				// if we can parse the title
				bt, size := ParseBraceTree(v.S)
				if size == len(v.S) {
					// go through each immediate leaf child of the root
					for _, wordNode := range bt.Children {
						if wordNode.IsLeaf() {

							// convert each word to good title case
							leafWords := make([]string, 0)
							for _, w := range splitWords(wordNode.Leaf) {
								leafWords = append(leafWords, toGoodTitle(w))
							}

							// update the leaf node
							wordNode.Leaf = strings.Join(leafWords, "")
						}
					}
					// save the string
					v.S = bt.Flatten()
				}
			}
			return v
		})
}

// RemoveExactDups find entries that are Equal and that have the same Key and deletes one of
// them.
func (db *Database) RemoveExactDups() {

	// bin everything by key (lowercase)
	bins := make(map[string][]*Entry)
	for _, e := range db.Pubs {
		key := strings.ToLower(e.Key)
		if _, ok := bins[key]; !ok {
			bins[key] = make([]*Entry, 0)
		}
		bins[key] = append(bins[key], e)
	}

	// for each bin
	ndel := 0
	for _, entries := range bins {
		// for every pair of entries
		for i := range entries {
			for j := i + 1; j < len(entries); j++ {
				// if i and j are dups
				if entries[i].Equals(entries[j]) {
					entries[i].Kind = Deleted
					ndel++
					break // move to next i
				}
			}
		}
	}

	db.removeDeleted(ndel)
}

// removeDeleted removes the entries marked Kind == Deleted. There must be exactly
// ndel of them (which is passed in for efficiency)
func (db *Database) removeDeleted(ndel int) {
	newPubs := make([]*Entry, len(db.Pubs)-ndel)
	i := 0
	for _, e := range db.Pubs {
		if e.Kind != Deleted {
			newPubs[i] = e
			i++
		}
	}
	db.Pubs = newPubs
}

// RemoveContainedEntries tries to find entries that are contained in others
func (db *Database) RemoveContainedEntries() {

	// bin the entries by title
	byTitle := make(map[string][]*Entry)
	for _, e := range db.Pubs {
		// if e has a string field called title
		if title, ok := e.Fields["title"]; ok {
			if title.T == StringType {
				if _, ok := byTitle[title.S]; !ok {
					byTitle[title.S] = make([]*Entry, 0)
				}
				byTitle[title.S] = append(byTitle[title.S], e)
			}
		}
	}

	// within each title group, check each pair (A,B) to see if A is contained
	// in B. If so, mark it Deleted
	ndel := 0
	for _, entries := range byTitle {
		for i := 0; i < len(entries); i++ {
			for j := i + 1; j < len(entries); j++ {
				if entries[i].IsSubset(entries[j]) {
					entries[i].Kind = Deleted
					ndel++
				} else if entries[j].IsSubset(entries[i]) {
					entries[j].Kind = Deleted
					ndel++
				}
			}
		}
	}

	// remove all the deleted
	db.removeDeleted(ndel)
}

// CheckField is a helper function that checks the `tag` field in entries
// using the given `check` function
func (db *Database) CheckField(tag string, check func(*Value) string) {
	for _, e := range db.Pubs {
		if v, ok := e.Fields[tag]; ok {
			if msg := check(v); msg != "" {
				db.addError(e, tag, msg)
			}
		}
	}
}

// isAllCaps returns true if every letter in the string is a uppercase
func isAllCaps(s string) bool {
	for _, r := range s {
		if unicode.IsLetter(r) && !unicode.IsUpper(r) {
			return false
		}
	}
	return true
}

// isAllLower returns true iff every letter in the string is a lowercase letter
func isAllLower(s string) bool {
	for _, r := range s {
		if unicode.IsLetter(r) && !unicode.IsLower(r) {
			return false
		}
	}
	return true
}

// CheckAuthorLast checks for authors where the last name parsed like "J H" or "JH" or "J.H."
// or if the last name is all lowercase. Must have called db.NormalizeAuthors(), otherwise
// this is a no-op
func (db *Database) CheckAuthorLast() {
	for _, e := range db.Pubs {
		if e.AuthorList != nil {
			for _, a := range e.AuthorList {
				if a.Others == true {
					continue
				}
				if strings.TrimSpace(a.Last) == "" {
					db.addError(e, "author", fmt.Sprintf("name %v has empty last name", a))
				} else if isAllCaps(a.Last) {
					db.addError(e, "author", fmt.Sprintf("name %v has no lowercase in last name", a))
				} else if isAllLower(a.Last) {
					db.addError(e, "author", fmt.Sprintf("last name in %v is all lowercase", a.Last))
				}
			}
		}
	}

}

// CheckYearsAreInt adds errors if a year is not an integer
func (db *Database) CheckYearsAreInt() {
	db.CheckField("year",
		func(v *Value) string {
			if v.T == StringType {
				return fmt.Sprintf("year is not an integer %q", v.S)
			} else {
				return ""
			}
		})
}

// CheckEtAl reports the error of using "et al" within a author list
func (db *Database) CheckEtAl() {
	etal := regexp.MustCompile(`[eE][tT]\s+[aA][lL]`)
	db.CheckField("author",
		func(v *Value) string {
			if v.T == StringType && etal.MatchString(v.S) {
				return "author contains et al"
			} else {
				return ""
			}
		})
}

// CheckAllFields is a helper that runs the given check function for each field
func (db *Database) CheckAllFields(check func(string, *Value) string) {
	for _, e := range db.Pubs {
		for tag, value := range e.Fields {
			if msg := check(tag, value); msg != "" {
				db.addError(e, tag, msg)
			}
		}
	}
}

// CheckASCII reports errors where non-ASCII are used in any field
func (db *Database) CheckASCII() {
	db.CheckAllFields(
		func(tag string, v *Value) string {
			if v.T == StringType {
				for i, r := range v.S {
					if r > unicode.MaxASCII {
						return fmt.Sprintf("contains non-ascii character '%c' at position %d", r, i)
					}
				}
			}
			return ""
		})
}

// CheckUndefinedSymbols reports symbols that are not defined
func (db *Database) CheckUndefinedSymbols() {
	db.CheckAllFields(
		func(tag string, v *Value) string {
			if v.T == SymbolType {
				ls := strings.ToLower(v.S)
				if _, ok := db.Symbols[ls]; ok {
					return ""
				}
				if _, ok := predefinedSymbols[ls]; ok {
					return ""
				}
				return fmt.Sprintf("symbol %q is undefined", v.S)
			}
			return ""
		})
}

// CheckLoneHyphenInTitle reports errors where - is used when --- is probably meant
func (db *Database) CheckLoneHyphenInTitle() {
	hyphen := regexp.MustCompile(`\s-\s`)
	db.CheckField("title",
		func(v *Value) string {
			if v.T == StringType && hyphen.MatchString(v.S) {
				return "title contains lone \" - \" when --- is probably needed"
			}
			return ""
		})
}

// CheckPageRanges reports errors where a pages looks like X--Y where X > Y
func (db *Database) CheckPageRanges() {
	pages := regexp.MustCompile(`^(\d+)--(\d+)$`)
	db.CheckField("pages",
		func(v *Value) string {
			if v.T == StringType && pages.MatchString(v.S) {
				ab := pages.FindStringSubmatch(v.S)
				if len(ab) == 3 {
					start, err1 := strconv.Atoi(ab[1])
					end, err2 := strconv.Atoi(ab[2])
					if err1 == nil && err2 == nil {
						if start > end {
							return fmt.Sprintf("page range is empty %d--%d", start, end)
						}
					}
				}
			}
			return ""
		})
}

// CheckDuplicateKeys finds entries with duplicate keys
func (db *Database) CheckDuplicateKeys() {
	keys := make(map[string]bool)
	dups := make(map[string]*Entry)
	for _, e := range db.Pubs {
		kl := strings.ToLower(e.Key)
		if _, ok := keys[kl]; ok {
			dups[kl] = e
		}
		keys[kl] = true
	}

	for _, e := range dups {
		db.addError(e, "", fmt.Sprintf("key %q is defined more than once", e.Key))
	}
}

// CheckRequiredFields reports any missing required fields
func (db *Database) CheckRequiredFields() {
	for _, e := range db.Pubs {
		if _, ok := required[e.Kind]; ok {
			for _, req := range required[e.Kind] {
				found := false
				for _, r := range strings.Split(req, "/") {
					if _, ok := e.Fields[r]; ok {
						found = true
						break
					}
				}
				if !found {
					db.addError(e, req,
						fmt.Sprintf("missing required field %q in %s", req, e.Kind))
				}
			}
		}
	}
}

// CheckUnmatchedDollarSigns checks whether a string has an odd number of
// unescaped dollar signs
func (db *Database) CheckUnmatchedDollarSigns() {
	db.CheckAllFields(
		func(tag string, v *Value) string {
			if v.T == StringType {
				ndollar := 0
				escape := false
				for _, r := range v.S {
					switch r {
					case '$':
						if !escape {
							ndollar++
						}
					case '\\':
						escape = !escape
					default:
						escape = false
					}
				}
				if ndollar%2 != 0 {
					return "contains unbalanced $"
				}
			}
			return ""
		})
}

// CheckRedudantSymbols finds groups of @string definitions that define the
// same string
func (db *Database) CheckRedundantSymbols() {
	x := make(map[string][]string)

	for sym, val := range db.Symbols {
		if val.T == StringType {
			if _, ok := x[val.S]; !ok {
				x[val.S] = make([]string, 0)
			}
			x[val.S] = append(x[val.S], sym)
		}
	}

	for repl, syms := range x {
		if len(syms) > 1 {
			db.addError(nil, "", fmt.Sprintf("symbols all define %q: %s",
				repl, strings.Join(syms, ",")))
		}
	}
}

/*=====================================================================================
 * Duplicate Entrie Checking
 *====================================================================================*/

// removeNonLetters removes non-letters from a string (also keeping whitespace)
func removeNonLetters(s string) string {
	w := ""
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsSpace(r) {
			w += string(r)
		}
	}
	return w
}

// titleHash returns a simplified title useful for grouping pubs
func titleHash(e *Entry) string {
	tlw := make(map[string]bool)
	for _, w := range titleLowerWords {
		tlw[w] = true
	}

	// if we have a string title and can parse it
	if titleval, ok := e.Fields["title"]; ok && titleval.T == StringType {
		if bt, size := ParseBraceTree(titleval.S); size == len(titleval.S) {
			words := make([]string, 0)

			for _, w := range strings.Fields(removeNonLetters(bt.FlattenForSorting())) {
				w = strings.ToLower(w)
				if _, ok := tlw[w]; !ok {
					words = append(words, w)
				}
			}
			return strings.Join(words, " ")
		}
	}
	return ""
}

func (db *Database) FindDupsByTitle() map[string][]*Entry {
	H := make(map[string][]*Entry)
	for _, e := range db.Pubs {
		hash := titleHash(e)
		if _, ok := H[hash]; !ok {
			H[hash] = make([]*Entry, 0)
		}
		H[hash] = append(H[hash], e)
	}
	return H
}

func (db *Database) RemoveDupsByTitle() {
	ndel := 0
	for hash, list := range db.FindDupsByTitle() {
		if hash != "" && len(list) > 1 {
			// check all pairs to see if one can be deleted
			for i := 0; i < len(list); i++ {
				for j := i + 1; j < len(list); j++ {
					if list[i].IsSubset(list[j]) {
						list[i].Kind = Deleted
						ndel++
					} else if list[j].IsSubset(list[i]) {
						list[j].Kind = Deleted
						ndel++
					} else {
						fmt.Printf("%s %s are different somehow\n", list[i].Key, list[j].Key)
					}
				}
			}
		}
	}

	// remove all the deleted
	db.removeDeleted(ndel)
}
