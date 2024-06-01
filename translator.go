package main

import (
	"bytes"
	"fmt"
	"io"
	"mvdan.cc/sh/v3/syntax"
	"regexp"
	"runtime/debug"
	"strconv"
	"strings"
)

type ErrorType int

const (
	ErrorTodo ErrorType = iota
	ErrorFixme
)

type Error struct {
	pos syntax.Pos
	t   ErrorType
	msg string
}

func (e Error) Error() string {
	prefix := "[error]"
	switch e.t {
	case ErrorTodo:
		prefix = "[TODO]"
	case ErrorFixme:
		prefix = "[FIXME]"
	}
	return fmt.Sprintf("%s: %s %s", e.pos.String(), prefix, e.msg)
}

var _ error = Error{} // check error interface implementation

type ErrorCallback func(e error)

type WordPartOption struct {
	dQuoted    bool
	pattern    bool // for glob
	regex      bool // for =~
	singleWord bool // not perform glob/brace expansion, field splitting
}

type TranslationType int

const (
	TranslateNone TranslationType = iota
	TranslateEval
	TranslateSource
	TranslatePattern
)

type Offset struct {
	line uint
	col  uint
}

type Translator struct {
	in              []byte    // original input buffer
	out             io.Writer // for output
	dump            io.Writer // for parsed ast dump
	tranType        TranslationType
	offset          Offset
	indentLevel     int
	funcLevel       int
	caseExprCount   int
	errorCallback   ErrorCallback
	staticReturnMap map[*syntax.CallExpr]struct{}
}

func NewTranslator(tt TranslationType) *Translator {
	return &Translator{
		tranType: tt,
	}
}

func (t *Translator) SetDump(d io.Writer) {
	t.dump = d
}

func adjustPos(pos syntax.Pos, offset Offset) syntax.Pos {
	if offset.line == 0 && offset.col == 0 {
		return pos
	}
	line := pos.Line()
	col := pos.Col()
	if line == 1 {
		col += offset.col
	}
	line += offset.line
	return syntax.NewPos(pos.Offset(), line, col)
}

func (t *Translator) todo(pos syntax.Pos, s string) bool {
	e := Error{pos: adjustPos(pos, t.offset), t: ErrorTodo, msg: s}
	if t.errorCallback != nil {
		t.errorCallback(&e)
	}
	panic(e)
}

func (t *Translator) fixmeCase(pos syntax.Pos, a any) {
	e := Error{pos: adjustPos(pos, t.offset), t: ErrorFixme, msg: fmt.Sprintf("unsupported switch-case type %T", a)}
	if t.errorCallback != nil {
		t.errorCallback(&e)
	}
	panic(e)
}

func withLineNum(buf []byte) string {
	var sb = strings.Builder{}
	var ss = strings.Split(string(buf), "\n")
	var width = len(strconv.Itoa(len(ss)))
	for i, s := range ss {
		var line = fmt.Sprintf("%*d  %s\n", width, i+1, s)
		sb.WriteString(line)
	}
	return sb.String()
}

func (t *Translator) parse(buf []byte) (file *syntax.File, err error) {
	reader := bytes.NewReader(buf)
	file, err = syntax.NewParser().Parse(reader, "")
	if err != nil {
		if t.errorCallback != nil {
			t.errorCallback(err)
		}
		err = fmt.Errorf("+++++  error message  +++++\n%s\n\n"+
			"+++++  input script  +++++\n%s", err.Error(), withLineNum(buf))
	}
	return
}

