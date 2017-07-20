package bib

import (
	"fmt"
	"io"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

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

type BibTeXError struct {
	BadEntry *Entry
	Tag      string
	Msg      string
}

func (db *Database) addError(e *Entry, tag string, msg string) {
	db.Errors = append(db.Errors, &BibTeXError{
		BadEntry: e,
		Tag:      tag,
		Msg:      msg,
	})
}

func (db *Database) PrintErrors(w io.Writer) {
	for _, er := range db.Errors {
		if er.Tag != "" {
			fmt.Fprintf(w, "warn: %s %s: %s\n", er.BadEntry.Key, er.Tag, er.Msg)
		} else {
			fmt.Fprintf(w, "warn: %s: %s\n", er.BadEntry.Key, er.Msg)
		}
	}
}

// returns true if v1 < v2
func (v1 *Value) Less(v2 *Value) bool {
	if (v1.T == StringType || v1.T == SymbolType) && (v2.T == StringType || v2.T == SymbolType) {
		return v1.S < v2.S
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
				ans = v1.Less(v2)
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

// RemoveWholeFieldBraces removes the braces from fields that look like:
// {{foo bar baz}}
func (db *Database) RemoveWholeFieldBraces() {
	db.TransformEachField(
		func(tag string, v *Value) *Value {
			// we only transform non-author, string-type fields
			if v.T == StringType && tag != "author" {
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

func (db *Database) CheckField(tag string, check func(*Value) string) {
	for _, e := range db.Pubs {
		if v, ok := e.Fields[tag]; ok {
			if msg := check(v); msg != "" {
				db.addError(e, tag, msg)
			}
		}
	}
}

func (db *Database) CheckYearsAreInt() {
	db.CheckField("year",
		func(v *Value) string {
			if v.T == StringType {
				return fmt.Sprintf("year is not an integer \"%s\"", v.S)
			} else {
				return ""
			}
		})
}

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

func (db *Database) CheckAllFields(check func(string, *Value) string) {
	for _, e := range db.Pubs {
		for tag, value := range e.Fields {
			if msg := check(tag, value); msg != "" {
				db.addError(e, tag, msg)
			}
		}
	}
}

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
				return fmt.Sprintf("symbol \"%s\" is undefined", v.S)
			}
			return ""
		})
}

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
		db.addError(e, "", fmt.Sprintf("key \"%s\" is defined more than once", e.Key))
	}
}
