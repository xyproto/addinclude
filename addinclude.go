/* -*- coding: utf-8 -*-
 * vim: set enc=utf8
 *
 * Alexander F Rødseth <xyproto@archlinux.org>
 * Nov 2010
 * Jan 2011
 * Feb 2011
 * Apr 2011
 * Nov 2017
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
	versionString = "addinclude 1.0"
	ifdef         = "#ifdef"
	ifndef        = "#ifndef"
	incl          = "#include"
	dosEOL        = "\r\n"
	unixEOL       = "\n"
)

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// SourceCode represents the text in a C source file
type SourceCode struct {
	text           string
	newline        string
	memoHasIfdef   bool
	memoHasIfndef  bool
	memoHasInclude bool
}

// Create a new SourceCode struct
func newSourceCode(text string) *SourceCode {
	source := new(SourceCode)
	source.set(text)
	return source
}

func (src *SourceCode) get() string                  { return src.text }
func (src *SourceCode) getNewline() string           { return src.newline }
func (src *SourceCode) has(text string) bool         { return strings.Contains(src.text, text) }
func (src *SourceCode) first(text string) int        { return strings.Index(src.text, text) }
func (src *SourceCode) hasIfdef() bool               { return src.memoHasIfdef }
func (src *SourceCode) hasIfndef() bool              { return src.memoHasIfndef }
func (src *SourceCode) hasInclude() bool             { return src.memoHasInclude }
func (src *SourceCode) firstIfdef() int              { return src.first(ifdef) }
func (src *SourceCode) firstIfndef() int             { return src.first(ifndef) }
func (src *SourceCode) firstInclude() int            { return src.first(incl) }
func (src *SourceCode) nextInclude(pos int) int      { return strings.Index(src.text[pos+1:], incl) }
func (src *SourceCode) firstIncludeAfterIfdef() int  { return src.firstIncludeAfterWord(ifdef) }
func (src *SourceCode) firstIncludeAfterIfndef() int { return src.firstIncludeAfterWord(ifndef) }
func (src *SourceCode) theRest(pos int) *SourceCode  { return newSourceCode(src.text[pos:]) }

func (src *SourceCode) set(text string) {
	src.text = text
	src.newline = src.discoverNewline()
	// memoization
	src.memoHasIfdef = src.has(ifdef)
	src.memoHasIfndef = src.has(ifndef)
	src.memoHasInclude = src.has(incl)
}

func (src *SourceCode) discoverNewline() string {
	// If there is a \r\n, assume it's DOS/Windows line endings
	if strings.Contains(src.text, dosEOL) {
		return dosEOL
	}
	return unixEOL
}

func (src *SourceCode) hasIfdefBefore(pos int) bool {
	found := src.firstIfdef()
	return (found != -1) && (found < pos)
}

func (src *SourceCode) firstIncludeAfterWord(word string) int {
	if !src.hasInclude() && !src.has(word) {
		return 0
	}
	if src.hasInclude() && !src.has(word) {
		return src.firstInclude()
	}
	pos := src.first(word)
	tail := src.theRest(pos)
	if tail.hasInclude() {
		return pos + tail.firstInclude()
	}
	return pos
}

func (src *SourceCode) endofline(pos int) int {
	// Finds the end of the line at the given position
	tail := src.theRest(pos + 1)
	npos := tail.first(src.newline)
	if npos != -1 {
		return pos + 1 + npos
	}
	return 0
}

// Try to find an appropriate insertion position for new includes
func (src *SourceCode) findInsertPos() int {
	const (
		HAS_INCLUDE = 1 << iota
		HAS_IFDEF
		HAS_IFNDEF
	)

	n := 0
	if src.hasInclude() {
		n |= HAS_INCLUDE
	}
	if src.hasIfdef() {
		n |= HAS_IFDEF
	}
	if src.hasIfndef() {
		n |= HAS_IFNDEF
	}

	pos := 0
	switch n {
	case HAS_INCLUDE:
		pos = src.firstInclude()
	case HAS_IFDEF:
		pos = src.firstIfdef()
	case HAS_IFNDEF:
		pos = src.firstIfndef()
	case HAS_IFDEF | HAS_IFNDEF:
		pos = min(src.firstIfdef(), src.firstIfndef())
	case HAS_INCLUDE | HAS_IFDEF:
		pos = src.firstIncludeAfterIfdef()
	case HAS_INCLUDE | HAS_IFNDEF:
		pos = src.firstIncludeAfterIfndef()
	case HAS_INCLUDE | HAS_IFDEF | HAS_IFNDEF:
		pos = min(src.firstIncludeAfterIfdef(), src.firstIncludeAfterIfndef())
	default:
		return 0
	}
	return src.endofline(pos)
}

// Try to expand include-strings (for instance, "stdin" becomes "#include <stdin.h>")
func expandInclude(include string, cppStyle bool) string {

	if !strings.Contains(include, " ") {
		// Include is just a word
		if !strings.Contains(include, "<") && !strings.Contains(include, "\"") {
			// ...and needs brackets
			if !cppStyle && !strings.Contains(include, ".") {
				// Add .h if it is missing
				include = include + ".h"
			}
			// Add brackets
			return incl + " <" + include + ">"
		}
		// ...and does not need brackets
		if !cppStyle && !strings.Contains(include, ".") {
			// Add .h if it is missing, inside the brackets
			bracketchar := include[len(include)-1:]
			include = include[0:len(include)-1] + ".h" + bracketchar
			//include = include + bracketchar
		}
		return incl + " " + include
	}

	// Include is two words?
	if strings.Count(include, " ") == 1 {
		spacepos := strings.Index(include, " ")
		firstword := include[0:spacepos]
		tail := include[spacepos+1:]
		if firstword != incl {
			return expandInclude(tail, cppStyle)
		}
		// We have the second word, now fix it up
		return expandInclude(tail, cppStyle)
	}

	// Quit
	fmt.Fprintf(os.Stderr, "Unusual include: %s\n", include)
	os.Exit(3)
	return include
}

func addIncludeToFile(filename, include string, fixInclude, atTop, cppStyle bool) {
	var (
		source       SourceCode
		fixedInclude string
	)
	if fixInclude {
		fixedInclude = expandInclude(include, cppStyle)
	} else {
		fixedInclude = include
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
	if !atTop {
		pos = source.findInsertPos()
	}

	newline := source.getNewline()
	newtext := filetext[:pos] + newline + fixedInclude + newline + filetext[pos:]
	ioutil.WriteFile(filename, []byte(newtext), 0)
}

func main() {

	nofixText := "don't change the include text"
	topText := "add the include at the top"
	versionText := "show the current version"
	cppText := "don't add .h to the include name"
	verboseText := "more verbose output"
	helpText := "this brief help"

	flag.Usage = func() {
		fmt.Println(versionString)
		fmt.Println()
		fmt.Println("adds an include to a C header- or source file")
		fmt.Println()
		fmt.Println("Arguments:")
		fmt.Println("\tfilename, include")
		fmt.Println("\t-n or --nofix\t\t", nofixText)
		fmt.Println("\t-t or --top\t\t", topText)
		fmt.Println("\t-v or --version\t\t", versionText)
		fmt.Println("\t+ or --c++\t\t", cppText)
		fmt.Println("\t-v or --verbose\t\t", verboseText)
		fmt.Println("\t-h or --help\t\t", helpText)
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("\taddinclude file.h '#include <string.h>'")
		fmt.Println("\taddinclude --top file.h stdlib")
		fmt.Println("\taddinclude file.h '\"some.h\"'")
		fmt.Println("\taddinclude file.cpp memory")
		fmt.Println()
	}

	var (
		missingArgs = func() {
			fmt.Fprintf(os.Stderr, "Needs a filename and an include. Use --help for more info.\n")
			os.Exit(1)
		}

		/* I wish there were support for long and short options in the flag package */
		nofixShort = flag.Bool("n", false, nofixText)
		nofixLong  = flag.Bool("nofix", false, nofixText)

		topShort = flag.Bool("t", false, topText)
		topLong  = flag.Bool("top", false, topText)

		versionShort = flag.Bool("v", false, versionText)
		versionLong  = flag.Bool("version", false, versionText)

		cppShort = flag.Bool("+", false, cppText)
		cppLong  = flag.Bool("c++", false, cppText)

		helpShort = flag.Bool("h", false, helpText)
		helpLong  = flag.Bool("help", false, helpText)

		verboseShort = flag.Bool("V", false, verboseText)
		verboseLong  = flag.Bool("verbose", false, verboseText)
	)

	flag.Parse()

	nofixFlag := *nofixLong || *nofixShort
	topFlag := *topLong || *topShort
	versionFlag := *versionLong || *versionShort
	cppFlag := *cppLong || *cppShort
	verboseFlag := *verboseLong || *verboseShort
	helpFlag := *helpLong || *helpShort

	args := flag.Args()

	if helpFlag {
		flag.Usage()
	} else if versionFlag {
		fmt.Println(versionString)
	} else if len(args) == 2 {
		filename := flag.Arg(0)
		include := flag.Arg(1)
		cppFile := strings.HasSuffix(filename, ".cpp")
		if verboseFlag {
			fmt.Println("C++ mode:", cppFile || cppFlag)
		}
		// Notice the !
		addIncludeToFile(filename, include, !nofixFlag, topFlag, cppFile || cppFlag)
	} else {
		missingArgs()
	}
}