func (t *Translator) Translate(buf []byte, out io.Writer) (err error) {
	// reset state
	t.in = buf
	t.out = out
	t.indentLevel = 0
	t.funcLevel = 0
	t.caseExprCount = 0
	t.staticReturnMap = make(map[*syntax.CallExpr]struct{})

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("+++++  error message  +++++\n%s\n\n"+
				"+++++  stack trace from panic  +++++\n%s\n"+
				"+++++  input script  +++++\n%s", r, debug.Stack(), withLineNum(buf))
		}
	}()

	switch t.tranType {
	case TranslateEval, TranslateSource, TranslateNone:
		// parse
		f, err := t.parse(buf)
		if err != nil {
			return err
		}

		// dump
		if t.dump != nil {
			_, _ = fmt.Fprintln(t.dump, "+++++  dump parsed ast  +++++")
			_ = syntax.DebugPrint(t.dump, f)
			_, _ = fmt.Fprintln(t.dump)
		}

		// translate
		switch t.tranType {
		case TranslateNone:
			_ = syntax.NewPrinter().Print(t.out, f)
		case TranslateEval:
			t.emitLine("{")
			t.visitStmts(f.Stmts)
			t.emitLine("}")
		case TranslateSource:
			t.emitLine("function(argv: [String]): Int => {")
			t.indentLevel++
			t.indent()
			t.emitLine("let old_argv = $__shtx_set_argv($argv)")
			t.indent()
			t.emitLine("defer { $__shtx_set_argv($old_argv); }")
			t.indent()
			t.emitLine("try {")
			t.visitStmts(f.Stmts)
			t.indent()
			t.emitLine("} catch e: _Return { return $e.status(); }")
			t.indent()
			t.emitLine("return $?")
			t.indentLevel--
			t.emitLine("}")
		case TranslatePattern:
		}
	case TranslatePattern:
		re := GlobToRegex(string(buf))
		t.emitLine(re)
	}
	return nil
}

func (t *Translator) emit(s string) {
	_, _ = fmt.Fprint(t.out, s)
}

func (t *Translator) emitLine(s string) {
	_, _ = fmt.Fprintln(t.out, s)
}

func (t *Translator) newline() {
	_, _ = fmt.Fprintln(t.out)
}

func (t *Translator) indent() {
	for i := 0; i < t.indentLevel; i++ {
		t.emit("  ")
	}
}

func (t *Translator) isToplevel() bool {
	return t.funcLevel == 0
}

func (t *Translator) visitStmts(stmts []*syntax.Stmt) {
	t.indentLevel++
	for _, stmt := range stmts {
		t.indent()
		t.visitStmt(stmt)
		t.newline()
	}
	t.indentLevel--
}

func (t *Translator) visitStmt(stmt *syntax.Stmt) {
	if stmt.Negated {
		switch n := stmt.Cmd.(type) {
		case *syntax.CallExpr, *syntax.DeclClause, *syntax.BinaryCmd, *syntax.TestClause:
			t.emit("! ")
		default:
			t.fixmeCase(n.Pos(), n)
		}
	}
	t.visitCommand(stmt.Cmd, stmt.Redirs)
	_ = stmt.Background && t.todo(stmt.Semicolon, "support &")
	_ = stmt.Coprocess && t.todo(stmt.Semicolon, "unsupported |&")
}

var declReplacement = map[string]string{
	"declare": "__shtx_declare",
	"export":  "__shtx_export",
	"local":   "__shtx_local",
}

func (t *Translator) visitCommand(cmd syntax.Command, redirs []*syntax.Redirect) {
	cmdRedir := false
	switch n := cmd.(type) {
	case nil:
		cmdRedir = true
		t.emit(":") //FIXME: '> /dev/null' or '< /dev/null' semantics
	case *syntax.CallExpr:
		cmdRedir = true
		t.visitCallExpr(n)
	case *syntax.DeclClause:
		v, ok := declReplacement[n.Variant.Value]
		if !ok {
			t.todo(n.Variant.Pos(), "unsupported decl: "+n.Variant.Value)
		}
		cmdRedir = true
		t.emit(v)
		t.emit(" ")
		t.visitAssigns(n.Args, true)
	case *syntax.BinaryCmd:
		_ = n.Op == syntax.PipeAll && t.todo(n.OpPos, "unsupported: |&")
		t.emit("(")
		t.visitStmt(n.X)
		t.emit(" " + n.Op.String() + " ")
		t.visitStmt(n.Y)
		t.emit(")")
	case *syntax.Block:
		t.emitLine("{")
		t.visitStmts(n.Stmts)
		t.indent()
		t.emit("}")
	case *syntax.IfClause:
		t.visitIfClause(n, false)
	case *syntax.CaseClause:
		t.visitCaseClause(n)
	case *syntax.FuncDecl:
		t.visitFuncDecl(n)
	case *syntax.TestClause:
		t.visitTestExpr(n.X)
	default:
		t.fixmeCase(n.Pos(), n)
	}
	t.visitRedirects(redirs, cmdRedir)
}

