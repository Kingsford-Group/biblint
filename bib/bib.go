package bib

import (
	"ckingsford/bibutil/lexer"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

//==================================================================
// BibTeX database structure
//==================================================================

// EntryKind are the known types of BibTeX entries
type EntryKind string

const (
	Deleted       EntryKind = "**DELETED**"
	Other                   = "other"
	String                  = "string"
	Preamble                = "preamble"
	Article                 = "article"
	Book                    = "book"
	Booklet                 = "booklet"
	InBook                  = "inbook"
	InCollection            = "incollection"
	InProceedings           = "inproceedings"
	Manual                  = "manual"
	MastersThesis           = "mastersthesis"
	Misc                    = "misc"
	PhdThesis               = "phdthesis"
	Proceedings             = "proceedings"
	TechReport              = "techreport"
	Unpublished             = "unpublished"
)

// identToKind maps a (lowercase) string into the EntryKind type
var identToKind = map[string]EntryKind{
	"other":         Other,
	"string":        String,
	"preamble":      Preamble,
	"article":       Article,
	"book":          Book,
	"inbook":        InBook,
	"incollection":  InCollection,
	"inproceedings": InProceedings,
	"manual":        Manual,
	"mastersthesis": MastersThesis,
	"misc":          Misc,
	"phdthesis":     PhdThesis,
	"proceedings":   Proceedings,
	"techreport":    TechReport,
	"unpublished":   Unpublished,
}

// required lists the required fields for each EntryKind type
var required = map[EntryKind][]string{
	Other:         []string{},
	String:        []string{},
	Preamble:      []string{},
	Article:       []string{"author", "title", "journal", "year", "volume"},
	Book:          []string{"author/editor", "title", "publisher", "year"},
	Booklet:       []string{"title"},
	InBook:        []string{"author/editor", "title", "chapter/pages", "publisher", "year"},
	InCollection:  []string{"author", "title", "booktitle", "publisher", "year"},
	InProceedings: []string{"author", "title", "booktitle", "year"},
	Manual:        []string{"title"},
	MastersThesis: []string{"author", "title", "school", "year"},
	Misc:          []string{},
	PhdThesis:     []string{"author", "title", "school", "year"},
	Proceedings:   []string{"title", "year"},
	TechReport:    []string{"author", "title", "institution", "year"},
	Unpublished:   []string{"author", "title", "note"},
}

// optional lists the "optional" fields for each EntryKind. Optional fields are those
// that are often used for the entry type but that are not "required"
var optional = map[EntryKind][]string{
	Other:         []string{},
	String:        []string{},
	Preamble:      []string{},
	Article:       []string{"number", "pages", "month"},
	Book:          []string{"volume", "number", "series", "address", "edition", "month"},
	Booklet:       []string{"author", "howpublished", "address", "month", "year"},
	InBook:        []string{"volume", "number", "series", "type", "address", "edition", "month"},
	InCollection:  []string{"editor", "volume", "number", "series", "type", "chapter", "pages", "address", "edition", "month"},
	InProceedings: []string{"editor", "volume", "number", "series", "pages", "address", "month", "organization", "publisher"},
	Manual:        []string{"author", "organization", "address", "edition", "month", "year"},
	MastersThesis: []string{"type", "address", "month"},
	Misc:          []string{"author", "title", "howpublished", "month", "year"},
	PhdThesis:     []string{"type", "address", "month"},
	Proceedings:   []string{"editor", "volume", "number", "series", "address", "month", "publisher", "organization"},
	TechReport:    []string{"type", "number", "address", "month"},
	Unpublished:   []string{"month", "year"},
}

// blessed lists fields that are neither required nor "optional" but that are
// commonly used in bibtex entries. We treat "key" and "note" as blessed
// instead of "optional", since those fields are "optional" for any entry type
var blessed = []string{"key", "note", "url", "doi", "pmc", "pmid", "keywords", "issn", "isbn"}

// predefinedSymbols lists the predefined symbols
var predefinedSymbols = map[string]string{
	"jan": "January",
	"feb": "February",
	"mar": "March",
	"apr": "April",
	"may": "May",
	"jun": "June",
	"jul": "July",
	"aug": "August",
	"sep": "September",
	"oct": "October",
	"nov": "November",
	"dec": "December",
}

// toEntryKind coverts a string to an EntryKind
func toEntryKind(s string) EntryKind {
	if k, ok := identToKind[strings.ToLower(s)]; ok {
		return k
	} else {
		return Other
	}
}

//==================================================================
// Parser
//==================================================================

// ParserError holds a parser error
type ParserError struct {
	err error
	tok *lexer.Token
	msg string
}

/*
 Parser is a parser for BibTeX files. The set of files that the current Parser accepts
 overlaps imperfectly with the set of files that bibtex itself will accept. For example,

    - BibTeX requires every tag = value pair except the last to be followed by
    a comma, while we do not (since this is a common erorr and yet the commas
    are not needed to parse correctly)

    - BibTeX allows either {} or () to be used to deliminate entries and
    strings, e.g.  @article(title="foo") is allowed. We instead require {} to
    be used.

    - We do not yet support the # concatenation operator

    - We accept non-string @strings, e.g. @string(year = 2017) is parsed, while
    for bibtex strings must be strings

    - We, for simplicity, accept @preamble "this is a string", when the
    official syntax is @preamble{this is a string}. This is because in all
    other cases {} and "" strings can be used, but for preambles, {} is
    supposed to be used. Since no one uses @preamble anyway, this is not expected
    to be a problem.
*/

type Parser struct {
	lex            *lexer.Lexer
	errors         []*ParserError
	curToken       *lexer.Token
	peekToken      *lexer.Token
	bracesAsString bool
}

// NewParser creates a new BibTeX parser reading form the given
// io.Reader. [curToken will be the first token in the stream]
func NewParser(f io.Reader) *Parser {
	lex := lexer.New(f)
	p := &Parser{
		lex:    lex,
		errors: make([]*ParserError, 0),
	}
	p.advanceTokens()
	p.advanceTokens()
	return p
}

// peekError records a syntax error detected by a peek operation:
// This is called when we expect expected but got something else in
// the peek location
func (p *Parser) peekError(expected lexer.TokenType) {
	p.addError(fmt.Sprintf("expected %s, got %s instead", expected, p.peekToken.Type))
}

// addError records an error with the message to the parser, which can
// be received with NErrors() and PrintErrors(), etc.
func (p *Parser) addError(msg string) {
	e := &ParserError{
		err: p.lex.Err(),
		tok: p.peekToken,
		msg: msg,
	}
	p.errors = append(p.errors, e)
}

// advanceTokens() advances the tokens by one so that the former peek
// becomes curToken and the new peek is read from the lexer.
func (p *Parser) advanceTokens() error {
	peekToken, err := p.lex.NextToken(p.bracesAsString)
	if err != nil {
		// XXX: Add error reporting here
		return err
	} else {
		p.curToken = p.peekToken
		p.peekToken = peekToken
	}
	return nil
}

// curTokenIs returns true iff the current token is of the given type.
func (p *Parser) curTokenIs(t lexer.TokenType) bool {
	return p.curToken.Type == t
}

// peekTokenIs returns true iff the peek token is of the given type
func (p *Parser) peekTokenIs(t lexer.TokenType) bool {
	return p.peekToken.Type == t
}

// expectPeek asserts that the peek token is of the given type. If it
// is, then we advance so that the peek becomes the current; otherwise
// we record a syntax error
func (p *Parser) expectPeek(t lexer.TokenType) bool {
	if p.peekTokenIs(t) {
		p.advanceTokens()
		return true
	} else {
		p.peekError(t)
		return false
	}
}

// reads a tag/value pair in an entry. We expect a sequence of tokens that look like
//          IDENT = [STRING|IDENT]
// where the first IDENT is in the cur position. Returns true if we read a k/v
// pair successfully, in which case it will have been added to the given Entry.
// In well-formed entry, at end, the current token will be either a "," indicating
// that the k/v pair ended with a , or a } indicating that the entry is over
// NOTE: BibTeX allows the last k/v to end with a "," so you can see " , }" at end
// end of an entry.
func (p *Parser) readTagValue(entry *Entry) bool {
	var tag string
	var v *Value

	tag = strings.ToLower(p.curToken.Literal)

	if !p.expectPeek(lexer.EQUALS) {
		return false
	}

	if p.peekTokenIs(lexer.IDENT) {
		p.advanceTokens()
		if i, err := strconv.Atoi(p.curToken.Literal); err == nil {
			v = &Value{T: NumberType, I: i}
		} else {
			v = &Value{T: SymbolType, S: p.curToken.Literal}
		}

	} else if p.peekTokenIs(lexer.STRING) {
		p.advanceTokens()
		v = &Value{T: StringType, S: p.curToken.Literal}

	} else {
		p.peekError(lexer.STRING)
		return false
	}

	// save the data into the entry
	entry.Fields[tag] = v
	return true
}

// parsePreamble handles parsing @preamble{text text text} entries.
func (p *Parser) parsePreamble() *Entry {
	entry := newEntry()
	p.bracesAsString = true
	entry.Kind = Preamble
	entry.EntryString = p.curToken.Literal
	if !p.expectPeek(lexer.IDENT) {
		return nil
	}
	if !p.expectPeek(lexer.STRING) {
		return nil
	}
	entry.Key = p.curToken.Literal
	p.bracesAsString = false
	return entry
}

// parseEntry reads an @type{tag=value, tag=value, ...} entry It will handle
// @string and @preamble correctly Note that for @string, this function
// supports the non-BibTeX syntax of allowing several symbols to be defined in
// one @string. It will return the symbols as tag/value pairs in Fields
func (p *Parser) parseEntry() *Entry {

	// must handle @preamble specially b/c (a) it has no tag = value pairs and (b) it
	// must be treated as a string
	if p.peekTokenIs(lexer.IDENT) && toEntryKind(p.peekToken.Literal) == Preamble {
		return p.parsePreamble()
	}

	// @ IDENT { IDENT , [IDENT = [STRING|IDENT] COMMA]* }
	entry := newEntry()

	// read the entry type, which should be followed by a LBRACE
	if !p.expectPeek(lexer.IDENT) {
		return nil
	}
	entry.Kind = toEntryKind(p.curToken.Literal)
	entry.EntryString = p.curToken.Literal

	if !p.expectPeek(lexer.LBRACE) {
		return nil
	}

	if entry.Kind != String {
		// read the key
		if !p.expectPeek(lexer.IDENT) {
			return nil
		}
		entry.Key = p.curToken.Literal

		// get to the entries, which should be preceeded by a COMMA
		if !p.expectPeek(lexer.COMMA) {
			return nil
		}
	}

	// now that we are inside the tag/value pairs, '{' characters
	// start strings
	p.bracesAsString = true
	defer func() { p.bracesAsString = false }()

	// until we find the end of the entry
	for !p.peekTokenIs(lexer.RBRACE) {

		// expect to find an INDENT
		if !p.expectPeek(lexer.IDENT) {
			return nil
		}

		// read the current IDENT plus = VALUE
		if !p.readTagValue(entry) {
			return nil
		}

		// if the entry ends with a COMMA, eat it up

		// XXX: note that this reads incorrectly formated bibtex with missing
		// commas between k/v pairs. Those commas are "uncessary" but required
		// --- so we will parse some badly formated bibtex, but get the "right"
		// answer anyway

		if p.peekTokenIs(lexer.COMMA) {
			p.advanceTokens()
		}
	}

	return entry
}

// ParseBibTeX reads the bibtex given to the parser when it was created and
// returns a database of entries
func (p *Parser) ParseBibTeX() *Database {

	p.bracesAsString = false

	// create the database that we will read into
	database := NewDatabase()

	for !p.curTokenIs(lexer.EOF) {
		switch p.curToken.Type {
		case lexer.AT:
			if entry := p.parseEntry(); entry != nil {
				switch entry.Kind {
				case String:
					if len(entry.Fields) != 1 {
						p.addError("Wrong number of string definitions in @string")
					} else {
						for k, v := range entry.Fields {
							database.Symbols[k] = v
						}
					}
				case Preamble:
					database.Preamble = append(database.Preamble, entry.Key)
				default:
					database.Pubs = append(database.Pubs, entry)
				}
			}
		}
		p.advanceTokens()
	}
	return database
}

// NErrors() returns the number of errors encountered while parsing
func (p *Parser) NErrors() int {
	return len(p.errors)
}

// PrintErrors writes the stored error messages to the given Writer
func (p *Parser) PrintErrors(w io.Writer) {
	for _, e := range p.errors {
		line, col := e.tok.Position()
		fmt.Fprintf(w, "error: line %d, col %d: %s", line, col, e.msg)
		if e.err != nil {
			fmt.Fprintf(w, " (%v)", e.err)
		}
		fmt.Fprintln(w)
	}
}

/*===============================================================================*
 * Output routines
 *===============================================================================*/

// writeTagValue writes a tag = value pair in an entry to w.
func writeTagValue(w io.Writer, tag string, value *Value) {
	fmt.Fprintf(w, "  %-10s = ", strings.ToLower(tag))
	value.write(w)
	fmt.Fprintf(w, ",\n")
}

// writeEntry writes an entire entry to w. If the kind of the entry is
// String or Preamble, the formating will *not* be correct. The fields
// will be ordered by first required, then optional, then blessed, then
// everything else
func writeEntry(w io.Writer, e *Entry) {
	fmt.Fprintf(w, "\n@%s{%s,\n",
		strings.ToLower(e.EntryString),
		e.Key)

	// if this entry kind has a list of required fields,
	// print each of the required fields in order
	printed := make(map[string]bool)
	if req, ok := required[e.Kind]; ok {
		for _, r := range req {
			for _, s := range strings.Split(r, "/") {
				if v, ok := e.Fields[s]; ok {
					writeTagValue(w, s, v)
					printed[s] = true
				}
			}
		}
	}

	// print the known optional fields
	if opt, ok := optional[e.Kind]; ok {
		for _, r := range opt {
			if v, ok := e.Fields[r]; ok {
				writeTagValue(w, r, v)
				printed[r] = true
			}
		}
	}

	// print the blessed fields
	for _, tag := range blessed {
		if v, ok := e.Fields[tag]; ok {
			writeTagValue(w, tag, v)
			printed[tag] = true
		}
	}

	// print all the other tags, in sorted order
	for _, tag := range e.Tags() {
		if _, ok := printed[tag]; !ok {
			writeTagValue(w, tag, e.Fields[tag])
		}
	}

	fmt.Fprintf(w, "}\n")
}

// writeValue formats and writes the value to the given field
func (value *Value) write(w io.Writer) {
	switch value.T {
	case StringType:
		fmt.Fprintf(w, "{%s}", value.S)
	case NumberType:
		fmt.Fprintf(w, "%d", value.I)
	case SymbolType:
		fmt.Fprintf(w, "%s", value.S)
	default:
		panic("unknown field value type")
	}
}

// writeSymbol writes an @string entry for the given k/v pair
func writeSymbol(w io.Writer, k string, v *Value) {
	fmt.Fprintf(w, "@string{ %-10s = ", k)
	v.write(w)
	fmt.Fprintf(w, " }\n")
}

// writePreamble writes an @preamble entnry for the given string
func writePreamble(w io.Writer, k string) {
	fmt.Fprintf(w, "@preamble{%s}\n", k)
}

// writeDatabase writes the entire database to w
func (db *Database) WriteDatabase(w io.Writer) {
	for _, v := range db.Preamble {
		writePreamble(w, v)
	}
	fmt.Fprintln(w)

	// write the symbols in sorted order
	syms := make([]string, len(db.Symbols))
	i := 0
	for k := range db.Symbols {
		syms[i] = k
		i++
	}
	sort.Strings(syms)
	for _, k := range syms {
		writeSymbol(w, k, db.Symbols[k])
	}

	for _, e := range db.Pubs {
		writeEntry(w, e)
	}
}

/*=======================================================================================
 * Author handling routines
 *======================================================================================*/

// parseVon finds the von part of a last name. The last name should be provided
// as a list of words
func parseVon(last []string) (string, string) {
	if len(last) == 1 {
		return "", last[0]
	}

	// find the last word that starts with a lowercase letter, excluding the
	// last word
	j := len(last) - 2
	for ; j >= 0; j-- {
		if r, _ := utf8.DecodeRuneInString(last[j]); unicode.IsLower(r) {
			break
		}
	}

	// von is the string between i and the last word that starts with a lower
	// case (excluding the last word
	von := ""
	if j >= 0 {
		von = strings.Join(last[:j+1], " ")
	}

	// last is everything else
	lastName := strings.Join(last[j+1:], " ")

	return von, lastName
}

// Given a name (without commas), finds the first, von and last name parts
// the jr part must be empty in this case
func parseNameParts(name string) (string, string, string) {
	s := splitOnTopLevel(name)

	// first is the longest sequence of words that start with a uppercase
	// that is not the entire string
	first := ""
	space := ""
	i := 0
	for ; i < len(s)-1; i++ {
		r, _ := utf8.DecodeRuneInString(s[i])
		if r != utf8.RuneError && unicode.IsUpper(r) {
			first = first + space + s[i]
			space = " "
		} else {
			break
		}
	}

	von, last := parseVon(s[i:])
	return first, von, last
}

// NormalizeName returns an Author object parsed from a name string
// We try to follow BibTeX's (rediculous) rules for parsing names
func NormalizeName(name string) *Author {
	// if we're given an empty string
	name = strings.TrimSpace(name)
	if len(name) == 0 {
		return nil
	}
	if strings.ToLower(name) == "others" {
		return &Author{Others: true}
	}

	parts := splitOnTopLevelString(name, ",", false)
	for i, p := range parts {
		parts[i] = strings.TrimSpace(p)
	}

	a := &Author{}
	switch len(parts) {
	case 1:
		f, v, l := parseNameParts(parts[0])
		a.First = f
		a.Von = v
		a.Last = l
	case 2:
		v, l := parseVon(splitOnTopLevel(parts[0]))
		a.Von = v
		a.Last = l
		a.First = parts[1]
	case 3:
		v, l := parseVon(splitOnTopLevel(parts[0]))
		a.Von = v
		a.Last = l
		a.Jr = parts[1]
		a.First = parts[2]
	default:
		return nil
	}
	return a
}

// surround a {} if it contains a top-level "
func quoteName(s string) string {
	hastopquote := false
	nbrace := 0
	for _, r := range s {
		switch r {
		case '{':
			nbrace++
		case '}':
			nbrace--
		case '"':
			if nbrace == 0 {
				hastopquote = true
			}
		}
	}
	if hastopquote {
		return "{" + s + "}"
	}
	return s
}

// String() converts an author to a normalized name string.  We always use the
// von Last, First or von Last, Jr, First formats depending onn whether Jr is
// empty or not
func (a *Author) String() string {
	if a.Others {
		return "others"
	}
	last := a.Last
	if a.Von != "" {
		last = a.Von + " " + a.Last
	}

	last = quoteName(last)
	first := quoteName(a.First)
	jr := quoteName(a.Jr)

	if last == "" {
		return a.First
	} else if a.First == "" {
		return last
	} else if a.Jr == "" {
		return fmt.Sprintf("%s, %s", last, first)
	} else {
		return fmt.Sprintf("%s, %s, %s", last, jr, first)
	}
}
