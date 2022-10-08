package main

import (
	"fmt"
	"io"
	"mvdan.cc/sh/v3/syntax"
)

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
}

func (t *Translator) visitCommand(cmd syntax.Command, redirs []*syntax.Redirect) {

	switch n := cmd.(type) {
	case nil:
		t.emit(":") //FIXME: '> /dev/null' or '< /dev/null' semantics
	case *syntax.CallExpr:
		t.visitCallExpr(n)
	default:
		panic(fmt.Sprintf("unsupported node type %T", n))
	}
	if redirs != nil {
		panic("FIXME: unsupported: redirection")
	} else {
		t.newline()
	}
}

func (t *Translator) visitAssigns(assigns []*syntax.Assign) {
	if len(assigns) > 0 {
		panic("FIXME: unsupported: env assignment")
	}
}

func isCmdName(word *syntax.Word) bool {
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

func (t *Translator) visitCallExpr(expr *syntax.CallExpr) {
	t.visitAssigns(expr.Assigns)
	if len(expr.Args) == 0 {
		return
	}
	for i, arg := range expr.Args {
		if i == 0 {
			if isCmdName(arg) {
				t.emit(arg.Parts[0].(*syntax.Lit).Value)
			} else {
				panic("FIXME: non command literal")
			}
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
		default:
			panic(fmt.Sprintf("unsupported node type %T", n))
		}
	}
}