func (t *Translator) toRedirOpStr(redirect *syntax.Redirect) string {
	var op = redirect.Op
	switch op {
	case syntax.RdrInOut, syntax.Hdoc:
		t.todo(redirect.OpPos, "unsupported redir op: "+op.String())
	default:
		return op.String()
	}
	return ""
}

func (t *Translator) visitRedirects(redirs []*syntax.Redirect, cmd bool) {
	if len(redirs) > 0 && !cmd {
		t.emit(" with")
	}
	for _, redir := range redirs {
		t.emit(" ")
		if redir.N != nil {
			fd, e := strconv.Atoi(redir.N.Value)
			if e != nil {
				t.todo(redir.N.Pos(), "must be integer: "+redir.N.Value)
			}
			if fd != 1 && fd != 2 {
				t.todo(redir.N.Pos(), "only allow 1 or 2")
			}
			t.emit(strconv.Itoa(fd))
		}
		t.emit(t.toRedirOpStr(redir))
		t.emit(" ")
		t.visitWordParts(redir.Word.Parts)
		_ = redir.Hdoc != nil && t.todo(redir.OpPos, "support heredoc")
	}
}

func (t *Translator) visitAssigns(assigns []*syntax.Assign, shellAssign bool) {
	for i, assign := range assigns {
		_ = assign.Append && t.todo(assign.Pos(), "support +=")
		_ = (assign.Naked && !shellAssign) && t.todo(assign.Pos(), "support Naked")
		_ = assign.Index != nil && t.todo(assign.Index.Pos(), "support indexed assign")
		if shellAssign {
			_ = assign.Array != nil && t.todo(assign.Array.Pos(), "support array literal assign")
			if i > 0 {
				t.emit(" ")
			}
			if assign.Naked {
				if assign.Name != nil {
					t.emit(assign.Name.Value)
				} else if assign.Value != nil {
					t.visitWordPartsWith(assign.Value.Parts, WordPartOption{singleWord: true})
				}
			} else {
				t.emit(assign.Name.Value)
				t.emit("=")
				if assign.Value != nil {
					t.visitWordPartsWith(assign.Value.Parts, WordPartOption{singleWord: true})
				}
			}
		} else {
			if i > 0 {
				t.emit("; ")
			}
			if assign.Array != nil { // aaa=(a b c)
				t.emit("$__shtx_set_array_var(@( ")
				t.emit(assign.Name.Value)
				t.emit(" )[0], ")
				t.visitArrayExpr(assign.Array)
				t.emit(")")
			} else {
				t.emit("$__shtx_set_var(@( ")
				t.emit(assign.Name.Value)
				t.emit(" ")
				if assign.Value != nil {
					t.visitWordPartsWith(assign.Value.Parts, WordPartOption{singleWord: true})
				}
				t.emit(" ))")
			}
		}
	}
}

func (t *Translator) visitArrayExpr(array *syntax.ArrayExpr) {
	t.emit("@(")
	for i, elem := range array.Elems {
		if i > 0 {
			t.emit(" ")
		}
		if elem.Index != nil {
			t.visitArithmExpr(elem.Index)
		} else {
			t.emit("''")
		}
		t.emit(" ")
		t.visitWordParts(elem.Value.Parts)
	}
	t.emit(")")
}

