/* -*- coding: utf-8 -*-
 * vim: set enc=utf8
 *
 * Alexander Rødseth <rodseth@gmail.com>
 * Nov 2010
 * Jan 2011
 * Feb 2011
 * Apr 2011
 *
 * GPL2
 */

package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

const (
	version = "0.9"
	ifdef   = "#ifdef"
	ifndef  = "#ifndef"
	incl    = "#include"
	dosEOL  = "\r\n"
	unixEOL = "\n"
)

// Represents the text in a C source file
type SourceCode struct {
	text               string
	newline            string
	rememberHasIfdef   bool
	rememberHasIfndef  bool
	rememberHasInclude bool
}

// "Constructor"
func newSourceCode(text string) *SourceCode {
	source := new(SourceCode)
	source.set(text)
	return source
}

func (my *SourceCode) get() string                  { return my.text }
func (my *SourceCode) getNewline() string           { return my.newline }
func (my *SourceCode) has(text string) bool         { return strings.Index(my.text, text) != -1 }
func (my *SourceCode) first(text string) int        { return strings.Index(my.text, text) }
func (my *SourceCode) hasIfdef() bool               { return my.rememberHasIfdef }
func (my *SourceCode) hasIfndef() bool              { return my.rememberHasIfndef }
func (my *SourceCode) hasInclude() bool             { return my.rememberHasInclude }
func (my *SourceCode) firstIfdef() int              { return my.first(ifdef) }
func (my *SourceCode) firstIfndef() int             { return my.first(ifndef) }
func (my *SourceCode) firstInclude() int            { return my.first(incl) }
func (my *SourceCode) nextInclude(pos int) int      { return strings.Index(my.text[pos+1:], incl) }
func (my *SourceCode) firstIncludeAfterIfdef() int  { return my.firstIncludeAfterWord(ifdef) }
func (my *SourceCode) firstIncludeAfterIfndef() int { return my.firstIncludeAfterWord(ifndef) }
func (my *SourceCode) theRest(pos int) *SourceCode  { return newSourceCode(my.text[pos:]) }

func (my *SourceCode) set(text string) {
	my.text = text
	my.newline = my.discoverNewline()
	// memoization
	my.rememberHasIfdef = my.has(ifdef)
	my.rememberHasIfndef = my.has(ifndef)
	my.rememberHasInclude = my.has(incl)
}

func (my *SourceCode) discoverNewline() string {
	// If there is a \r\n, assume it's DOS/Windows line endings
	if strings.Index(my.text, dosEOL) != -1 {
		return dosEOL
	}
	return unixEOL
}

func (my *SourceCode) hasIfdefBefore(pos int) bool {
	found := my.firstIfdef()
	return (found != -1) && (found < pos)
}

func (my *SourceCode) firstIncludeAfterWord(word string) int {
	if !my.hasInclude() && !my.has(word) {
		return 0
	} else if my.hasInclude() && !my.has(word) {
		return my.firstInclude()
	} else {
		pos := my.first(word)
		tail := my.theRest(pos)
		if tail.hasInclude() {
			return pos + tail.firstInclude()
		} else {
			return pos
		}
	}
	return 0
}

