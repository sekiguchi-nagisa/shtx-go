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

func todo(pos syntax.Pos, s string) bool {
	panic(fmt.Sprintf("%s: [TODO] %s", pos.String(), s))
}

func fixmeCase(pos syntax.Pos, a any) {
	panic(fmt.Sprintf("%s: [FIXME] unsupported switch-case type %T", pos.String(), a))
}

type WordPartOption struct {
	dQuoted bool
	pattern bool
}

type TranslationType int

const (
	TranslateNone TranslationType = iota
	TranslateEval
	TranslateSource
	TranslatePattern
)

type Translator struct {
	out           io.Writer // for output
	dump          io.Writer // for parsed ast dump
	tranType      TranslationType
	indentLevel   int
	funcLevel     int
	caseExprCount int
}

func NewTranslator(tt TranslationType) *Translator {
	return &Translator{
		tranType: tt,
	}
}

func (t *Translator) SetDump(d io.Writer) {
	t.dump = d
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

func (t *Translator) Translate(buf []byte, out io.Writer) (err error) {
	// reset state
	t.out = out
	t.indentLevel = 0
	t.funcLevel = 0
	t.caseExprCount = 0

	// parse
	var f *syntax.File
	switch t.tranType {
	case TranslateEval, TranslateSource:
		var e error
		reader := bytes.NewReader(buf)
		f, e = syntax.NewParser().Parse(reader, "")
		if e != nil {
			return fmt.Errorf("+++++  error message  +++++\n%s\n\n"+
				"+++++  input script  +++++\n%s", e.Error(), withLineNum(buf))
		}

		// dump
		if t.dump != nil {
			_, _ = fmt.Fprintln(t.dump, "+++++  dump parsed ast  +++++")
			_ = syntax.DebugPrint(t.dump, f)
			_, _ = fmt.Fprintln(t.dump)
		}
	case TranslateNone:
	case TranslatePattern: // do nothing
	}

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("+++++  error message  +++++\n%s\n\n"+
				"+++++  stack trace from panic  +++++\n%s\n"+
				"+++++  input script  +++++\n%s", r, debug.Stack(), withLineNum(buf))
		}
	}()

	// translate
	switch t.tranType {
	case TranslateNone:
		_ = syntax.NewPrinter().Print(t.out, f)
	case TranslateEval:
		t.emitLine("{")
		t.visitStmts(f.Stmts)
		t.emitLine("}")
	case TranslateSource:
		t.emitLine("function(argv : [String]) => {")
		t.indentLevel++
		t.indent()
		t.emitLine("let old_argv = $__shtx_set_argv($argv)")
		t.indent()
		t.emitLine("defer { $__shtx_set_argv($old_argv); }")
		t.indentLevel--
		t.visitStmts(f.Stmts)
		t.emitLine("}")
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
		if _, ok := stmt.Cmd.(*syntax.CallExpr); ok {
			t.emit("! ")
		} else {
			todo(stmt.Pos(), "support !")
		}
	}
	t.visitCommand(stmt.Cmd, stmt.Redirs)
	_ = stmt.Background && todo(stmt.Semicolon, "support &")
	_ = stmt.Coprocess && todo(stmt.Semicolon, "unsupported |&")
}

var declReplacement = map[string]string{
	"export": "__shtx_export",
	"local":  "__shtx_local",
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
			todo(n.Variant.Pos(), "unsupported decl: "+n.Variant.Value)
		}
		t.emit(v)
		t.emit(" ")
		t.visitAssigns(n.Args, true)
	case *syntax.BinaryCmd:
		_ = n.Op == syntax.PipeAll && todo(n.OpPos, "unsupported: |&")
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
	default:
		fixmeCase(n.Pos(), n)
	}
	t.visitRedirects(redirs, cmdRedir)
}

func toRedirOpStr(redirect *syntax.Redirect) string {
	var op = redirect.Op
	switch op {
	case syntax.RdrInOut, syntax.Hdoc:
		todo(redirect.OpPos, "unsupported redir op: "+op.String())
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
				todo(redir.N.Pos(), "must be integer: "+redir.N.Value)
			}
			if fd != 1 && fd != 2 {
				todo(redir.N.Pos(), "only allow 1 or 2")
			}
			t.emit(strconv.Itoa(fd))
		}
		t.emit(toRedirOpStr(redir))
		t.emit(" ")
		t.visitWordParts(redir.Word.Parts)
		_ = redir.Hdoc != nil && todo(redir.OpPos, "support heredoc")
	}
}