func (t *Translator) visitArithmExpr(expr syntax.ArithmExpr) {
	v := toNumericConstant(expr)
	_ = v == "" && t.todo(expr.Pos(), "support non-const arithmetic expr")
	t.emit(v)
}

func (t *Translator) visitIfClause(clause *syntax.IfClause, elif bool) {
	if elif {
		t.emit(" elif ")
	} else {
		t.emit("if ")
	}

	// cond
	if len(clause.Cond) == 1 {
		t.emit("(")
		t.visitStmt(clause.Cond[0])
		t.emit(")")
	} else {
		t.emitLine("{")
		t.visitStmts(clause.Cond)
		t.indent()
		t.emit("}")
	}

	// then
	t.emitLine(" {")
	t.visitStmts(clause.Then)
	t.indent()
	t.emit("}")

	// else or elif
	if clause.Else != nil {
		if clause.Else.Cond != nil { // elif
			t.visitIfClause(clause.Else, true)
		} else { // else
			t.emitLine(" else {")
			t.visitStmts(clause.Else.Then)
			t.indent()
			t.emit("}")
		}
	}
}

func (t *Translator) visitCasePattern(pattern *syntax.Word, caseVarName string) {
	literal := pattern.Lit()
	if literal == "" || strings.HasPrefix(literal, "~") {
		t.emit("$__shtx_glob_match(@( $" + caseVarName)
		t.emit(" ")
		t.visitWordPartsWith(pattern.Parts, WordPartOption{pattern: true})
		t.emit(" ))")
	} else {
		t.emit("$" + caseVarName)
		t.emit(" =~ ")
		t.emit(LiteralGlobToRegex(literal))
	}
}

func (t *Translator) visitCaseClause(clause *syntax.CaseClause) {
	t.caseExprCount++
	var caseVarName = "case_" + strconv.Itoa(t.caseExprCount)

	t.emitLine("{")
	t.indentLevel++
	t.indent()
	t.emit("var " + caseVarName + "=''; " + caseVarName + "=")
	t.visitWordPartsWith(clause.Word.Parts, WordPartOption{singleWord: true})
	t.emitLine("")

	// case items
	for i, item := range clause.Items {
		_ = item.Op != syntax.Break && t.todo(item.OpPos, "not support "+item.Op.String())

		t.indent()
		if i == 0 {
			t.emit("if ")
		} else {
			t.emit("elif ")
		}
		for i2, pattern := range item.Patterns {
			if i2 > 0 {
				t.emit(" || ")
			}
			t.visitCasePattern(pattern, caseVarName)
		}
		t.emitLine(" {")
		t.visitStmts(item.Stmts)
		t.indent()
		t.emitLine("}")
	}

	t.indentLevel--
	t.indent()
	t.emit("}")
}

func (t *Translator) resolveStaticReturn(funcBody *syntax.Stmt) {
	switch n := funcBody.Cmd.(type) {
	case *syntax.CallExpr:
		if t.isStaticReturn(n.Args) {
			t.staticReturnMap[n] = struct{}{}
		}
	case *syntax.Block:
		if size := len(n.Stmts); size > 0 {
			t.resolveStaticReturn(n.Stmts[size-1])
		}
	}
}

func (t *Translator) visitFuncDecl(clause *syntax.FuncDecl) {
	t.funcLevel++
	t.emit("$__shtx_func('")
	t.emit(clause.Name.Value) // FIXME: escape command name
	t.emitLine("', (){")
	t.indentLevel++
	t.indent()
	t.emitLine("let ctx = $__shtx_enter_func($0, $@)")
	t.indent()
	t.emitLine("defer { $__shtx_exit_func($ctx); }")
	t.indent()
	t.emitLine("try {")
	t.indentLevel++
	t.indent()
	t.resolveStaticReturn(clause.Body)
	t.visitStmt(clause.Body)
	t.newline()
	t.indentLevel--
	t.indent()
	t.emitLine("} catch e: _Return { return $e.status(); }")
	t.indentLevel--
	t.indent()
	t.emit("})")
	t.funcLevel--
	t.staticReturnMap = make(map[*syntax.CallExpr]struct{}) // clear map
}

