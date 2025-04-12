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
  fake_call e'ch''o' hello
}
`},
	"non-literal-command2": {`"echo" $"hello"\ \ 'world'`, `{
  fake_call "echo" "hello"\ \ 'world'
}
`},
	"non-literal-command3": {`2to3`, `{
  fake_call 2to3
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
	"redirection89": {"declare -p AAA BBB 2>&1", `{
  __shtx_declare "-p" AAA BBB 2>& 1
}
`},
	"back-quote": {"echo \"`echo hello`\" `  # this is a comment` A", `{
  echo "$(echo hello)"  A
}
`},
	"back-quote-escape1": {"printf '%s\\n'  \"`echo \\\"$BASH\\\" '\\\\@'`\"", `{
  __shtx_printf '%s\n' "$(echo "${$__shtx_get_var(@( 'BASH' ))}" '\@')"
}
`},
	"cmd-sub": {`"$(echo "$(echo AAA; echo BBB)")"`, `{
  fake_call "$(echo "$({
    echo AAA
    echo BBB
  })")"
}
`},
	"proc-sub1": {`diff <(ls;echo) <(ps)`, `{
  diff <({
    ls
    echo
  }) <(ps)
}
`},
	"proc-sub2": {`curl -L >(cat -n)`, `{
  curl -L >(cat -n)
}
`},
	"env-assign1": {`AAA=12 BBB="$(false)" CCC=`, `{
  $__shtx_set_var(@( AAA = "12" )); $__shtx_set_var(@( BBB = "$(false)" )); $__shtx_set_var(@( CCC =  ))
}
`},
	"env_assign2": {
		`AA=12 BB=34 echo`, `{
  AA="12" BB="34" echo
}
`},
	"param-expand": {`echo "$AAA" ge"(${GGG}}"`, `{
  echo "${$__shtx_get_var(@( 'AAA' ))}" ge"(${$__shtx_get_var(@( 'GGG' ))}}"
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
  fake_call "declare"
  __shtx_declare "-f" hoge
  __shtx_declare AAA="12"
}
`},
	"builtin-typeset": {`typeset 1234; \typeset`, `{
  __shtx_typeset "1234"
  __shtx_typeset
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
  fake_call \\[ -n hoge ]
}
`},
	"builtin-read": {`read $?`, `{
  __shtx_read ${$__shtx_get_var(@( '?' ))}
}
`},
	"builtin-trap": {`trap -- "" INT`, `{
  __shtx_trap -- "" INT
}
`},
	"builtin-builtin": {`builtin ls; "builtin" echo`, `{
  __shtx_builtin ls
  fake_call "builtin" echo
}
`},
	"non-callable-command1": {`__shtx_printf`, `{
  fake_call __shtx_printf
}
`},
	"non-callable-command2": {`\__shtx_printf`, `{
  fake_call \__shtx_printf
}
`},
	"non-callable-command3": {`fake_eval`, `{
  fake_call fake_eval
}
`},
	"non-callable-command4": {`fake_\source`, `{
  fake_call fake_\source
}
`},
	"special-param1": {`echo "$#: $0: $1 ${002}"`, `{
  echo "${$__shtx_get_var(@( '#' ))}: ${$__shtx_get_var(@( '0' ))}: ${$__shtx_get_var(@( '1' ))} ${$__shtx_get_var(@( '002' ))}"
}
`},
	"special-param2": {`echo $#: "$0: $1 ${002}"`, `{
  echo ${$__shtx_get_var(@( '#' ))}: "${$__shtx_get_var(@( '0' ))}: ${$__shtx_get_var(@( '1' ))} ${$__shtx_get_var(@( '002' ))}"
}
`},
	"special-param3": {
		`echo "$?"`, `{
  echo "${$__shtx_get_var(@( '?' ))}"
}
`},
	"special-param4": {`echo "$?"`, `{
  echo "${$__shtx_get_var(@( '?' ))}"
}
`},
	"special-param5": {
		`printf "$*"`, `{
  __shtx_printf "${$__shtx_get_var(@( '*' ))}"
}
`},
	"special-param6": {`echo "$@"`, `{
  echo $__shtx_get_array_var('@')
}
`},
	"special-param7": {`echo 12"$@"`, `{
  echo $__shtx_concat(new [Any]()
    .add( @(12"")[0] )
    .add($__shtx_get_array_var('@'))
    .add( @("")[0] )
  )
}
`},
	"special-param8": {`echo 23"4${@}5$?"6`, `{
  echo $__shtx_concat(new [Any]()
    .add( @(23"4")[0] )
    .add($__shtx_get_array_var('@'))
    .add( @("5${$__shtx_get_var(@( '?' ))}"6)[0] )
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
	"negate1": {`! echo hello`, `{
  ! echo hello
}
`},
	"negate2": {`! ls | grep ds`, `{
  ! (ls | grep ds)
}
`},
	"negate3": {`! [[ -e ~/test ]]`, `{
  ! (test -e ~"/test")
}
`},
	"subshell": {`(ls&&echo hello;echo hey)`, `{
  (&({
    (ls && echo hello)
    echo hey
  }))
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
  if $__shtx_cond(true 1) {
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
  if $__shtx_cond(true 1) {
    echo true 1
  } elif $__shtx_cond(true 2) {
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
  if $__shtx_cond(true 1) {
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
  if $__shtx_cond({
    echo false
    true 1
  }) {
    if $__shtx_cond(true 11) {
      echo true 11
    } elif $__shtx_cond(true 2) {
      :
    }
  } else {
    echo false
  }
}
`},
	"param-expand-op1": {`echo "${hoge:-45}"`, `{
  echo "${$__shtx_get_var(@( 'hoge' ':-' "45" ))}"
}
`},
	"param-expand-op2": {`echo "${45-hoge}"`, `{
  echo "${$__shtx_get_var(@( '45' '-' "hoge" ))}"
}
`},
	"param-expand-op3": {`echo "${?:?hello world}"`, `{
  echo "${$__shtx_get_var(@( '?' ':?' "hello world" ))}"
}
`},
	"param-expand-op4": {`echo "${var=}"`, `{
  echo "${$__shtx_get_var(@( 'var' '='  ))}"
}
`},
	"param-expand-replace1": {`echo "${a/hello/world}"`, `{
  echo "${$__shtx_get_var(@( 'a' '/' "hello" "world" ))}"
}
`},
	"param-expand-replace2": {`echo "${a//~"root"/}"`, `{
  echo "${$__shtx_get_var(@( 'a' '//' ~""$__shtx_escape_glob_meta("root")  ))}"
}
`},
	"param-expand-replace3": {`echo "${a/#~"root"/$HOME}"`, `{
  echo "${$__shtx_get_var(@( 'a' '/' "#~"$__shtx_escape_glob_meta("root") ${$__shtx_get_var(@( 'HOME' ))} ))}"
}
`},
	"function1": {`function hoge() true`, `{
  let src_0 = "function hoge() true"
  $__shtx_func('hoge', $src_0, (){
    let ctx = $__shtx_enter_func($0, $@)
    defer { $__shtx_exit_func($ctx); }
    try {
      true
    } catch e: _Return { return $e.status(); }
  })
}
`},
	"function2": {`hoge() { echo hello; } > /dev/null`, `{
  let src_0 = "hoge() { echo hello; } > /dev/null"
  $__shtx_func('hoge', $src_0, (){
    let ctx = $__shtx_enter_func($0, $@)
    defer { $__shtx_exit_func($ctx); }
    try {
      {
        echo hello
      } with > /dev/null
    } catch e: _Return { return $e.status(); }
  })
}
`},
	"function3": {`ff() { local AAA BBB=12; local CCC=12 && echo "$CCC"; }`, `{
  let src_0 = "ff() { local AAA BBB=12; local CCC=12 && echo \"\$CCC\"; }"
  $__shtx_func('ff', $src_0, (){
    let ctx = $__shtx_enter_func($0, $@)
    defer { $__shtx_exit_func($ctx); }
    try {
      {
        __shtx_local AAA BBB="12"
        (__shtx_local CCC="12" && echo "${$__shtx_get_var(@( 'CCC' ))}")
      }
    } catch e: _Return { return $e.status(); }
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
    var case_1=''; case_1="${$__shtx_get_var(@( '1' ))}"
    if $case_1 =~ $/^shell$/ || $case_1 =~ $/^rehash$/ {
      echo match
      {
        var case_2=''; case_2="-${$__shtx_get_var(@( '2' ))}"
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
    elif $__shtx_glob_match(@( $case_1 $__shtx_escape_glob_meta("${$__shtx_get_var(@( 'HOME' ))}") )) {
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
	"for1": {`for aa in 1 2 3; do echo "<$aa>"; done`, `{
  for aa in @(1 2 3) {
    $__shtx_enter_loop(); defer { $__shtx_exit_loop(); }
    $__shtx_set_var(['aa', '=', $aa])
    try {
      echo "<${$__shtx_get_var(@( 'aa' ))}>"
    } catch e: _BreakContinue {
      $__shtx_check_loop($e) ? (continue) : (break)
    }
  }
}
`},
	"for2": {`for aaa; do echo "<$aaa>"; done`, `{
  for aaa in $__shtx_get_array_var('@') {
    $__shtx_enter_loop(); defer { $__shtx_exit_loop(); }
    $__shtx_set_var(['aaa', '=', $aaa])
    try {
      echo "<${$__shtx_get_var(@( 'aaa' ))}>"
    } catch e: _BreakContinue {
      $__shtx_check_loop($e) ? (continue) : (break)
    }
  }
}
`},
	"for3": {`for aaa in; do echo "<$aaa>"; done`, `{
  for aaa in @() {
    $__shtx_enter_loop(); defer { $__shtx_exit_loop(); }
    $__shtx_set_var(['aaa', '=', $aaa])
    try {
      echo "<${$__shtx_get_var(@( 'aaa' ))}>"
    } catch e: _BreakContinue {
      $__shtx_check_loop($e) ? (continue) : (break)
    }
  }
}
`},
	"break-continue": {`break; continue; "break"`, `{
  __shtx_break
  __shtx_continue
  fake_call "break"
}
`},
	"assign-param-expand": {
		`AAA=$aaa`, `{
  $__shtx_set_var(@( AAA = ${$__shtx_get_var(@( 'aaa' ))} ))
}
`},
	"assign-cmdsub": {
		`AAA=$(echo a b c)`, `{
  $__shtx_set_var(@( AAA = "$(echo a b c)" ))
}
`},
	"assign-glob": {
		`AAA=*\*`, `{
  $__shtx_set_var(@( AAA = "**" ))
}
`},
	"assign-special1": {
		`AAA=$@`, `{
  $__shtx_set_var(@( AAA = ${$__shtx_get_var(@( '@' ))} ))
}
`},
	"assign-special2": {
		`AAA="$@"`, `{
  $__shtx_set_var(@( AAA = "${$__shtx_get_var(@( '@' ))}" ))
}
`},
	"assign-special3": {
		`AAA=$*`, `{
  $__shtx_set_var(@( AAA = ${$__shtx_get_var(@( '*' ))} ))
}
`},
	"append1": {
		`AAA+=$aaa`, `{
  $__shtx_set_var(@( AAA += ${$__shtx_get_var(@( 'aaa' ))} ))
}
`},
	"test-expr1": {`[[ $HOME && ! ($HOME == *.txt) ]]`, `{
  ((test ${$__shtx_get_var(@( 'HOME' ))}) && (!(($__shtx_glob_match(@( ${$__shtx_get_var(@( 'HOME' ))} "*.txt" ))))))
}
`},
	"test-expr2": {`[[ -x $BASH || $BASH != 'bash' ]]`, `{
  ((test -x ${$__shtx_get_var(@( 'BASH' ))}) || (!$__shtx_glob_match(@( ${$__shtx_get_var(@( 'BASH' ))} $__shtx_escape_glob_meta('bash') ))))
}
`},
	"test-expr3": {`[[ 1234 < 4567 ]]`, `{
  (test "1234" "<" "4567")
}
`},
	"test-expr-regex1": {`[[ $BASH =~ b*sh ]]`, `{
  ($__shtx_regex_match(@( ${$__shtx_get_var(@( 'BASH' ))} "b*sh" )))
}
`},
	"test-expr-regex2": {`[[ $BASH =~ 'b*sh'$PID ]]`, `{
  ($__shtx_regex_match(@( ${$__shtx_get_var(@( 'BASH' ))} $__shtx_escape_regex_meta('b*sh')${$__shtx_get_var(@( 'PID' ))} )))
}
`},
	"return1": {`test -e 23 || return $?; return 56`, `{
  (test -e 23 || __shtx_return ${$__shtx_get_var(@( '?' ))})
  __shtx_return 56
}
`},
	"return2": {`fff() { return 12; return 34; }`, `{
  let src_0 = "fff() { return 12; return 34; }"
  $__shtx_func('fff', $src_0, (){
    let ctx = $__shtx_enter_func($0, $@)
    defer { $__shtx_exit_func($ctx); }
    try {
      {
        __shtx_return 12
        return 34
      }
    } catch e: _Return { return $e.status(); }
  })
}
`},
	"array_assign1": {`AAA=(aaa '123' "$(ls)")`, `{
  $__shtx_set_array_var('AAA', '=', @(aaa '123' "$(ls)"))
}
`},
	"array_assign2": {`de=()`, `{
  $__shtx_set_array_var('de', '=', @())
}
`},
	"array_assign3": {`de=([1]=111 222 [4]=444)`, `{
  (new _SparseArrayBuilder('de')).at(@( 1 111 )).add(@( 222 )).at(@( 4 444 )).build($false)
}
`},
	"array_append1": {`de+=(111 222)`, `{
  $__shtx_set_array_var('de', '+=', @(111 222))
}
`},
	"array_append2": {`de+=([1]=111 222 [4]=444)`, `{
  (new _SparseArrayBuilder('de')).at(@( 1 111 )).add(@( 222 )).at(@( 4 444 )).build($true)
}
`},
	"array_index1": {`de=(); echo "${de[0]}"`, `{
  $__shtx_set_array_var('de', '=', @())
  echo "${$__shtx_get_var_at(@( 'de' 0 ))}"
}
`},
	"array_index2": {`echo "${de[*]:-35243$OSTYPE}"`, `{
  echo "${$__shtx_get_var_at(@( 'de' '*' ':-' "35243"${$__shtx_get_var(@( 'OSTYPE' ))} ))}"
}
`},
	"array_expand1": {`de=(); echo "${de[@]}"`, `{
  $__shtx_set_array_var('de', '=', @())
  echo $__shtx_get_array_var('de')
}
`},
	"array_expand2": {`echo 23"4${array[@]}5$?"6`, `{
  echo $__shtx_concat(new [Any]()
    .add( @(23"4")[0] )
    .add($__shtx_get_array_var('array'))
    .add( @("5${$__shtx_get_var(@( '?' ))}"6)[0] )
  )
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
	"empty": {``, `function(argv: [String]): Int => {
  let old_argv = $__shtx_set_argv($argv)
  defer { $__shtx_set_argv($old_argv); }
  try {
  } catch e: _Return { return $e.status(); }
  return $?
}
`},
	"simple-command": {`echo hello`, `function(argv: [String]): Int => {
  let old_argv = $__shtx_set_argv($argv)
  defer { $__shtx_set_argv($old_argv); }
  try {
    echo hello
  } catch e: _Return { return $e.status(); }
  return $?
}
`},
	"return": {`return 34`, `function(argv: [String]): Int => {
  let old_argv = $__shtx_set_argv($argv)
  defer { $__shtx_set_argv($old_argv); }
  try {
    __shtx_return 34
  } catch e: _Return { return $e.status(); }
  return $?
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
