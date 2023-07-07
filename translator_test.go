package main

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"testing"
)

var evalTestCases = []struct {
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
  echo 1>& 3
}
`},
	{"echo 1 >&2", `{
  echo 1 >& 2
}
`},
	{"echo >>hoge", `{
  echo >> hoge
}
`},
	{"echo &>>hoge", `{
  echo &>> hoge
}
`},
	{"echo &>hoge", `{
  echo &> hoge
}
`},
	{"echo <hoge", `{
  echo < hoge
}
`},
	{"echo jfira<&34", `{
  echo jfira <& 34
}
`},
	{"echo 12 >| /dev/null", `{
  echo 12 >| /dev/null
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
  echo "${{__shtx_var_get $? 'AAA'; $REPLY; }}" ge"(${{__shtx_var_get $? 'GGG'; $REPLY; }}}"
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
	{`echo "$#: $0: $1 ${002}"`, `{
  echo "${{__shtx_var_get $? '#'; $REPLY; }}: ${{__shtx_var_get $? '0'; $REPLY; }}: ${{__shtx_var_get $? '1'; $REPLY; }} ${{__shtx_var_get $? '002'; $REPLY; }}"
}
`},
	{`echo $#: "$0: $1 ${002}"`, `{
  echo ${{__shtx_var_get $? '#'; $REPLY; }}: "${{__shtx_var_get $? '0'; $REPLY; }}: ${{__shtx_var_get $? '1'; $REPLY; }} ${{__shtx_var_get $? '002'; $REPLY; }}"
}
`},
	{"shift 2", `{
  __shtx_shift 2
}
`},
	{`shif\t`, `{
  __shtx_shift
}
`},
	{`[ -n hoge ]`, `{
  __shtx_[ -n hoge ]
}
`},
	{`\[ -n hoge ]`, `{
  __shtx_[ -n hoge ]
}
`},

	{`\\[ -n hoge ]`, `{
  __shtx_dyna_call \\[ -n hoge ]
}
`},
	{
		`echo "$?"`, `{
  echo "${{__shtx_var_get $? '?'; $REPLY; }}"
}
`},
	{
		`true && echo | grep`, `{
  (true && (echo | grep))
}
`},
	{`echo | grep || echo`, `{
  ((echo | grep) || echo)
}
`},
	{`true 1 && false 1 || true 2 && false 2`,
		`{
  (((true 1 && false 1) || true 2) && false 2)
}
`},
	{
		`{ echo 1; echo 2;}`, `{
  {
    echo 1
    echo 2
  }
}
`},
	{`if true 1; then
echo true 1
fi`, `{
  if (true 1) {
    echo true 1
  }
}
`},
	{`if true 1; then
  echo true 1
elif true 2; then
  echo true 2; echo true 22
fi
`, `{
  if (true 1) {
    echo true 1
  } elif (true 2) {
    echo true 2
    echo true 22
  }
}
`},
	{`if true 1; then
  echo true 1
else
  echo false
fi
`, `{
  if (true 1) {
    echo true 1
  } else {
    echo false
  }
}
`},
	{`
if echo false; true 1; then
  if true 11; then
    echo true 11
  elif true 2; then
     :
  fi
else
  echo false
fi
`, `{
  if {
    echo false
    true 1
  } {
    if (true 11) {
      echo true 11
    } elif (true 2) {
      :
    }
  } else {
    echo false
  }
}
`},
	{`echo "${hoge:-45}"`, `{
  echo "${{__shtx_var_get $? 'hoge' ':-' 45; $REPLY; }}"
}
`},
	{`echo "${45-hoge}"`, `{
  echo "${{__shtx_var_get $? '45' '-' hoge; $REPLY; }}"
}
`},
	{`echo "${?:?hoge}"`, `{
  echo "${{__shtx_var_get $? '?' ':?' hoge; $REPLY; }}"
}
`},
	{`echo "${var=}"`, `{
  echo "${{__shtx_var_get $? 'var' '=' ; $REPLY; }}"
}
`},
	{`function hoge() true`, `{
  $__shtx_func('hoge', (){
    $__shtx_enter_func($0, $@)
    defer { $__shtx_exit_func(); }
    true
  })
}
`},
	{`hoge() { echo hello; } > /dev/null`, `{
  $__shtx_func('hoge', (){
    $__shtx_enter_func($0, $@)
    defer { $__shtx_exit_func(); }
    {
      echo hello
    } with > /dev/null
  })
}
`},
	{`ff() { local AAA BBB=12; local CCC=12 && echo "$CCC"; }`, `{
  $__shtx_func('ff', (){
    $__shtx_enter_func($0, $@)
    defer { $__shtx_exit_func(); }
    {
      __shtx_local AAA BBB=12
      (__shtx_local CCC=12 && echo "${{__shtx_var_get $? 'CCC'; $REPLY; }}")
    }
  })
}
`},
	{`echo $?`, `{
  echo ${{__shtx_var_get $? '?'; $REPLY; }}
}
`},
	{`echo "$?"`, `{
  echo "${{__shtx_var_get $? '?'; $REPLY; }}"
}
`},
}

func TestEval(t *testing.T) {
	for _, testCase := range evalTestCases {
		tx := NewTranslator(TranslateEval)
		assert.NotNil(t, tx)

		r := []byte(testCase.before)
		buf := bytes.Buffer{}

		e := tx.Translate(r, &buf)
		assert.NoError(t, e)

		assert.Equal(t, testCase.after, buf.String())
	}
}

var sourceTestCases = []struct {
	before string
	after  string
}{
	{``, `function(argv : [String]) => {
  let old_argv = $__shtx_set_argv($argv)
  defer { $__shtx_set_argv($old_argv); }
}
`},
	{`echo hello`, `function(argv : [String]) => {
  let old_argv = $__shtx_set_argv($argv)
  defer { $__shtx_set_argv($old_argv); }
  echo hello
}
`},
}

func TestSource(t *testing.T) {
	for _, testCase := range sourceTestCases {
		tx := NewTranslator(TranslateSource)
		assert.NotNil(t, tx)

		r := []byte(testCase.before)
		buf := bytes.Buffer{}

		e := tx.Translate(r, &buf)
		assert.NoError(t, e)

		assert.Equal(t, testCase.after, buf.String())
	}
}