var ReCmdName = regexp.MustCompile(`^[_a-zA-Z][_a-zA-Z0-9-]*$`)

func toLiteralCmdName(word *syntax.Word) string {
	literal := word.Lit()
	unescaped := unescapeCmdName(literal)
	if strings.HasPrefix(unescaped, "__shtx_") || strings.HasPrefix(unescaped, "fake_") {
		return ""
	}
	if unescaped == "[" || unescaped == ":" || ReCmdName.MatchString(unescaped) {
		return literal
	}
	return ""
}

var cmdNameReplacement = map[string]string{
	"[":       "__shtx_[",
	"declare": "__shtx_declare",
	"export":  "__shtx_export",
	"local":   "__shtx_local",
	"unset":   "__shtx_unset",
	"shift":   "__shtx_shift",
	"read":    "__shtx_read",
	"printf":  "__shtx_printf",
	"return":  "__shtx_return",
	"trap":    "__shtx_trap",
	"eval":    "fake_eval",
	".":       "fake_source",
	"source":  "fake_source",
}

func remapCmdName(name string) string {
	unescaped := unescapeCmdName(name)
	if v, ok := cmdNameReplacement[unescaped]; ok {
		return v
	} else {
		keywords := []string{
			"alias", "assert", "defer", "else", "export-env", "exportenv", "import-env", "importenv",
			"interface", "new", "try", "throw", "typedef", "var"}
		for _, keyword := range keywords {
			if name == keyword {
				return "\\" + name
			}
		}
		return name // if not found replacement, return original value
	}
}

func (t *Translator) visitCmdName(word *syntax.Word) {
	if literal := toLiteralCmdName(word); len(literal) > 0 {
		name := remapCmdName(literal)
		t.emit(name)
	} else {
		t.emit("__shtx_dyna_call ")
		t.visitWordParts(word.Parts)
	}
}

func (t *Translator) isStaticReturn(args []*syntax.Word) bool {
	return len(args) == 2 && args[0].Lit() == "return" && !t.isToplevel()
}

func (t *Translator) visitCallExpr(expr *syntax.CallExpr) {
	envAssign := len(expr.Args) > 0
	t.visitAssigns(expr.Assigns, envAssign)
	if len(expr.Assigns) > 0 && envAssign {
		t.emit(" ")
	}

	if _, v := t.staticReturnMap[expr]; v {
		t.emit("return")
		if len(expr.Args) == 2 {
			t.emit(" ")
			word := expr.Args[1].Lit()
			if n, e := strconv.ParseInt(word, 10, 32); e == nil {
				t.emit(strconv.FormatInt(n, 10))
				if n < 0 || n > 255 {
					t.emit(" and 255")
				}
			} else {
				t.emit("{ var s='';s=")
				t.visitWordPartsWith(expr.Args[1].Parts, WordPartOption{singleWord: true})
				t.emit("; $__shtx_parse_status($s); }")
			}
		}
		return
	}
	for i, arg := range expr.Args {
		if i == 0 {
			t.visitCmdName(arg)
		} else {
			t.emit(" ")
			t.visitWordParts(arg.Parts)
		}
	}
}

var ReIdentifier = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
var RePositional = regexp.MustCompile(`^[0-9]+$`)

func isVarName(name string) bool {
	return ReIdentifier.MatchString(name)
}

func isSpecialParam(name string) bool {
	params := []string{"#", "?", "*", "@", "$"}
	for _, param := range params {
		if name == param {
			return true
		}
	}
	return false
}

func isValidParamName(name string) bool {
	return isVarName(name) || RePositional.MatchString(name) || isSpecialParam(name)
}