func (t *Translator) visitAssigns(assigns []*syntax.Assign, shellAssign bool) {
	for i, assign := range assigns {
		_ = assign.Append && todo(assign.Pos(), "support +=")
		_ = (assign.Naked && !shellAssign) && todo(assign.Pos(), "support Naked")
		_ = assign.Index != nil && todo(assign.Index.Pos(), "support indexed assign")
		_ = assign.Array != nil && todo(assign.Array.Pos(), "support array literal assign")
		if shellAssign {
			if i > 0 {
				t.emit(" ")
			}
			t.emit(assign.Name.Value)
			if !assign.Naked {
				t.emit("=")
				if assign.Value != nil {
					t.visitWordParts(assign.Value.Parts)
				}
			}
		} else {
			if i > 0 {
				t.emit("; ")
			}
			t.emit("__shtx_var_set ")
			t.emit(assign.Name.Value)
			t.emit(" ")
			if assign.Value != nil {
				t.visitWordParts(assign.Value.Parts)
			}
		}
	}
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
	t.emit("let " + caseVarName + " = @(")
	t.visitWordParts(clause.Word.Parts)
	t.emitLine(")[0]")

	// case items
	for i, item := range clause.Items {
		_ = item.Op != syntax.Break && todo(item.OpPos, "not support "+item.Op.String())

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

func (t *Translator) visitFuncDecl(clause *syntax.FuncDecl) {
	t.emit("$__shtx_func('")
	t.emit(clause.Name.Value) // FIXME: escape command name
	t.emitLine("', (){")
	t.indentLevel++
	t.indent()
	t.emitLine("let ctx = $__shtx_enter_func($0, $@)")
	t.indent()
	t.emitLine("defer { $__shtx_exit_func($ctx); }")
	t.indent()
	t.visitStmt(clause.Body)
	t.indentLevel--
	t.newline()
	t.indent()
	t.emit("})")
}

func toLiteralCmdName(word *syntax.Word) string {
	literal := word.Lit()
	unescaped := unescapeCmdName(literal)
	if strings.HasPrefix(unescaped, "__shtx_") || strings.HasPrefix(unescaped, "fake_") {
		return ""
	}
	return literal //FIXME: check literal format
}

var cmdNameReplacement = map[string]string{
	"[":      "__shtx_[",
	"export": "__shtx_export",
	"local":  "__shtx_local",
	"unset":  "__shtx_unset",
	"shift":  "__shtx_shift",
	"read":   "__shtx_read",
	"printf": "__shtx_printf",
	"eval":   "fake_eval",
	".":      "fake_source",
	"source": "fake_source",
}

func remapCmdName(name string) string {
	unescaped := unescapeCmdName(name)
	if v, ok := cmdNameReplacement[unescaped]; ok {
		return v
	} else {
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

func (t *Translator) visitCallExpr(expr *syntax.CallExpr) {
	envAssign := len(expr.Args) > 0
	t.visitAssigns(expr.Assigns, envAssign)
	if len(expr.Assigns) > 0 && envAssign {
		t.emit(" ")
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
var RePositional = regexp.MustCompile(`^[0-9]*$`)

func isVarName(name string) bool {
	return ReIdentifier.MatchString(name)
}

func isValidParamName(name string) bool {
	return isVarName(name) || RePositional.MatchString(name) || name == "#" || name == "?" || name == "*"
}

func toExpansionOpStr(pos syntax.Pos, expansion *syntax.Expansion) string {
	var op = expansion.Op
	switch op {
	case syntax.AlternateUnset, syntax.AlternateUnsetOrNull, syntax.DefaultUnset, syntax.DefaultUnsetOrNull,
		syntax.ErrorUnset, syntax.ErrorUnsetOrNull, syntax.AssignUnset, syntax.AssignUnsetOrNull:
		return op.String()
	default:
		todo(pos, "unsupported expansion op: "+op.String())
	}
	return ""
}

func (t *Translator) visitWordPart(part syntax.WordPart, option WordPartOption) {
	switch n := part.(type) {
	case *syntax.Lit:
		if option.pattern {
			t.emit(quoteCmdArgAsGlobStr(n.Value))
		} else {
			t.emit(n.Value)
		}
	case *syntax.SglQuoted:
		if option.pattern {
			t.emit("$__shtx_escape_glob_meta(")
		}
		if n.Dollar {
			t.emit("$")
		}
		t.emit("'")
		t.emit(n.Value)
		t.emit("'")
		if option.pattern {
			t.emit(")")
		}
	case *syntax.DblQuoted:
		_ = n.Dollar // always ignore prefix dollar even if Dollar is true
		if option.pattern {
			t.emit("$__shtx_escape_glob_meta(")
		}
		t.emit("\"")
		for _, wordPart := range n.Parts {
			t.visitWordPart(wordPart, WordPartOption{dQuoted: true})
		}
		t.emit("\"")
		if option.pattern {
			t.emit(")")
		}
	case *syntax.ParamExp:
		if n.Param.Value != "?" && n.Param.Value != "#" && !option.dQuoted && !option.pattern {
			todo(n.Pos(), "support unquoted parameter expansion")
		}
		_ = n.Excl && todo(n.Pos(), "not support ${!a}")
		_ = n.Length && todo(n.Pos(), "support ${#a}")
		_ = n.Width && todo(n.Pos(), "not support ${%a}")
		_ = n.Index != nil && todo(n.Index.Pos(), "support ${a[i]}")
		_ = n.Slice != nil && todo(n.Pos(), "not support ${a:x:y}")
		_ = n.Repl != nil && todo(n.Pos(), "not support ${a/x/y}")
		_ = n.Names != 0 && todo(n.Pos(), "not support ${!prefix*}")
		_ = !isValidParamName(n.Param.Value) && todo(n.Param.Pos(), "unsupported param name: "+n.Param.Value)
		t.emit("${{__shtx_var_get $? '")
		t.emit(n.Param.Value)
		t.emit("'")
		if n.Exp != nil {
			t.emit(" '")
			t.emit(toExpansionOpStr(n.Pos(), n.Exp))
			t.emit("' ")
			if n.Exp.Word != nil {
				t.visitWordParts(n.Exp.Word.Parts)
			}
		}
		t.emit("; $REPLY; }}")
	case *syntax.CmdSubst:
		_ = n.TempFile && todo(n.Pos(), "not support ${")
		_ = n.ReplyVar && todo(n.Pos(), "not support ${|")
		if len(n.Stmts) == 0 {
			// skip empty command substitution, $(), ``, `# this is a comment`
			return
		}

		_ = !option.dQuoted && !option.pattern && todo(n.Pos(), "support unquoted command substitution")
		if option.pattern {
			t.emit("\"")
		}
		if len(n.Stmts) == 1 {
			t.emit("$(")
			t.visitCommand(n.Stmts[0].Cmd, n.Stmts[0].Redirs)
			t.emit(")")
		} else {
			t.emitLine("$({")
			t.visitStmts(n.Stmts)
			t.indent()
			t.emit("})")
		}
		if option.pattern {
			t.emit("\"")
		}
	default:
		fixmeCase(n.Pos(), n)
	}
}

func isArrayExpandDblQuoted(quoted *syntax.DblQuoted) bool {
	for _, part := range quoted.Parts {
		switch n := part.(type) {
		case *syntax.ParamExp:
			if n.Param.Value == "@" {
				return true
			}
		}
	}
	return false
}

func isArrayExpand(parts []syntax.WordPart) bool {
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

func isSimpleArgsExpand(parts []syntax.WordPart) bool {
	if len(parts) == 1 {
		switch n := parts[0].(type) {
		case *syntax.DblQuoted:
			if len(n.Parts) != 1 {
				return false
			}
			switch nn := n.Parts[0].(type) {
			case *syntax.ParamExp:
				if nn.Param.Value == "@" {
					return true
				}
			}
		}
	}
	return false
}

func (t *Translator) expandDblQuoted(quoted *syntax.DblQuoted) {
	t.emit("\"")
	for _, part := range quoted.Parts {
		switch n := part.(type) {
		case *syntax.ParamExp:
			if n.Param.Value == "@" {
				t.emitLine("\")[0] )")
				t.indent()
				t.emitLine(".add($__shtx_get_args())")
				t.indent()
				t.emit(".add( @(\"")
				continue
			}
		}
		option := WordPartOption{}
		option.dQuoted = true
		t.visitWordPart(part, option)
	}
	t.emit("\"")
}

func (t *Translator) visitWordPartsWith(parts []syntax.WordPart, option WordPartOption) {
	if !isArrayExpand(parts) {
		for _, part := range parts {
			t.visitWordPart(part, option)
		}
		return
	}

	_ = option.pattern && todo(parts[0].Pos(), "pattern with array expand is not supported")

	// for `$@` or `$array[@]`
	if isSimpleArgsExpand(parts) { // "$@"
		t.emit("$__shtx_get_args()")
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
