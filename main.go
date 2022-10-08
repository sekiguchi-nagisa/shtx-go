package main

import (
	"bufio"
	"fmt"
	flags "github.com/jessevdk/go-flags"
	"os"
	"runtime/debug"
)

type Options struct {
	DumpAST string `short:"d" long:"dump" description:"Dump internal ast to specified file (default to stderr)" optional:"true" optional-value:"/dev/stderr"`
	Version bool   `short:"v" long:"version" description:"Show version info"`
	Args    struct {
		SCRIPT string
	} `positional-args:"yes"`
}

func getVersion() string {
	info, ok := debug.ReadBuildInfo()
	if ok {
		return info.Main.Version
	} else {
		return "(unknown)"
	}
}

func main() {
	options := Options{}
	p := flags.NewParser(&options, flags.Default)
	if _, e := p.Parse(); e != nil {
		if flagsErr, ok := e.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
			os.Exit(0)
		} else {
			p.WriteHelp(os.Stderr)
			os.Exit(1)
		}
	}

	if options.Version {
		fmt.Println(getVersion())
		os.Exit(0)
	}

	script := options.Args.SCRIPT
	if script == "" {
		fmt.Fprintln(os.Stderr, "the argument `SCRIPT` was not provided")
		p.WriteHelp(os.Stderr)
		os.Exit(1)
	}
	if script == "-" {
		script = "/dev/stdin"
	}
	f, e := os.Open(script)
	defer f.Close()
	if e != nil {
		fmt.Fprintf(os.Stderr, "%s\n", e.Error())
		os.Exit(1)
	}

	b := bufio.NewReader(f)
	tx := NewTranslator()
	if len(options.DumpAST) != 0 {
		d, e := os.Create(options.DumpAST)
		defer d.Close()
		if e != nil {
			fmt.Fprintf(os.Stderr, "%s\n", e.Error())
			os.Exit(1)
		}
		tx.SetDump(d)
	}

	if e := tx.Translate(b, os.Stdout); e != nil {
		fmt.Fprintln(os.Stderr, e.Error())
		os.Exit(1)
	}
}