func (t *Translator) toExpansionOpStr(pos syntax.Pos, expansion *syntax.Expansion) string {
	var op = expansion.Op
	switch op {
	case syntax.AlternateUnset, syntax.AlternateUnsetOrNull, syntax.DefaultUnset, syntax.DefaultUnsetOrNull,
		syntax.ErrorUnset, syntax.ErrorUnsetOrNull, syntax.AssignUnset, syntax.AssignUnsetOrNull:
		return op.String()
	default:
		t.todo(pos, "unsupported expansion op: "+op.String())
	}
	return ""
}

func toConstant(expr syntax.ArithmExpr) string {
	switch n := expr.(type) {
	case *syntax.Word:
		return n.Lit()
	}
	return ""
}

func toNumericConstant(expr syntax.ArithmExpr) string {
	v := toConstant(expr)
	if v != "" {
		num, e := strconv.Atoi(v)
		if e == nil {
			return strconv.Itoa(num)
		}
	}
	return ""
}

func (t *Translator) visitWordPart(part syntax.WordPart, option WordPartOption) {
	switch n := part.(type) {
	case *syntax.Lit:
		if option.pattern {
			t.emit(quoteCmdArgAsGlobStr(n.Value))
		} else if option.regex {
			t.emit(quoteCmdArgAsRegexStr(n.Value))
		} else if option.singleWord {
			t.emit(quoteCmdArgAsLiteralStr(n.Value))
		} else {
			t.emit(n.Value)
		}
	case *syntax.SglQuoted:
		if option.pattern {
			t.emit("$__shtx_escape_glob_meta(")
		} else if option.regex {
			t.emit("$__shtx_escape_regex_meta(")
		}
		if n.Dollar {
			t.emit("$")
		}
		t.emit("'")
		t.emit(n.Value)
		t.emit("'")
		if option.pattern || option.regex {
			t.emit(")")
		}
	case *syntax.DblQuoted:
		_ = n.Dollar // always ignore prefix dollar even if Dollar is true
		if option.pattern {
			t.emit("$__shtx_escape_glob_meta(")
		} else if option.regex {
			t.emit("$__shtx_escape_regex_meta(")
		}
		t.emit("\"")
		for _, wordPart := range n.Parts {
			t.visitWordPart(wordPart, WordPartOption{dQuoted: true})
		}
		t.emit("\"")
		if option.pattern || option.regex {
			t.emit(")")
		}
	case *syntax.ParamExp:
		if n.Param.Value != "?" && n.Param.Value != "#" && n.Param.Value != "$" && !option.dQuoted && !option.pattern && !option.regex && !option.singleWord {
			t.todo(n.Pos(), "support unquoted parameter expansion")
		}
		_ = n.Excl && t.todo(n.Pos(), "not support ${!a}")
		_ = n.Length && t.todo(n.Pos(), "support ${#a}")
		_ = n.Width && t.todo(n.Pos(), "not support ${%a}")
		_ = n.Slice != nil && t.todo(n.Pos(), "not support ${a:x:y}")
		_ = n.Repl != nil && t.todo(n.Pos(), "not support ${a/x/y}")
		_ = n.Names != 0 && t.todo(n.Pos(), "not support ${!prefix*}")
		_ = !isValidParamName(n.Param.Value) && t.todo(n.Param.Pos(), "unsupported param name: "+n.Param.Value)
		t.emit("${$__shtx_get_var")
		if n.Index != nil {
			t.emit("_at")
		}
		t.emit("(@( '")
		t.emit(n.Param.Value)
		t.emit("'")
		if n.Index != nil {
			t.emit(" ")
			if v := toConstant(n.Index); v == "*" {
				t.emit("'*'")
			} else {
				t.visitArithmExpr(n.Index)
			}
		}
		if n.Exp != nil {
			t.emit(" '")
			t.emit(t.toExpansionOpStr(n.Pos(), n.Exp))
			t.emit("' ")
			if n.Exp.Word != nil {
				t.visitWordParts(n.Exp.Word.Parts)
			}
		}
		t.emit(" ))}")
	case *syntax.CmdSubst:
		_ = n.TempFile && t.todo(n.Pos(), "not support ${")
		_ = n.ReplyVar && t.todo(n.Pos(), "not support ${|")
		if len(n.Stmts) == 0 {
			// skip empty command substitution, $(), ``, `# this is a comment`
			return
		}

		stmts := n.Stmts

		_ = !option.dQuoted && !option.pattern && !option.regex && !option.singleWord && t.todo(n.Pos(), "support unquoted command substitution")
		if option.dQuoted && n.Backquotes { // unescape and re-parse
			tmpBuf := t.in[n.Pos().Offset()+1 : n.End().Offset()-1] // remove prefix and suffix back-quote
			t.offset = Offset{                                      // adjust line num offset for better error message
				line: n.Pos().Line() - 1,
				col:  n.Pos().Col(),
			}
			f, e := t.parse([]byte(unescapeDoubleQuoted(string(tmpBuf), false)))
			defer func() { t.offset = Offset{0, 0} }()
			if e != nil {
				panic(e) // force return
			}
			stmts = f.Stmts
		}

		if option.pattern || option.regex || option.singleWord {
			t.emit("\"")
		}
		if len(stmts) == 1 {
			t.emit("$(")
			t.visitCommand(stmts[0].Cmd, stmts[0].Redirs)
			t.emit(")")
		} else {
			t.emitLine("$({")
			t.visitStmts(stmts)
			t.indent()
			t.emit("})")
		}
		if option.pattern || option.regex || option.singleWord {
			t.emit("\"")
		}
	default:
		t.fixmeCase(n.Pos(), n)
	}
}

