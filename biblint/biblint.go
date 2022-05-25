// (c) 2018-2022 by Carl Kingsford (carlk@cs.cmu.edu). See LICENSE.txt.
package main

import (
	"flag"
	"fmt"
	"github.com/Kingsford-Group/biblint/bib"
	"log"
	"os"
	"sort"
	"strings"
)

const version = "v0.4"

// Represents a function that handles a subcommand
type subcommandFunc func(*subcommand) bool

type subcommand struct {
	name  string
	desc  string
	flags *flag.FlagSet
	do    subcommandFunc
}

// subcommands maps a subcommand name to its handler
var subcommands = make(map[string]*subcommand, 0)

var quiet bool

// registerSubcommand creates a record for the given subcommand. The handler do
// will be called when name is used as the subcommand on the command line.
func registerSubcommand(name, desc string, do subcommandFunc) *subcommand {
	c := &subcommand{
		name:  name,
		desc:  desc,
		flags: flag.NewFlagSet(name, flag.ExitOnError),
		do:    do,
	}
	// define flags that are common to all subcommands
	c.flags.BoolVar(&quiet, "quiet", false, "minimize output messages")
	subcommands[name] = c
	return c
}

// printSubcommandDesc is used by the help system to print out the registered subcommands.
func printSubcommandDesc() {
	cmds := make([]string, len(subcommands))
	i := 0
	for k := range subcommands {
		cmds[i] = k
		i++
	}
	sort.Strings(cmds)

	for _, c := range cmds {
		fmt.Printf("  %-8s : %s\n", subcommands[c].name, subcommands[c].desc)
	}
}

// startSubcommand parses the flags and prints the banner.
func startSubcommand(c *subcommand) bool {
	c.flags.Parse(os.Args[2:])
	if !c.flags.Parsed() {
		c.flags.Usage()
		return false
	}

	if !quiet {
		printBanner()
	}
	return true
}

// parseBibFromArgs reads the first argument as a bib file and returns the database.
func parseBibFromArgs(c *subcommand) (*bib.Database, bool) {
	if c.flags.NArg() < 1 {
		fmt.Println("error: missing filename in fmt")
		c.flags.Usage()
		return nil, false
	}

	// read the bibtex file
	f, err := os.Open(c.flags.Arg(0))
	if err != nil {
		fmt.Printf("error: couldn't open %s\n", c.flags.Arg(0))
		return nil, false
	}
	p := bib.NewParser(f)
	db := p.ParseBibTeX()
	if p.NErrors() > 0 {
		p.PrintErrors(os.Stderr)
	}
	return db, true

}

// doClean reads a bibtex file and formats it using a "standard" format.
func doClean(c *subcommand) bool {
	sortby := c.flags.String("sort", "year", "sorts the entry by `field`")
	reverse := c.flags.Bool("reverse", true, "reverse the sort order")
	blessed := c.flags.String("blessed", "", "Comma separated list of blessed `fields`")

	if !startSubcommand(c) {
		return false
	}

	db, ok := parseBibFromArgs(c)
	if !ok {
		return false
	}

	// parse the blessed fields
	blessedArr := strings.Split(*blessed, ",")
	for i, b := range blessedArr {
		blessedArr[i] = strings.TrimSpace(strings.ToLower(b))
	}

	// clean it up
	db.NormalizeWhitespace()
	db.RemoveWholeFieldBraces()
	db.CanonicalBrace()
	db.ConvertTitlesToMinBraces()
	db.ConvertIntStringsToInt()
	db.ReplaceSymbols()
	db.ReplaceAbbrMonths()
	db.RemoveNonBlessedFields(blessedArr)
	db.RemoveEmptyFields()
	db.ReplaceAuthorEtAl()
	db.NormalizeAuthors()
	db.RemovePeriodFromTitles()
	db.FixHyphensInPages()
	db.FixTruncatedPageNumbers()
	db.TitleCaseJournalNames()
	db.RemoveContainedEntries()

	db.RemoveExactDups()

	db.SortByField(*sortby, *reverse)

	// write it out
	db.WriteDatabase(os.Stdout)
	if !quiet {
		log.Printf("Wrote %d publications.", len(db.Pubs))
	}
	return true
}

// doCheck runs the check command.
func doCheck(c *subcommand) bool {
	if !startSubcommand(c) {
		return false
	}

	db, ok := parseBibFromArgs(c)
	if !ok {
		return false
	}

	db.CheckYearsAreInt()
	db.CheckEtAl()
	db.CheckASCII()
	db.CheckLoneHyphenInTitle()
	db.CheckPageRanges()
	db.CheckUndefinedSymbols()
	db.CheckDuplicateKeys()
	db.CheckRequiredFields()
	db.CheckUnmatchedDollarSigns()
	db.CheckRedundantSymbols()

	db.NormalizeAuthors()
	db.CheckAuthorLast()

	db.PrintErrors(os.Stdout)

	return true
}

// doDups runs the dups command, identifying and printing possible duplicates.
func doDups(c *subcommand) bool {
	if !startSubcommand(c) {
		return false
	}

	db, ok := parseBibFromArgs(c)
	if !ok {
		return false
	}

	for hash, list := range db.FindDupsByTitle() {
		if hash != "" && len(list) > 1 {
			fmt.Printf("Possible Duplicates:\n")
			for _, e := range list {
				// title field must exist since hash != ""
				fmt.Printf("   %s \"%s\"\n", e.Key, e.Fields["title"].S)
			}
		}
	}

	return true
}

// printBanner prints out the version, tool name and copyright info
func printBanner() {
	fmt.Fprintf(os.Stderr, "biblint %s (c) 2017-2018 Carl Kingsford. See LICENSE.txt.\n", version)
}

func registerAllSubcommands() {
	// register the subcommands
	registerSubcommand("clean", "Clean up nonsense in a BibTeX file", doClean)
	registerSubcommand("check", "Look for errors that can't be automatically corrected", doCheck)
	registerSubcommand("dups", "Look for duplicate entries", doDups)
}

func main() {
	log.SetOutput(os.Stderr)
	log.SetPrefix("biblint: ")
	log.SetFlags(0)

	registerAllSubcommands()

	// if no command listed, report error
	if len(os.Args) == 1 {
		fmt.Println("usage: biblint <commad> [<args>]")
		fmt.Println("The most commonly used biblint commands are: ")
		printSubcommandDesc()
		return
	}

	// parse the command line according to this subcommand
	c, ok := subcommands[os.Args[1]]
	if !ok {
		log.Fatalf("error: %q is not a valid subcommand.\n", os.Args[1])
	}
	c.do(c)
}
