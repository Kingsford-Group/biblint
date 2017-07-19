package main

import (
	"ckingsford/bibutil/bib"
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
)

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

// registerSubcommand creates a record for the given subcommand. The handler do
// will be called when name is used as the subcommand on the command line.
func registerSubcommand(name, desc string, do subcommandFunc) {
	subcommands[name] = &subcommand{
		name:  name,
		desc:  desc,
		flags: flag.NewFlagSet(name, flag.ExitOnError),
		do:    do,
	}
}

// printSubcommandDesc is used by the help system to print out the registered subcommands
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

// doFmt reads a bibtex file and formats it using a "standard" format.
func doClean(c *subcommand) bool {
	if c.flags.NArg() < 1 {
		fmt.Println("error: missing filename in fmt")
		return false
	}

	// read the bibtex file
	f, err := os.Open(c.flags.Arg(0))
	if err != nil {
		fmt.Printf("error: couldn't open %s\n", c.flags.Arg(0))
		return false
	}
	p := bib.NewParser(f)
	db := p.ParseBibTeX()
	if p.NErrors() > 0 {
		p.PrintErrors(os.Stderr)
	}

	// clean it up
	db.NormalizeWhitespace()
	db.RemoveWholeFieldBraces()
	db.ConvertIntStringsToInt()
	db.ReplaceSymbols()
	db.ReplaceAbbrMonths()
	db.RemoveNonBlessedFields([]string{})
	db.RemoveEmptyFields()
	db.NormalizeAuthors()
	db.RemovePeriodFromTitles()
	db.FixHyphensInPages()

	db.RemoveExactDups()

	db.SortByField("year", true)

	db.CheckYearsAreInt()
	db.CheckEtAl()
	db.CheckASCII()
	db.CheckLoneHyphenInTitle()
	db.CheckPageRanges()
	db.PrintErrors(os.Stderr)

	// write it out
	db.WriteDatabase(os.Stdout)
	log.Printf("Wrote %d publications.", len(db.Pubs))
	return true
}

func main() {
	log.SetOutput(os.Stderr)
	log.SetPrefix("biblint: ")
	log.SetFlags(0)

	// register the subcommands
	registerSubcommand("clean", "Clean up nonsense in a BibTeX file", doClean)

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
	c.flags.Parse(os.Args[2:])
	c.do(c)
}
