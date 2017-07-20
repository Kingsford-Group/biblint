
## biblint clean

BibLint's `clean` command tries to format the bib file in a consistent way.  It
tries to correct common mistakes, and removes information that is not part of
the "citation". 

**Note that `clean` does NOT guarantee no data loss. In fact, the typical situation is that
data will be lost (e.g. abstracts are removed from the database)**

Specifically, `clean` does the following:

- @preamble entries are at the top of the file

- @string entries immediately follow any preamble entries. They are listed
  alphabetically sorted by the symbol they define

- Entries follow, sorted in reverse chronological order.

- Fields that are empty are removed

- Non-blessed fields are removed. A field is blessed if it is a required or
  known optional field *for any entry type* or one of "key", "note", "url",
  "doi", "pmc", "pmid", "keywords", "issn", "isbn".  Note that "abstract" tags
  are removed.

- Titles that end with `[[:lower:]]\." have the terminating "." removed.

- Pages entries that look like NUMBER -[-] NUMBER are changed to NUMBER--NUMBER

- Pages that are aaaa--bb, where len(a) > len(b), are replaced by aaaa--aabb

- Exact duplicates are removed. Exact dups are those that have the same entry
  type, the same fields, and the same exact values for each field

- {} is used to eliminate fields

- If an entire field is braced or if no braces are in the field, individual
  words that are in strange case will be surrounded by {}. Specifically, {}
  surrounds any word with a " or that has "sTrange" case (an uppercase letter
  anyplace except the first non-punctuation character). This won't brace things
  like "(Strange" or "Hyphenated-Word", but will brace "mRNA"

- Author names in the "author" field are always given as von Last, First or von
  Last, Jr., First  (names in the "editor" field are not changed)

- Plain integer values are unquoted

- If a month field is {Jan} or {January}, it will be converted to the
  predefined symbol "jan"

- If the value of a field uniquely matches the definition of a symbol, it will
  be replaced by the symbol

- Consecutive, unbraced whitespace will be replaced by a single space character

- Missing commas after "tag=value" pairs are added

## biblint check

The `check` command looks for problems that can't be fixed by `clean`.
Specifically, it will report the following problems:

- A lone, white-space-surrounded - instead of ---

- "et al" in an author list

- Non-ASCII characters anyplace

- Years that are not integers

- Use of undefined symbols

- Duplicate defined symbols

- Duplicate keys

- Page ranges x--y where y < x

- Missing required fields for each entry type

## Parser "quirks"

The biblint parser accepts some bib syntax that is not officially supported by
bibtex. This is done for a combination of reasons: sometimes the bib file can
be parsed correctly and sometimes forcing bibtex syntax to be rejected would
complicate the parser too much. For example:

- Commas separating tag=value pairs in an entry are optional --- they will be
  added by clean if they are missing

- `@preamble"foo bar baz"` is accepted in addition to `@preamble{foo bar baz}`.
  Since no one uses @preamble anyway, this is not likely to cause problems.
  (We do this since the lexer knows about both "" and {} strings and, once
  lexed, treats them the same. The only place this is not true in bibtex is the
  @preamble.

- BibTeX supports using () to delimit entries and strings, e.g., `@string(foo =
  "bar")` or `@article(title = "hi there")`. We only support {} in this case.
  (this might change -- but since so few bib files use () in this case, it's
  low priority)

## Known Bugs / Issues

- We do not yet support the `#` string concatenation operator.

