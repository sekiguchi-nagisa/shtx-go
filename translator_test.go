package main

import (
	"bytes"
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

var evalTestCases = map[string]struct {
	before string
	after  string
}{
	"simple-command-echo": {"echo hello", `{
  echo hello
}
`},
	"multiple-command": {`ls   -la
ps ax  # this is a comment
# comment
false;
`, `{
  ls -la
  ps ax
  false
}
`},
	"comment": {"# this is a comment", `{
}
`},
	"simple-command-seq": {"echo AAA; echo BBB; echo CCC;", `{
  echo AAA
  echo BBB
  echo CCC
}
`},
	"dollar-string": {`echo hello'he\y'$'\x00qq\na'`, `{
  echo hello'he\y'$'\x00qq\na'
}
`},
	"non-literal-command1": {`e'ch''o' hello`, `{
  __shtx_dyna_call e'ch''o' hello
}
`},
	"non-literal-command2": {`"echo" $"hello"\ \ 'world'`, `{
  __shtx_dyna_call "echo" "hello"\ \ 'world'
}
`},
	"redirection1": {"echo 1>& 3", `{
  echo 1>& 3
}
`},
	"redirection2": {"echo 1 >&2", `{
  echo 1 >& 2
}
`},
	"redirection3": {"echo >>hoge", `{
  echo >> hoge
}
`},
	"redirection4": {"echo &>>hoge", `{
  echo &>> hoge
}
`},
	"redirection5": {"echo &>hoge", `{
  echo &> hoge
}
`},
	"redirection6": {"echo <hoge", `{
  echo < hoge
}
`},
	"redirection7": {"echo jfira<&34", `{
  echo jfira <& 34
}
`},
	"redirection8": {"echo 12 >| /dev/null", `{
  echo 12 >| /dev/null
}
`},
	"back-quote": {"echo \"`echo hello`\" `  # this is a comment` A", `{
  echo "$(echo hello)"  A
}
`},
	"back-quote-escape": {"printf '%s\\n'  \"`echo \\\"$BASH\\\"`\"", `{
  __shtx_printf '%s\n' "$(echo "${{__shtx_var_get $? 'BASH'; $REPLY; }}")"
}
`},
	"cmd-sub": {`"$(echo "$(echo AAA; echo BBB)")"`, `{
  __shtx_dyna_call "$(echo "$({
    echo AAA
    echo BBB
  })")"
}
`},
	"env-assign1": {`AAA=12 BBB="$(false)" CCC=`, `{
  __shtx_var_set AAA "12"; __shtx_var_set BBB "$(false)"; __shtx_var_set CCC 
}
`},
	"env_assign2": {
		`AA=12 BB=34 echo`, `{
  AA="12" BB="34" echo
}
`},
	"param-expand": {`echo "$AAA" ge"(${GGG}}"`, `{
  echo "${{__shtx_var_get $? 'AAA'; $REPLY; }}" ge"(${{__shtx_var_get $? 'GGG'; $REPLY; }}}"
}
`},
	"escaped-simple-command": {`\ls -la`, `{
  \ls -la
}
`},
	"builtin-export1": {`\expor\t AAA=@@@; export BBB CCC=56`, `{
  __shtx_export AAA=@@@
  __shtx_export BBB CCC="56"
}
`},
	"builtin-export2": {
		`\expo\rt AAA=12 BBB=45`, `{
  __shtx_export AAA=12 BBB=45
}
`},
	"builtin-local": {`local aaa; local -q; local aaa=`, `{
  __shtx_local aaa
  __shtx_local "-q"
  __shtx_local aaa=
}
`},
	"builtin-declare": {`"declare"; declare -f hoge; declare AAA=12`, `{
  __shtx_dyna_call "declare"
  __shtx_declare "-f" hoge
  __shtx_declare AAA="12"
}
`},
	"builtin-unset": {`\uns\e\t A; unset B`, `{
  __shtx_unset A
  __shtx_unset B
}
`},
	"builtin-eval": {`\eval echo hello`, `{
  fake_eval echo hello
}
`},
	"builtin-shift1": {"shift 2", `{
  __shtx_shift 2
}
`},
	"builtin-shift2": {`shif\t`, `{
  __shtx_shift
}
`},
	"builtin-test1": {`[ -n hoge ]`, `{
  __shtx_[ -n hoge ]
}
`},
	"builtin-test2": {`\[ -n hoge ]`, `{
  __shtx_[ -n hoge ]
}
`},

	"builtin-test3": {`\\[ -n hoge ]`, `{
  \\[ -n hoge ]
}
`},
	"builtin-read": {`read $?`, `{
  __shtx_read ${{__shtx_var_get $? '?'; $REPLY; }}
}
`},
	"non-callable-command1": {`__shtx_printf`, `{
  __shtx_dyna_call __shtx_printf
}
`},
	"non-callable-command2": {`\__shtx_printf`, `{
  __shtx_dyna_call \__shtx_printf
}
`},
	"non-callable-command3": {`fake_eval`, `{
  __shtx_dyna_call fake_eval
}
`},
	"non-callable-command4": {`fake_\source`, `{
  __shtx_dyna_call fake_\source
}
`},
	"special-param1": {`echo "$#: $0: $1 ${002}"`, `{
  echo "${{__shtx_var_get $? '#'; $REPLY; }}: ${{__shtx_var_get $? '0'; $REPLY; }}: ${{__shtx_var_get $? '1'; $REPLY; }} ${{__shtx_var_get $? '002'; $REPLY; }}"
}
`},
	"special-param2": {`echo $#: "$0: $1 ${002}"`, `{
  echo ${{__shtx_var_get $? '#'; $REPLY; }}: "${{__shtx_var_get $? '0'; $REPLY; }}: ${{__shtx_var_get $? '1'; $REPLY; }} ${{__shtx_var_get $? '002'; $REPLY; }}"
}
`},
	"special-param3": {
		`echo "$?"`, `{
  echo "${{__shtx_var_get $? '?'; $REPLY; }}"
}
`},
	"special-param4": {`echo "$?"`, `{
  echo "${{__shtx_var_get $? '?'; $REPLY; }}"
}
`},
	"special-param5": {
		`printf "$*"`, `{
  __shtx_printf "${{__shtx_var_get $? '*'; $REPLY; }}"
}
`},
	"special-param6": {`echo "$@"`, `{
  echo $__shtx_get_args()
}
`},
	"special-param7": {`echo 12"$@"`, `{
  echo $__shtx_concat(new [Any]()
    .add( @(12"")[0] )
    .add($__shtx_get_args())
    .add( @("")[0] )
  )
}
`},
	"special-param8": {`echo 23"4${@}5$?"6`, `{
  echo $__shtx_concat(new [Any]()
    .add( @(23"4")[0] )
    .add($__shtx_get_args())
    .add( @("5${{__shtx_var_get $? '?'; $REPLY; }}"6)[0] )
  )
}
`},
	"binary1": {
		`true && echo | grep`, `{
  (true && (echo | grep))
}
`},
	"binary2": {`echo | grep || echo`, `{
  ((echo | grep) || echo)
}
`},
	"binary3": {`true 1 && false 1 || true 2 && false 2`,
		`{
  (((true 1 && false 1) || true 2) && false 2)
}
`},
	"negate": {`! echo hello`, `{
  ! echo hello
}
`},
	"group": {
		`{ echo 1; echo 2;}`, `{
  {
    echo 1
    echo 2
  }
}
`},
	"if1": {`if true 1; then
echo true 1
fi`, `{
  if (true 1) {
    echo true 1
  }
}
`},
	"if2": {`if true 1; then
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
	"if3": {`if true 1; then
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
	"if4": {`
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
	"param-expand-op1": {`echo "${hoge:-45}"`, `{
  echo "${{__shtx_var_get $? 'hoge' ':-' 45; $REPLY; }}"
}
`},
	"param-expand-op2": {`echo "${45-hoge}"`, `{
  echo "${{__shtx_var_get $? '45' '-' hoge; $REPLY; }}"
}
`},
	"param-expand-op3": {`echo "${?:?hoge}"`, `{
  echo "${{__shtx_var_get $? '?' ':?' hoge; $REPLY; }}"
}
`},
	"param-expand-op4": {`echo "${var=}"`, `{
  echo "${{__shtx_var_get $? 'var' '=' ; $REPLY; }}"
}
`},
	"function1": {`function hoge() true`, `{
  $__shtx_func('hoge', (){
    let ctx = $__shtx_enter_func($0, $@)
    defer { $__shtx_exit_func($ctx); }
    true
  })
}
`},
	"function2": {`hoge() { echo hello; } > /dev/null`, `{
  $__shtx_func('hoge', (){
    let ctx = $__shtx_enter_func($0, $@)
    defer { $__shtx_exit_func($ctx); }
    {
      echo hello
    } with > /dev/null
  })
}
`},
	"function3": {`ff() { local AAA BBB=12; local CCC=12 && echo "$CCC"; }`, `{
  $__shtx_func('ff', (){
    let ctx = $__shtx_enter_func($0, $@)
    defer { $__shtx_exit_func($ctx); }
    {
      __shtx_local AAA BBB="12"
      (__shtx_local CCC="12" && echo "${{__shtx_var_get $? 'CCC'; $REPLY; }}")
    }
  })
}
`},
	"case1": {
		`case "$1" in
shell|rehash) echo match
  case "-$2" in
  -s) echo '-s';;
  -l) echo '-l';;
  esac ;;
*) echo default
esac
`, `{
  {
    var case_1=''; case_1="${{__shtx_var_get $? '1'; $REPLY; }}"
    if $case_1 =~ $/^shell$/ || $case_1 =~ $/^rehash$/ {
      echo match
      {
        var case_2=''; case_2="-${{__shtx_var_get $? '2'; $REPLY; }}"
        if $case_2 =~ $/^-s$/ {
          echo '-s'
        }
        elif $case_2 =~ $/^-l$/ {
          echo '-l'
        }
      }
    }
    elif $case_1 =~ $/^.*$/ {
      echo default
    }
  }
}
`},
	"case2": {
		`case 1234 in
