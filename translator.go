package main

import (
	"fmt"
	"io"
	"mvdan.cc/sh/v3/syntax"
	"strconv"
)

func todo(s string) bool {
	panic(fmt.Sprintf("[TODO] %s", s))
}

func fixmeCase(a any) {
	panic(fmt.Sprintf("[FIXME] unsupported switch-case type %T", a))
}

type TranslatorOptions uint

const (
	// None is no options (default)
	None TranslatorOptions = 0

	// NoTranslate does not perform translate (just echo input)
	NoTranslate = 1 << iota
)

type Translator struct {
	out         io.Writer // for output
	dump        io.Writer // for parsed ast dump
	option      TranslatorOptions
	indentLevel int
	funcLevel   int
}

func NewTranslator() *Translator {
	return &Translator{
		option: None,
	}
}

func (t *Translator) SetOption(options TranslatorOptions) {
	t.option |= options
}

func (t *Translator) SetDump(d io.Writer) {
	t.dump = d
}

func (t *Translator) Translate(in io.Reader, out io.Writer) error {
	// reset state
	t.out = out
	t.indentLevel = 0
	t.funcLevel = 0

	// parse
	f, e := syntax.NewParser().Parse(in, "")
	if e != nil {
		return fmt.Errorf("[error] %s", e.Error())
	}

	// dump
	if t.dump != nil {
		fmt.Fprintln(t.dump, "+++++  dump parsed ast  +++++")
		syntax.DebugPrint(t.dump, f)
		fmt.Fprintln(t.dump)
	}

	// translate
	if t.option&NoTranslate == NoTranslate {
		syntax.NewPrinter().Print(t.out, f)
	} else {
		t.visitFile(f)
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

func (t *Translator) visitFile(file *syntax.File) {
	t.emitLine("{")
	t.indentLevel++
	for _, stmt := range file.Stmts {
		t.visitStmt(stmt)
	}
	t.indentLevel--
	t.emitLine("}")
}

func (t *Translator) visitStmt(stmt *syntax.Stmt) {
	t.indent()
	t.visitCommand(stmt.Cmd, stmt.Redirs)
	_ = stmt.Negated && todo("support !")
	_ = stmt.Background && todo("support &")
	_ = stmt.Coprocess && todo("unsupported |&")
	t.newline()
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
	default:
		fixmeCase(n)
	}
	t.visitRedirects(redirs, cmdRedir)
}

func toRedirOpStr(op syntax.RedirOperator) string {
	switch op {
	case syntax.RdrOut, syntax.AppOut, syntax.RdrIn, syntax.DplOut,
		syntax.WordHdoc, syntax.RdrAll, syntax.AppAll:
		return op.String()
	default:
		todo("unsupported redir op: " + op.String())
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
		t.visitWordParts(redir.Word.Parts, false)
		_ = redir.Hdoc != nil && todo("support heredoc")
	}
}

func (t *Translator) visitAssigns(assigns []*syntax.Assign) {
	for _, assign := range assigns {
		_ = assign.Append && todo("support +=")
		_ = assign.Naked && todo("support Naked")
		_ = assign.Index != nil && todo("support indexed assign")
		_ = assign.Array != nil && todo("support array literal assign")
		t.emit(assign.Name.Value)
		t.emit("=")
		t.visitWordParts(assign.Value.Parts, false)
		t.emit(" ")
	}
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

func (t *Translator) visitCmdName(word *syntax.Word) {
	if isCmdLiteral(word) {
		t.emit(word.Parts[0].(*syntax.Lit).Value)
	} else { //FIXME: replace some builtin command with runtime helper functions
		t.emit("__shtx_dyna_call ")
		t.visitWordParts(word.Parts, false)
	}
}

func (t *Translator) visitCallExpr(expr *syntax.CallExpr) {
	if len(expr.Assigns) > 0 && len(expr.Args) == 0 {
		todo("support normal assignment")
	}

	t.visitAssigns(expr.Assigns)
	for i, arg := range expr.Args {
		if i == 0 {
			t.visitCmdName(arg)
		} else {
			t.emit(" ")
			t.visitWordParts(arg.Parts, false)
		}
	}
}

func (t *Translator) visitWordParts(parts []syntax.WordPart, dquoted bool) {
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
				t.indentLevel++
				for _, stmt := range n.Stmts {
					t.visitStmt(stmt)
				}
				t.indentLevel--
				t.indent()
				t.emit("})")
			}
		default:
			fixmeCase(n)
		}
	}
}
