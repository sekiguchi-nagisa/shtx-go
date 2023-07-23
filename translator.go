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

func todo(s string) bool {
	panic(fmt.Sprintf("[TODO] %s", s))
}

func fixmeCase(a any) {
	panic(fmt.Sprintf("[FIXME] unsupported switch-case type %T", a))
}

type TranslationType int

const (
	TranslateNone TranslationType = iota
	TranslateEval
	TranslateSource
)

type Translator struct {
	out         io.Writer // for output
	dump        io.Writer // for parsed ast dump
	tranType    TranslationType
	indentLevel int
	funcLevel   int
}

func NewTranslator(tt TranslationType) *Translator {
	return &Translator{
		tranType: tt,
	}
}

func (t *Translator) SetDump(d io.Writer) {
	t.dump = d
}

func (t *Translator) Translate(buf []byte, out io.Writer) (err error) {
	// reset state
	t.out = out
	t.indentLevel = 0
	t.funcLevel = 0

	// parse
	reader := bytes.NewReader(buf)
	f, e := syntax.NewParser().Parse(reader, "")
	if e != nil {
		return fmt.Errorf("+++++  error message  +++++\n%s\n\n"+
			"+++++  input script  +++++\n%s", e.Error(), buf)
	}

	// dump
	if t.dump != nil {
		fmt.Fprintln(t.dump, "+++++  dump parsed ast  +++++")
		syntax.DebugPrint(t.dump, f)
		fmt.Fprintln(t.dump)
	}

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("+++++  error message  +++++\n%s\n\n"+
				"+++++  stack trace from panic  +++++\n%s\n"+
				"+++++  input script  +++++\n%s", r, debug.Stack(), buf)
		}
	}()

	// translate
	switch t.tranType {
	case TranslateNone:
		syntax.NewPrinter().Print(t.out, f)
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
	}
	return nil
}

func (t *Translator) emit(s string) {
	fmt.Fprint(t.out, s)
}

func (t *Translator) emitLine(s string) {
	fmt.Fprintln(t.out, s)
}

func (t *Translator) newline() {
	fmt.Fprintln(t.out)
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
	t.visitCommand(stmt.Cmd, stmt.Redirs)
	_ = stmt.Negated && todo("support !")
	_ = stmt.Background && todo("support &")
	_ = stmt.Coprocess && todo("unsupported |&")
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
			todo("unsupported decl: " + n.Variant.Value)
		}
		t.emit(v)
		t.emit(" ")
		t.visitAssigns(n.Args, true)
	case *syntax.BinaryCmd:
		_ = n.Op == syntax.PipeAll && todo("unsupported: |&")
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
	case *syntax.FuncDecl:
		t.visitFuncDecl(n)
	default:
		fixmeCase(n)
	}
	t.visitRedirects(redirs, cmdRedir)
}

func toRedirOpStr(op syntax.RedirOperator) string {
	switch op {
	case syntax.RdrInOut, syntax.Hdoc:
		todo("unsupported redir op: " + op.String())
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
				todo("must be integer: " + redir.N.Value)
			}
			if fd != 1 && fd != 2 {
				todo("only allow 1 or 2")
			}
			t.emit(strconv.Itoa(fd))
		}
		t.emit(toRedirOpStr(redir.Op))
		t.emit(" ")
		t.visitWordParts(redir.Word.Parts, false)
		_ = redir.Hdoc != nil && todo("support heredoc")
	}
}

