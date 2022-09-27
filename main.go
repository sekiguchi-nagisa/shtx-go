package main

import (
	"bufio"
	"fmt"
	flags "github.com/jessevdk/go-flags"
	"io"
	"mvdan.cc/sh/v3/syntax"
	"os"
)

type Options struct {
	DumpAST string `short:"d" long:"dump" description:"dump internal ast to specified file (default to stderr)" optional:"true" optional-value:"/dev/stderr"`
	Args    struct {
		SCRIPT string
	} `positional-args:"yes" required:"yes"`
}

func dump(r io.Reader, w io.Writer) {
	f, e := syntax.NewParser().Parse(r, "")
	if e != nil {
		fmt.Fprintf(os.Stderr, "[error] %s\n", e.Error())
		os.Exit(1)
	}
	syntax.DebugPrint(w, f)
}

func translate(r io.Reader, w io.Writer) {
	f, e := syntax.NewParser().Parse(r, "")
	if e != nil {
		fmt.Fprintf(os.Stderr, "[error] %s\n", e.Error())
		os.Exit(1)
	}
	syntax.NewPrinter().Print(w, f)
}

func main() {
	options := Options{}
	p := flags.NewParser(&options, flags.Default)
	_, e := p.Parse()
	if e != nil {
		p.WriteHelp(os.Stderr)
		os.Exit(1)
	}

	fmt.Printf("%s\n", options.DumpAST)
	fmt.Printf("%s\n", options.Args)

	f, e := os.Open(options.Args.SCRIPT)
	defer f.Close()
	if e != nil {
		fmt.Fprintf(os.Stderr, "cannot open file: %s, caused by `%s'\n", options.Args.SCRIPT, e.Error())
		os.Exit(1)
	}

	b := bufio.NewReader(f)
	if len(options.DumpAST) == 0 {
		translate(b, os.Stdout)
	} else {
		d, e := os.Open(options.DumpAST)
		defer d.Close()
		if e != nil {
			fmt.Fprintf(os.Stderr, "cannot open file: %s, caused by `%s'\n", options.DumpAST, e.Error())
			os.Exit(1)
		}
		dump(b, d)
	}
}
