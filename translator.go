package main

import (
	"fmt"
	"io"
	"mvdan.cc/sh/v3/syntax"
)

type Translator struct {
	out  io.Writer // for output
	dump io.Writer // for parsed ast dump
}

func NewTranslator() *Translator {
	return &Translator{}
}

func (t *Translator) Translate(in io.Reader, out io.Writer) error {
	// reset io
	t.out = out

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
	syntax.NewPrinter().Print(t.out, f) //FIXME:
	return nil
}

func (t *Translator) SetDump(d io.Writer) {
	t.dump = d
}
