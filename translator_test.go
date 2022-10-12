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
	{"echo hello", `function(argv : [String]) => {
  echo hello
}
`},
	{`ls   -la
ps ax  # this is a comment
# comment
false;
`, `function(argv : [String]) => {
  ls -la
  ps ax
  false
}
`},
	{"# this is a comment", `function(argv : [String]) => {
}
`},
	{"echo AAA; echo BBB; echo CCC;", `function(argv : [String]) => {
  echo AAA
  echo BBB
  echo CCC
}
`},
	{`echo hello'he\y'$'\x00qq\na'`, `function(argv : [String]) => {
  echo hello'he\y'$'\x00qq\na'
}
`},
	{`e'ch''o' hello`, `function(argv : [String]) => {
  __shtx_dyna_call e'ch''o' hello
}
`},
	{`"echo" $"hello"\ \ 'world'`, `function(argv : [String]) => {
  __shtx_dyna_call "echo" "hello"\ \ 'world'
}
`},
	{"echo 1>& 3", `function(argv : [String]) => {
  echo 1>&3
}
`},
	{"echo 1 >&2", `function(argv : [String]) => {
  echo 1 >&2
}
`},
	{"echo >>hoge", `function(argv : [String]) => {
  echo >>hoge
}
`},
	{"echo &>>hoge", `function(argv : [String]) => {
  echo &>>hoge
}
`},
	{"echo &>hoge", `function(argv : [String]) => {
  echo &>hoge
}
`},
	{"echo <hoge", `function(argv : [String]) => {
  echo <hoge
}
`},
	{"echo `echo hello` `  # this is a comment` A", `function(argv : [String]) => {
  echo $(echo hello)  A
}
`},
	{`$(echo "$(echo AAA; echo BBB)")`, `function(argv : [String]) => {
  __shtx_dyna_call $(echo "$({
    echo AAA
    echo BBB
  })")
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
