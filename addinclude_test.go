package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewl(t *testing.T) {
	testcontent1 := "a\r\nb\r\nc"
	testcontent2 := "a\n\b\nc"
	source := newSourceCode(testcontent1)
	assert.Equal(t, dosEOL, source.getNewline())
	source.set(testcontent2)
	assert.Equal(t, unixEOL, source.getNewline())
}

func TestRememberHasIfdef(t *testing.T) {
	testcontent1 := "blabla\n#ifdef ost"
	testcontent2 := "blablabla"
	source := newSourceCode(testcontent1)
	assert.True(t, source.hasIfdef())
	source.set(testcontent2)
	assert.False(t, source.hasIfdef())
}

func TestTestfile1(t *testing.T) {
	testcontent := `#ifdef SOMETHING

#include <blubbelubb.h>

#define SOMETHING
#endif /* SOMETHING */
`
	source := newSourceCode(testcontent)
	assert.Equal(t, 41, source.findInsertPos())
}

func TestTestfile2(t *testing.T) {
	testcontent := `#include "paraply.h"
`
	source := newSourceCode(testcontent)
	assert.Equal(t, 20, source.findInsertPos())
}

func TestTestfile3(t *testing.T) {
	testcontent := `#ifdef SOMETHING
#define SOMETHING
#endif`
	source := newSourceCode(testcontent)
	assert.Equal(t, 16, source.findInsertPos())
}

func TestTestfile4(t *testing.T) {
	testcontent := ``
	source := newSourceCode(testcontent)
	assert.Equal(t, 0, source.findInsertPos())
}

func TestTestfile5(t *testing.T) {
	testcontent := `#include "jeje.h"

#ifdef SOMETHING

#include "ostebolle.h"

#endif`
	source := newSourceCode(testcontent)
	assert.Equal(t, 59, source.findInsertPos())
}

func TestFixInclu(t *testing.T) {
	assert.Equal(t, "#include <stdlib.h>", expandInclude("bolle stdlib", false))
	assert.Equal(t, "#include <stdlib.h>", expandInclude("#include <stdlib.h>", false))
	assert.Equal(t, "#include <stdlib.h>", expandInclude("include <stdlib.h>", false))
	assert.Equal(t, "#include \"stdlib.h\"", expandInclude("#include \"stdlib.h\"", false))
	assert.Equal(t, "#include <stdlib.h>", expandInclude("stdlib", false))
	assert.Equal(t, "#include \"stdlib.h\"", expandInclude("\"stdlib\"", false))
	assert.Equal(t, "#include <stdlib.h>", expandInclude("<stdlib>", false))
	assert.Equal(t, "#include <memory>", expandInclude("memory", true))
}

func TestTestfile6(t *testing.T) {
	testcontent := `#include "jeje.h"

#ifdef SOMETHING

#include "ostebolle.h"
#include <stdlib.h>


#endif`
	source := newSourceCode(testcontent)
	assert.Equal(t, 59, source.findInsertPos())
}
