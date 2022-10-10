package main

import (
	"fmt"
	"io"
	"mvdan.cc/sh/v3/syntax"
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
	out    io.Writer // for output
	dump   io.Writer // for parsed ast dump
	option TranslatorOptions
	level  int
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
	// reset io
	t.out = out
	t.level = 0

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
	for i := 0; i < t.level; i++ {
		t.emit("  ")
	}
}

func (t *Translator) visitFile(file *syntax.File) {
	t.emitLine("function(args : [String]) => {")
	t.level++
	for _, stmt := range file.Stmts {
		t.visitStmt(stmt)
	}
	t.level--
	t.emitLine("}")
}

func (t *Translator) visitStmt(stmt *syntax.Stmt) {
	t.indent()
	t.visitCommand(stmt.Cmd, stmt.Redirs)
	t.newline()
}

func (t *Translator) visitCommand(cmd syntax.Command, redirs []*syntax.Redirect) {

	switch n := cmd.(type) {
	case nil:
		t.emit(":") //FIXME: '> /dev/null' or '< /dev/null' semantics
	case *syntax.CallExpr:
		t.visitCallExpr(n)
	default:
		fixmeCase(n)
	}
	if redirs != nil {
		todo("support redirection")
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
		t.visitWordParts(assign.Value.Parts)
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
		t.visitWordParts(word.Parts)
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
			t.visitWordParts(arg.Parts)
		}
	}
}

func (t *Translator) visitWordParts(parts []syntax.WordPart) {
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
		default:
			fixmeCase(n)
		}
	}
}
