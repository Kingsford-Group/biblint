// (c) 2018 by Carl Kingsford (carlk@cs.cmu.edu). See LICENSE.txt.
package lexer

import (
	"fmt"
	"strings"
	"testing"
)

func TestLexer(t *testing.T) {
	const in = "//moo or what @article{fur, title = \"Now is \\\"the\\\" {time}\", author={Carl {Kingsford}}, pages=32@"

	bstring := false
	l := New(strings.NewReader(in))
	tok, err := l.NextToken(bstring)

	for err == nil && tok != EOFToken {
		if tok.Literal == "author" {
			bstring = true
		}

		fmt.Println(tok, err)
		tok, err = l.NextToken(bstring)
	}
}
