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