func (t *Translator) visitAssigns(assigns []*syntax.Assign, shellAssign bool) {
	for i, assign := range assigns {
		_ = assign.Append && todo("support +=")
		_ = (assign.Naked && !shellAssign) && todo("support Naked")
		_ = assign.Index != nil && todo("support indexed assign")
		_ = assign.Array != nil && todo("support array literal assign")
		if shellAssign {
			if i > 0 {
				t.emit(" ")
			}
			t.emit(assign.Name.Value)
			if !assign.Naked {
				t.emit("=")
				if assign.Value != nil {
					t.visitWordParts(assign.Value.Parts, false)
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
				t.visitWordParts(assign.Value.Parts, false)
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

func (t *Translator) visitFuncDecl(clause *syntax.FuncDecl) {
	t.emit("$__shtx_func('")
	t.emit(clause.Name.Value) // FIXME: escape command name
	t.emitLine("', (){")
	t.indentLevel++
	t.indent()
	t.emitLine("$__shtx_enter_func($0, $@)")
	t.indent()
	t.emitLine("let old = $CMD_FALLBACK")
	t.indent()
	t.emitLine("$CMD_FALLBACK = $__shtx_cmd_fallback")
	t.indent()
	t.emitLine("defer { $__shtx_exit_func(); $CMD_FALLBACK = $old; }")
	t.indent()
	t.visitStmt(clause.Body)
	t.indentLevel--
	t.newline()
	t.indent()
	t.emit("})")
}

func isCmdLiteral(word *syntax.Word) bool {
	if len(word.Parts) != 1 {
		return false
	}
	switch n := word.Parts[0].(type) {
	case *syntax.Lit:
		if len(n.Value) > 0 {

		}
		return true //FIXME: check literal format
	default:
		return false
	}
}

var cmdNameReplacement = map[string]string{
	"[":      "__shtx_[",
	"export": "__shtx_export",
	"local":  "__shtx_local",
	"unset":  "__shtx_unset",
	"shift":  "__shtx_shift",
	"eval":   "fake_eval",
	".":      "fake_source",
	"source": "fake_source",
}

func remapCmdName(name string) string {
	builder := strings.Builder{}
	builder.Grow(len(name))
	r := []rune(name)
	for i := 0; i < len(r); i++ {
		c := r[i]
		if c == '\\' {
			i++
			next := r[i]
			switch next {
			case '\n', '\r':
				continue
			default:
				c = next
			}
		}
		builder.WriteRune(c)
	}
	unescaped := builder.String()
	v, ok := cmdNameReplacement[unescaped]
	if ok {
		return v
	} else {
		return name // if not found replacement, return original value
	}
}

func (t *Translator) visitCmdName(word *syntax.Word) {
	if isCmdLiteral(word) {
		name := remapCmdName(word.Parts[0].(*syntax.Lit).Value)
		t.emit(name)
	} else {
		t.emit("__shtx_dyna_call ")
		t.visitWordParts(word.Parts, false)
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
			t.visitWordParts(arg.Parts, false)
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

func toExpansionOpStr(op syntax.ParExpOperator) string {
	switch op {
	case syntax.AlternateUnset, syntax.AlternateUnsetOrNull, syntax.DefaultUnset, syntax.DefaultUnsetOrNull,
		syntax.ErrorUnset, syntax.ErrorUnsetOrNull, syntax.AssignUnset, syntax.AssignUnsetOrNull:
		return op.String()
	default:
		todo("unsupported expansion op: " + op.String())
	}
	return ""
}

func (t *Translator) visitWordParts(parts []syntax.WordPart, dQuoted bool) {
	for _, part := range parts {
		switch n := part.(type) {
		case *syntax.Lit:
			t.emit(n.Value)
		case *syntax.SglQuoted:
			if n.Dollar {
				t.emit("$")
			}
			t.emit("'")
			t.emit(n.Value)
			t.emit("'")
		case *syntax.DblQuoted:
			// always ignore prefix dollar
			//FIXME: warning if Dollar is true ?
			t.emit("\"")
			t.visitWordParts(n.Parts, true)
			t.emit("\"")
		case *syntax.ParamExp:
			if n.Param.Value != "?" && n.Param.Value != "#" && !dQuoted {
				todo("support unquoted parameter expansion")
			}
			_ = n.Excl && todo("not support ${!a}")
			_ = n.Length && todo("support ${#a}")
			_ = n.Width && todo("not support ${%a}")
			_ = n.Index != nil && todo("support ${a[i]}")
			_ = n.Slice != nil && todo("not support ${a:x:y}")
			_ = n.Repl != nil && todo("not support ${a/x/y}")
			_ = n.Names != 0 && todo("not support ${!prefix*}")
			_ = !isValidParamName(n.Param.Value) && todo("unsupported param name: "+n.Param.Value)
			t.emit("${{__shtx_var_get $? '")
			t.emit(n.Param.Value)
			t.emit("'")
			if n.Exp != nil {
				t.emit(" '")
				t.emit(toExpansionOpStr(n.Exp.Op))
				t.emit("' ")
				if n.Exp.Word != nil {
					t.visitWordParts(n.Exp.Word.Parts, false)
				}
			}
			t.emit("; $REPLY; }}")
		case *syntax.CmdSubst:
			_ = n.TempFile && todo("not support ${")
			_ = n.ReplyVar && todo("not support ${|")
			if len(n.Stmts) == 0 {
				// skip empty command substitution, $(), ``, `# this is a comment`
				continue
			} else if len(n.Stmts) == 1 {
				t.emit("$(")
				t.visitCommand(n.Stmts[0].Cmd, n.Stmts[0].Redirs)
				t.emit(")")
			} else {
				t.emitLine("$({")
				t.visitStmts(n.Stmts)
				t.indent()
				t.emit("})")
			}
		default:
			fixmeCase(n)
		}
	}
}