func resolveArrayExpandParamName(part syntax.WordPart) string {
	switch n := part.(type) {
	case *syntax.ParamExp:
		if n.Param.Value == "@" {
			return "@"
		}
		if n.Index != nil {
			switch e := n.Index.(type) {
			case *syntax.Word:
				if e.Lit() == "@" {
					return n.Param.Value
				}
			}
		}
	}
	return ""
}

func isArrayExpandDblQuoted(quoted *syntax.DblQuoted) bool {
	for _, part := range quoted.Parts {
		if name := resolveArrayExpandParamName(part); name != "" {
			return true
		}
	}
	return false
}

func isArrayExpandWordParts(parts []syntax.WordPart) bool {
	for _, part := range parts {
		switch n := part.(type) {
		case *syntax.DblQuoted:
			if isArrayExpandDblQuoted(n) {
				return true
			}
		}
	}
	return false
}

func resolveSimpleArrayExpand(parts []syntax.WordPart) string {
	if len(parts) == 1 {
		switch n := parts[0].(type) {
		case *syntax.DblQuoted:
			if len(n.Parts) != 1 {
				return ""
			}
			return resolveArrayExpandParamName(n.Parts[0])
		}
	}
	return ""
}

func (t *Translator) expandDblQuoted(quoted *syntax.DblQuoted) {
	t.emit("\"")
	for _, part := range quoted.Parts {
		if name := resolveArrayExpandParamName(part); name != "" {
			t.emitLine("\")[0] )")
			t.indent()
			t.emit(".add($__shtx_get_array_var('")
			t.emit(name)
			t.emitLine("'))")
			t.indent()
			t.emit(".add( @(\"")
			continue
		}
		option := WordPartOption{}
		option.dQuoted = true
		t.visitWordPart(part, option)
	}
	t.emit("\"")
}

