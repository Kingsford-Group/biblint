
## Installing from Source

If you want to use `biblint`, take the following steps:

1. Install Go (version 1.18 or newer) from [http://golang.org](http://golang.org)
2. `go install github.com/Kingsford-Group/biblint/biblint@latest`
3. The command will be installed according to `go install`'s rules (in `~/go/bin` if you don't set the `GOBIN` variable; otherwise wherever `GOBIN` points).

## Building from Source

If you want to hack on the source code, take the following steps:

1. Install Go (version 1.18 or newer) from [http://golang.org](http://golang.org)
2. clone this repo wherever you want
3. `cd` into the `biblint/biblint` directory
4. `go build`

Currently, `biblint` has no external dependencies beyond Go and its standard library.

## biblint clean

BibLint's `clean` command tries to format the bib file in a consistent way.  It
tries to correct common mistakes, and removes information that is not part of
the "citation". 

**Note that `clean` does NOT guarantee no data loss. In fact, the typical situation is that
data will be lost (e.g. abstracts and many other fields are removed from the database).**

Usage:
```
biblint clean in.bib > out.bib
```

Specifically, `clean` does the following:

- @preamble entries are moved to the top of the file (in the order they appear
  in the file)

- @string entries immediately follow any preamble entries. They are listed
  alphabetically sorted by the symbol they define

- Entries follow, sorted in reverse chronological order (i.e. by year, by
  default). Use `-sort` option to change the field that will be sorted on, and
  the `-reverse` option to reverse the order (e.g. `-sort journal -reverse
  false`) will sort alphabetically by journal string. Note that the default
  sorted order is `reverse=true`, so if you want alphabetical, you must turn
  off reverse.

  biblint tries to be minimally smart about sorting: symbols are expanded to
  their defined value (recursively, up to depth 10), strings are compared
  ignoring {}, and if an int and a string are compared, the int is converted to
  a string and the comparison is done as strings.

- Fields that are empty are removed

- Non-blessed fields are removed. A field is blessed if it is a required or
  known optional field *for any entry type* or one of "key", "note", "url",
  "doi", "pmc", "pmid", "keywords", "issn", "isbn".  Note that "abstract" tags
  are removed. Use `-blessed f1,f2,f3...` to add additional blessed fields.

- Titles that end with `[[:lower:]]\.` have the terminating "." removed.

- Pages entries that look like NUMBER -[-] NUMBER are changed to NUMBER--NUMBER

- Pages that are aaaa--bb, where len(a) > len(b), are replaced by aaaa--aabb

- Exact duplicates are removed. Exact dups are those that have the same entry
  type, same key, the same fields, and the same exact values for each field

- If entry A has all the fields of B, with the same values, then A will be
  deleted.  (if A and B have the same fields, one of them will be deleted
  arbitrarily). This will be caught if the entries either have the same key
  or the same title

- {} is used to delimit fields

- If an entire field is braced, they are removed. This can be wrong, but the
  more common problem is that someone has double braced every field to avoid
  dealing with BibTeX quirks.

- Individual words in `title` or `booktitle` entires that are in strange case
  will be surrounded by {}. Specifically, {} surrounds any word with a " or
  that has "sTrange" case (an uppercase letter anyplace except the first
  non-punctuation character that is not preceded by a hyphen). This won't brace
  things like "(Strange" or "Hyphenated-Word", but will brace "mRNA"

- If an author field ends with `\set\s*al.?` it is replaced by " and others".

- Author names in the "author" field are always given as von Last, First or von
  Last, Jr., First  (names in the "editor" field are not changed)

- Plain integer values are unquoted

- If a month field is {Jan} or {January}, it will be converted to the
  predefined symbol "jan"

- If the value of a field uniquely matches the definition of a symbol, it will
  be replaced by the symbol

- Consecutive, unbraced whitespace will be replaced by a single space character

- Non-quoted whitespace is removed from the start and end of any value

- Missing commas after "tag=value" pairs are added

- If an entry contains duplicate "tag=" entries, the *first* one is kept (as in
  BibTeX) with a warning

- Lowercase, non-"small" words are capitalized in journal titles (as long as
  they are outside {} regions). Small words are "the", "a", "an", "but", "for",
  "and", "or", "nor", "to", "from", "on", "in", "of", "at", "by". (This list is
  likely to grow.)

- `@comment` lines and non-entry text are removed


## biblint check

The `check` command looks for problems that can't necessarily be fixed by `clean`.

Usage:
```
biblint check in.bib
```

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

- Fields that have an odd number of un-escaped (with \\) dollar signs

- `@string` definitions that define the same thing

- Last names that have all uppercase, all lowercase, or are empty (trying to
  catch last names resulting from the common mistake of an author = `Smith J
  H`, which is parsed by BibTeX as first name = "Smith", last name = "J H".) 

Errors are reported grouped by key in the following format:
```
Key "salmon":
  2105: key "salmon" is defined more than once
  1178:volume: missing required field "volume" in article
```
Each group starts with `Key` followed by the key in quotes. Each error is of
the two forms:
```
  LINE: message
  LINE:TAG: message
```
where `LINE` is the line number of the _entry_ (the line the `@` appears on)
and `TAG`, if present, is the tag within the entry that contains the error.
The line number is given for each message because, in the case of duplicate
keys, errors can be reported for any of those entries. They key is "<none>" if
the error doesn't involve an entry.

## biblint dups

The `dups` command tries to find duplicate entries by looking for pairs of entries
that look like they have the same title. Usage:

```
biblint dups in.bib
```

This finds entries where the titles map to the same string, once case,
punctuation, and small words are removed. It reports the dups by key and title,
but does not remove or modify the entries.

##  Typical Usage

### Cleaning bad bib files:

To clean up a bib file, a set of steps to take are:

1. Run `biblint clean in.bib > tmp.bib`
2. Fix any errors reported by `clean` in `tmp.bib`; goto 1 until no errors
3. Run `biblint dups tmp.bib`
4. Remove or fix any true dups reported by `dups` in `tmp.bib`
5. Run `biblint check tmp.bib`
6. Fix any errors reported by `check` in tmp.bib
7. `mv in.bib bad.bib && mv tmp.bib in.bib`

### Merging files:

The way to merge two or more files is to `cat` them and then run the cleaning
pipeline above.  If the bib files contain true duplicates or one file contains
strictly more information than the other for an entry, the dups or less
informative entries will be removed by `clean`.

## Other options

Use `-quiet` to prohibit printing of the banner.

## Parser "quirks"

The biblint parser accepts some bib syntax that is not officially supported by
bibtex. This is done for a combination of reasons: sometimes the bib file can
be parsed correctly and sometimes forcing non-bibtex syntax to be rejected
would complicate the parser too much. For example:

- Commas separating tag=value pairs in an entry are optional --- they will be
  added by `clean` if they are missing

- BibTeX allows both {} and () to deliminate string, preamble, and entry types
  (but not key values, which must be either {} or ""). That is, you can say
  `@article(key,title="foo")`. We also allow both () and {} but we also allow
  {) and (}. We will convert all these to {}.

- `@comment` in BibTeX comments to the end of the line. This is what we do as
  well. We strip all comments from the output .bib file. Someday, it might be
  nice to preserve @comment comments (but not non-entry junk) in the output.


## Known Bugs / Issues

- We do not yet support the `#` string concatenation operator.

- A title of the form `"strange {title here"` will be converted to `strange
  {title here}`. That is unmatched opening `{` will be closed at the end of a
  string. Escaping with `\{` doesn't stop this from happening. 

