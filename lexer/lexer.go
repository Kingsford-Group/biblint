package lexer

/*
   A lexer for bibtex files. lexer.New will create a new lexer and
   lexer.NextToken will repeatedly return the next Token.
*/

import (
	"bufio"
	"io"
	"strings"
	"unicode"
)

//==================================================================
// Tokens
//==================================================================

type TokenType string

// Token is returned by NextToken(). Literal is the string corresponding to the token.
type Token struct {
	Type    TokenType
	Literal string
	lineno  int
	colno   int
}

// The types of tokens that the lexer can return
const (
	ILLEGAL TokenType = "ILLEGAL"
	EOF               = "EOF"
	IDENT             = "IDENT"
	STRING            = "STRING"
	AT                = "@"
	COMMA             = ","
	LBRACE            = "{"
	RBRACE            = "}"
	HASH              = "#"
	EQUALS            = "="
)

var EOFToken = &Token{Type: EOF}

func (t *Token) Position() (int, int) {
	return t.lineno, t.colno
}

//==================================================================
// The Lexer
//==================================================================

type Lexer struct {
	stream *bufio.Reader
	ch     rune
	err    error
	lineno int
	colno  int
}

// NewLexer returns a new lexer than will return a stream of tokens in
// the bibtex language.
func New(f io.Reader) *Lexer {
	l := Lexer{
		stream: bufio.NewReader(f),
		ch:     0,
		err:    nil,
		lineno: 1,
		colno:  1,
	}
	l.nextRune()
	return &l
}

// nextRune reads the next rune from the buffered stream. It returns true if we
// succeed; if so, curRune() contains the next rune otherwise Err() will be
// non-nil
func (l *Lexer) nextRune() bool {
	ch, _, err := l.stream.ReadRune()
	if err != nil {
		l.err = err
		//l.ch = 0
		return false
	}

	l.ch = ch
	l.colno++

	if l.ch == '\n' {
		l.lineno++
		l.colno = 1
	}
	return true
}

func (l *Lexer) Position() (int, int) {
	return l.lineno, l.colno
}

// curRune returns the rune that was last read by nextRune()
func (l *Lexer) curRune() rune {
	return l.ch
}

// Err returns the last recorded error
func (l *Lexer) Err() error {
	return l.err
}

// skipWhiteSpace skips until the current rune is a non-whitespace character
func (l *Lexer) skipWhitespace() error {
	first := true
	for first || l.nextRune() {
		if !unicode.IsSpace(l.curRune()) {
			return l.Err()
		}
		first = false
	}
	return l.Err()
}

// skipToNewLine skips until the current run is '\n'
func (l *Lexer) SkipToNewLine() error {
    for l.curRune() != '\n' {
        l.nextRune()
    }
    return l.Err()
}

// readQuoteString reads the quoted string. It assumes that the current rune is
// *not* part of the string (e.g. it is the opening ") and it will not include
// terminating " in the returned string on error, the string will be nonsense
// handles \" escapes to include literal quotes in the string. It consumes
// the final "
func (l *Lexer) readQuoteString() (string, error) {
	escape := false
	b := make([]rune, 0)

	for l.nextRune() {
		if l.curRune() == '"' && !escape {
			l.nextRune()
			return string(b), l.Err()
		} else {
			b = append(b, l.curRune())
		}

		escape = !escape && l.curRune() == '\\'
	}
	return "", l.Err()
}

// readIdent reads an identifier which is a continugous string on non-space
// characters that are not @#,{}="( which are the special characters used by
// bibtex
func (l *Lexer) readIdent() (string, error) {
	b := []rune{l.curRune()}

	for l.nextRune() {
		if unicode.IsSpace(l.curRune()) || strings.ContainsRune("@#,{}=\"(", l.curRune()) {
			return string(b), l.Err()
		} else {
			b = append(b, l.curRune())
		}
	}
	return "", l.Err()
}

// readBracesString reads a {} deliminated string. {} pairs can be nested and
// are handled correctly. \{ and \} are treated property as plain characters
// assumes that the current rune is *not* part of the string (i.e. it is the
// opening '{'
func (l *Lexer) readBracesString() (string, error) {
	escape := false
	b := make([]rune, 0)
	bcount := 1

	for l.nextRune() {
		switch l.curRune() {
		case '{':
			if !escape {
				bcount++
			}

		case '}':
			if !escape {
				bcount--
			}
		}

		if bcount == 0 {
			l.nextRune()
			return string(b), l.Err()
		}
		b = append(b, l.curRune())

		escape = !escape && l.curRune() == '\\'
	}
	return "", l.Err()
}

func (l *Lexer) newToken(t TokenType, s string) *Token {
	return &Token{
		Type:    t,
		Literal: s,
		lineno:  l.lineno,
		colno:   l.colno,
	}
}

// NextToken produces the next token. Assumes curRune() will give the next
// unprocessed character we must maintain the above invariant after newLexer()
// and nextToken()) If braceStrings is true, treats {}-deliminated regions
// as a string (requiring balanced {} strings)
func (l *Lexer) NextToken(braceStrings bool) (*Token, error) {

	// move past any whitespace
	if err := l.skipWhitespace(); err != nil {
		if err == io.EOF {
			return EOFToken, nil
		} else {
			return nil, err
		}
	}

	var t *Token

	switch l.curRune() {
	case '@':
		t = l.newToken(AT, "@")
		l.nextRune()
	case ',':
		t = l.newToken(COMMA, ",")
		l.nextRune()
    case '}':
		t = l.newToken(RBRACE, "}")
		l.nextRune()
    case ')': // ) acts as a } where it can
        t = l.newToken(RBRACE, ")")
        l.nextRune()
	case '=':
		t = l.newToken(EQUALS, "=")
		l.nextRune()
	case '#':
		t = l.newToken(HASH, "#")
		l.nextRune()

	// either return the LBRACE symbol or scoop up the
	// entire string until the end brace
	case '{':
		if !braceStrings {
			t = l.newToken(LBRACE, "{")
			l.nextRune()
		} else {
			s, err := l.readBracesString()
			if err != nil {
				return nil, err
			}
			t = l.newToken(STRING, s)
		}

    case '(':
        t = l.newToken(LBRACE, "(")
        l.nextRune()

	case '"':
		s, err := l.readQuoteString()
		if err != nil {
			return nil, err
		}
		t = l.newToken(STRING, s)

	// read an identifier
	default:
		s, err := l.readIdent()
		if err != nil {
			return nil, err
		}
		t = l.newToken(IDENT, s)
	}
	return t, nil
}
