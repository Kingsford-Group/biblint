
## biblint clean

BibLint's `clean` command tries to format the bib file in a consistent way.
It tries to correct common mistakes, and removes information that is not part
of the "citation". Specifically, it does the following:

- @preamble entries are at the top of the file

- @string entries immediately follow any preamble entries. They are listed
  alphabetically sorted by the symbol they define

- Entries follow, sorted in reverse chronological order.

- Fields that are empty are removed

- Non-blessed fields are removed. A field is blessed if it is a required or
  known optional field *for any entry type* or one of "key", "note", "url",
  "doi", "pmc", "pmid", "keywords", "issn", "isbn".

- Titles that end with LOWERCASE "." have the terminating "." removed.

- Pages entries that look like NUMBER -[-] NUMBER are changed to NUMBER--NUMBER

- Pages that are aaaa--bb are replaced by aaaa--aabb

- Exact duplicates are removed. Exact dups are those that have the same entry type,
  the same fields, and the same exact values for each field

- {} is used to eliminate fields

- If an entire field is braced or if no braces are in the field, individual
  words that are in strange case will be surrounded by {}. Specifically, {}
  surrounds any word with a " or that has "sTrange" case (an uppercase letter
  anyplace except the first non-punctuation character). This won't brace things
  like "(Strange" or "Hyphenated-Word", but will brace "mRNA"

- Author names in the "author" field are always given as von Last, First or von
  Last, Jr., First  (names in the "editor" field are not changed)

- Plain integer values are unquoted

- If a month field is {Jan} or {January}, it will be converted to the predefined symbol "jan"

- If the value of a field uniquely matches the definition of a symbol, it will
  be replaced by the symbol

- Consequtive, unbraced whitespace will be replaced by " "

## biblint check

The `check` command looks for problems that can't be fixed by `clean`. Specifically, it will
report the following problems:

- Use of a lone, whitespace-surrounded - instead of ---

- "et al" in an author list

- Non-ASCII characters anyplace

- Years that are not integers

- Use of undefined symbols

- Duplicate defined symbols

- Duplicate keys