"1234"|'5678') echo 1 ;;
"$HOME") echo 2 ;;
~root) echo 3 ;;
"/*"\**) echo 4 ;;
*) echo default
esac
`, `{
  {
    var case_1=''; case_1="1234"
    if $__shtx_glob_match(@( $case_1 $__shtx_escape_glob_meta("1234") )) || $__shtx_glob_match(@( $case_1 $__shtx_escape_glob_meta('5678') )) {
      echo 1
    }
    elif $__shtx_glob_match(@( $case_1 $__shtx_escape_glob_meta("${{__shtx_var_get $? 'HOME'; $REPLY; }}") )) {
      echo 2
    }
    elif $__shtx_glob_match(@( $case_1 ~"root" )) {
      echo 3
    }
    elif $__shtx_glob_match(@( $case_1 $__shtx_escape_glob_meta("/*")"\**" )) {
      echo 4
    }
    elif $case_1 =~ $/^.*$/ {
      echo default
    }
  }
}
`},
	"assign-param-expand": {
		`AAA=$aaa`, `{
  __shtx_var_set AAA ${{__shtx_var_get $? 'aaa'; $REPLY; }}
}
`},
	"assign-cmdsub": {
		`AAA=$(echo a b c)`, `{
  __shtx_var_set AAA "$(echo a b c)"
}
`},
	"assign-glob": {
		`AAA=*\*`, `{
  __shtx_var_set AAA "**"
}
`},
	"assign-special1": {
		`AAA=$@`, `{
  __shtx_var_set AAA ${{__shtx_var_get $? '@'; $REPLY; }}
}
`},
	"assign-special2": {
		`AAA="$@"`, `{
  __shtx_var_set AAA "${{__shtx_var_get $? '@'; $REPLY; }}"
}
`},
	"assign-special3": {
		`AAA=$*`, `{
  __shtx_var_set AAA ${{__shtx_var_get $? '*'; $REPLY; }}
}
`},
}

func TestEval(t *testing.T) {
	for name, testCase := range evalTestCases {
		tx := NewTranslator(TranslateEval)
		assert.NotNil(t, tx)

		r := []byte(testCase.before)
		buf := bytes.Buffer{}

		e := tx.Translate(r, &buf)
		assert.NoError(t, e, fmt.Sprintf("failed: %s", name))

		assert.Equal(t, testCase.after, buf.String(), fmt.Sprintf("failed: %s", name))
	}
}

var sourceTestCases = map[string]struct {
	before string
	after  string
}{
	"empty": {``, `function(argv : [String]) => {
  let old_argv = $__shtx_set_argv($argv)
  defer { $__shtx_set_argv($old_argv); }
}
`},
	"simple-command": {`echo hello`, `function(argv : [String]) => {
  let old_argv = $__shtx_set_argv($argv)
  defer { $__shtx_set_argv($old_argv); }
  echo hello
}
`},
}

func TestSource(t *testing.T) {
	for name, testCase := range sourceTestCases {
		tx := NewTranslator(TranslateSource)
		assert.NotNil(t, tx)

		r := []byte(testCase.before)
		buf := bytes.Buffer{}

		e := tx.Translate(r, &buf)
		assert.NoError(t, e, fmt.Sprintf("failed: %s", name))

		assert.Equal(t, testCase.after, buf.String(), fmt.Sprintf("failed: %s", name))
	}
}
