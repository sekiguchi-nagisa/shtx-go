package main

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

var testCases = []struct {
	before string
	after  string
}{
	{"echo hello", `function(args : [String]) => {
  echo hello
}
`},
	{`ls   -la
ps ax  # this is a comment
# comment
false;
`, `function(args : [String]) => {
  ls -la
  ps ax
  false
}
`},
	{"# this is a comment", `function(args : [String]) => {
}
`},
	{"echo AAA; echo BBB; echo CCC;", `function(args : [String]) => {
  echo AAA
  echo BBB
  echo CCC
}
`},
	{`echo hello'he\y'$'\x00qq\na'`, `function(args : [String]) => {
  echo hello'he\y'$'\x00qq\na'
}
`},
	{`e'ch''o' hello`, `function(args : [String]) => {
  __shtx_dyna_call e'ch''o' hello
}
`},
}

func TestBase(t *testing.T) {
	for _, testCase := range testCases {
		tx := NewTranslator()
		assert.NotNil(t, tx)

		r := strings.NewReader(testCase.before)
		buf := bytes.Buffer{}

		e := tx.Translate(r, &buf)
		assert.Nil(t, e)

		assert.Equal(t, testCase.after, buf.String())
	}
}