func (t *Translator) visitWordPartsWith(parts []syntax.WordPart, option WordPartOption) {
	if !isArrayExpandWordParts(parts) || option.singleWord {
		for _, part := range parts {
			t.visitWordPart(part, option)
		}
		return
	}

	_ = (option.pattern || option.regex) && t.todo(parts[0].Pos(), "pattern with array expand is not supported")

	// for `$@` or `${array[@]}`
	if name := resolveSimpleArrayExpand(parts); name != "" {
		t.emit("$__shtx_get_array_var('")
		t.emit(name)
		t.emit("')")
		return
	}

	t.emitLine("$__shtx_concat(new [Any]()")
	t.indentLevel++
	t.indent()
	t.emit(".add( @(")
	for _, part := range parts {
		switch n := part.(type) {
		case *syntax.DblQuoted:
			if isArrayExpandDblQuoted(n) {
				t.expandDblQuoted(n)
				continue
			}
		}
		t.visitWordPart(part, option)
	}
	t.emitLine(")[0] )")
	t.indentLevel--
	t.indent()
	t.emit(")")
}

func (t *Translator) visitWordParts(parts []syntax.WordPart) {
	t.visitWordPartsWith(parts, WordPartOption{})
}

func (t *Translator) visitTestExpr(expr syntax.TestExpr) {
	t.emit("(")
	switch n := expr.(type) {
	case *syntax.BinaryTest:
		switch n.Op {
		case syntax.AndTest:
			t.visitTestExpr(n.X)
			t.emit(" && ")
			t.visitTestExpr(n.Y)
		case syntax.OrTest:
			t.visitTestExpr(n.X)
			t.emit(" || ")
			t.visitTestExpr(n.Y)
		case syntax.TsReMatch:
			t.emit("$__shtx_regex_match(@( ")
			switch left := n.X.(type) {
			case *syntax.Word:
				t.visitWordPartsWith(left.Parts, WordPartOption{singleWord: true})
			default:
				t.fixmeCase(left.Pos(), left)
			}
			t.emit(" ")
			switch right := n.Y.(type) {
			case *syntax.Word:
				t.visitWordPartsWith(right.Parts, WordPartOption{regex: true})
			default:
				t.fixmeCase(right.Pos(), right)
			}
			t.emit(" ))")
		case syntax.TsMatch, syntax.TsMatchShort, syntax.TsNoMatch:
			if n.Op == syntax.TsNoMatch {
				t.emit("!")
			}
			t.emit("$__shtx_glob_match(@( ")
			switch left := n.X.(type) {
			case *syntax.Word:
				t.visitWordPartsWith(left.Parts, WordPartOption{singleWord: true})
			default:
				t.fixmeCase(left.Pos(), left)
			}
			t.emit(" ")
			switch right := n.Y.(type) {
			case *syntax.Word:
				t.visitWordPartsWith(right.Parts, WordPartOption{pattern: true})
			default:
				t.fixmeCase(right.Pos(), right)
			}
			t.emit(" ))")
		default:
			t.emit("test ")
			switch left := n.X.(type) {
			case *syntax.Word:
				t.visitWordPartsWith(left.Parts, WordPartOption{singleWord: true})
			default:
				t.fixmeCase(left.Pos(), left)
			}
			t.emit(" \"")
			t.emit(n.Op.String())
			t.emit("\" ")
			switch right := n.Y.(type) {
			case *syntax.Word:
				t.visitWordPartsWith(right.Parts, WordPartOption{singleWord: true})
			default:
				t.fixmeCase(right.Pos(), right)
			}
		}
	case *syntax.UnaryTest:
		if n.Op == syntax.TsNot {
			t.emit("!")
			t.visitTestExpr(n.X)
		} else { // use test command
			t.emit("test ")
			t.emit(n.Op.String())
			t.emit(" ")
			switch n2 := n.X.(type) {
			case *syntax.Word:
				t.visitWordPartsWith(n2.Parts, WordPartOption{singleWord: true})
			default:
				t.fixmeCase(n2.Pos(), n2)
			}
		}
	case *syntax.ParenTest:
		t.visitTestExpr(n.X)
	case *syntax.Word:
		t.emit("test ")
		t.visitWordPartsWith(n.Parts, WordPartOption{singleWord: true})
	}
	t.emit(")")
}
