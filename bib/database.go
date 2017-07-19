package bib

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
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

// Author represents an Author and the various named parts
type Author struct {
	Others bool
	First  string
	Last   string
	Von    string
	Jr     string
}

// Entry represents some publication or entry in the database
type Entry struct {
	Kind        EntryKind
	EntryString string
	Key         string
	Fields      map[string]*Value
	AuthorList  []*Author
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
}

// NewDatabase creates a new empty database
func NewDatabase() *Database {
	return &Database{
		Pubs:     make([]*Entry, 0),
		Symbols:  make(map[string]*Value),
		Preamble: make([]string, 0),
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
			bn, size := ParseBraceTree(v.S)
			if v.T == StringType && tag != "author" {
				if size == len(v.S) {
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

func (db *Database) FixSingleHyphenInPages() {
	dash := regexp.MustCompile(`([[:digit:]])-([[:digit:]])`)
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
