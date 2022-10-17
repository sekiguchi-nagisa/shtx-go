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
	{"echo hello", `{
  echo hello
}
`},
	{`ls   -la
ps ax  # this is a comment
# comment
false;
`, `{
  ls -la
  ps ax
  false
}
`},
	{"# this is a comment", `{
}
`},
	{"echo AAA; echo BBB; echo CCC;", `{
  echo AAA
  echo BBB
  echo CCC
}
`},
	{`echo hello'he\y'$'\x00qq\na'`, `{
  echo hello'he\y'$'\x00qq\na'
}
`},
	{`e'ch''o' hello`, `{
  __shtx_dyna_call e'ch''o' hello
}
`},
	{`"echo" $"hello"\ \ 'world'`, `{
  __shtx_dyna_call "echo" "hello"\ \ 'world'
}
`},
	{"echo 1>& 3", `{
  echo 1>&3
}
`},
	{"echo 1 >&2", `{
  echo 1 >&2
}
`},
	{"echo >>hoge", `{
  echo >>hoge
}
`},
	{"echo &>>hoge", `{
  echo &>>hoge
}
`},
	{"echo &>hoge", `{
  echo &>hoge
}
`},
	{"echo <hoge", `{
  echo <hoge
}
`},
	{"echo `echo hello` `  # this is a comment` A", `{
  echo $(echo hello)  A
}
`},
	{`$(echo "$(echo AAA; echo BBB)")`, `{
  __shtx_dyna_call $(echo "$({
    echo AAA
    echo BBB
  })")
}
`},
	{`AAA=12 BBB="$(false)" CCC=`, `{
  __shtx_var_set AAA 12; __shtx_var_set BBB "$(false)"; __shtx_var_set CCC 
}
`},
	{
		`AA=12 BB=34 echo`, `{
  AA=12 BB=34 echo
}
`},
	{`echo "$AAA" ge"(${GGG}}"`, `{
  echo "${$__shtx_var_get('AAA')}" ge"(${$__shtx_var_get('GGG')}}"
}
`},
	{`\ls -la`, `{
  \ls -la
}
`},
	{`\expor\t AAA=@@@; export BBB CCC=56`, `{
  __shtx_export AAA=@@@
  __shtx_export BBB CCC=56
}
`},
	{
		`\expo\rt AAA=12 BBB=45`, `{
  __shtx_export AAA=12 BBB=45
}
`},
	{`\uns\e\t A; unset B`, `{
  __shtx_unset A
  __shtx_unset B
}
`},
	{`\eval echo hello`, `{
  fake_eval echo hello
}
`},
}

func TestBase(t *testing.T) {
	for _, testCase := range testCases {
		tx := NewTranslator(TranslateEval)
		assert.NotNil(t, tx)

		r := strings.NewReader(testCase.before)
		buf := bytes.Buffer{}

		e := tx.Translate(r, &buf)
		assert.Nil(t, e)

		assert.Equal(t, testCase.after, buf.String())
	}
}