func (my *SourceCode) endofline(pos int) int {
	// Finds the end of the line at the given position
	tail := my.theRest(pos + 1)
	npos := tail.first(my.newline)
	if npos != -1 {
		return pos + 1 + npos
	}
	return 0
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Try to find an appropriate insertion position for new includes
func (s *SourceCode) findInsertPos() int {
	var pos int
	if s.hasInclude() && !s.hasIfdef() && !s.hasIfndef() {
		pos = s.firstInclude()
	} else if !s.hasInclude() && s.hasIfdef() && !s.hasIfndef() {
		pos = s.firstIfdef()
	} else if !s.hasInclude() && !s.hasIfdef() && s.hasIfndef() {
		pos = s.firstIfndef()
	} else if !s.hasInclude() && s.hasIfdef() && s.hasIfndef() {
		pos = min(s.firstIfdef(), s.firstIfndef())
	} else if s.hasInclude() && s.hasIfdef() && !s.hasIfndef() {
		pos = s.firstIncludeAfterIfdef()
	} else if s.hasInclude() && !s.hasIfdef() && s.hasIfndef() {
		pos = s.firstIncludeAfterIfndef()
	} else if s.hasInclude() && s.hasIfdef() && s.hasIfndef() {
		pos = min(s.firstIncludeAfterIfdef(), s.firstIncludeAfterIfndef())
	} else {
		return 0
	}
	return s.endofline(pos)
}

// ---------------------------------------------------------------------

// Try to expand include-strings (for instance, "stdin" becomes "#include <stdin.h>")
func expandInclude(include string) string {
	if strings.Index(include, " ") == -1 {
		// Include is just a word
		if strings.Index(include, "<") == -1 && strings.Index(include, "\"") == -1 {
			// ...and needs brackets
			if strings.Index(include, ".") == -1 {
				// Add .h if it is missing
				include = include + ".h"
			}
			// Add brackets
			return incl + " <" + include + ">"
		} else {
			// ...and does not need brackets
			if strings.Index(include, ".") == -1 {
				// Add .h if it is missing, inside the brackets
				bracketchar := include[len(include)-1:]
				include = include[0:len(include)-1] + ".h" + bracketchar
				//include = include + bracketchar
			}
			return incl + " " + include
		}
	} else {
		// Include is two words?
		if strings.Count(include, " ") == 1 {
			spacepos := strings.Index(include, " ")
			firstword := include[0:spacepos]
			tail := include[spacepos+1:]
			if firstword != incl {
				return expandInclude(tail)
			}
			// We have the second word, now fix it up
			return expandInclude(tail)
		} else {
			fmt.Fprintf(os.Stderr, "Strange include: %s\n", include)
			os.Exit(3)
		}
	}
	return include
}

func addIncludeToFile(filename string, include string, fixinclude bool, at_top bool) {
	var (
		source        SourceCode
		fixed_include string
	)
	if fixinclude {
		fixed_include = expandInclude(include)
	} else {
		fixed_include = include
	}

	filedata, err := ioutil.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not read %s\n", filename)
		os.Exit(2)
	}
	filetext := string(filedata)
	source.set(filetext)

	// Set the placement position at the top, or at a suitable place
	pos := 0
	if !at_top {
		pos = source.findInsertPos()
	}

	newline := source.getNewline()
	newtext := filetext[:pos] + newline + fixed_include + newline + filetext[pos:]
	ioutil.WriteFile(filename, []byte(newtext), 0)
}

/*
 * ------------------------ main --------------------------------
 */

func main() {

	nofix_text := "don't change the include text"
	top_text := "add the include at the top"
	version_text := "show the current version"
	help_text := "this brief help"

	flag.Usage = func() {
		fmt.Println("addinclude adds an include to a C header- or source file")
		fmt.Println()
		fmt.Println("Arguments:")
		fmt.Println("\tfilename, include")
		fmt.Println("\t-n or --nofix\t\t", nofix_text)
		fmt.Println("\t-t or --top\t\t", top_text)
		fmt.Println("\t-v or --version\t\t", version_text)
		fmt.Println("\t-h or --help\t\t", help_text)
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("\taddinclude myfile.h '#include <string.h>'")
		fmt.Println("\taddinclude --top myfile.h stdlib")
		fmt.Println("\taddinclude myfile.h '\"some.h\"'")
		fmt.Println()
	}

	var missing_args = func() {
		fmt.Fprintf(os.Stderr, "Needs a filename and an include. Use --help for more info.\n")
		os.Exit(1)
	}

	/* Wish there were support for long and short options in the flag package */
	var nofix_long *bool = flag.Bool("nofix", false, nofix_text)
	var nofix_short *bool = flag.Bool("n", false, nofix_text)
	var top_long *bool = flag.Bool("top", false, top_text)
	var top_short *bool = flag.Bool("t", false, top_text)
	var version_long *bool = flag.Bool("version", false, version_text)
	var version_short *bool = flag.Bool("v", false, version_text)
	var help_long *bool = flag.Bool("help", false, help_text)
	var help_short *bool = flag.Bool("h", false, help_text)

	flag.Parse()

	nofix := *nofix_long || *nofix_short
	top := *top_long || *top_short
	version := *version_long || *version_short
	help := *help_long || *help_short

	args := flag.Args()

	if help {
		flag.Usage()
	} else if version {
		fmt.Println(version)
	} else if len(args) == 2 {
		filename := flag.Arg(0)
		include := flag.Arg(1)
		// Notice the !
		addIncludeToFile(filename, include, !nofix, top)
	} else {
		missing_args()
	}
}

/* This is my first Go-program that is not just "hello world" :) */
