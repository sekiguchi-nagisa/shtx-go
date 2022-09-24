package main

import (
	"bufio"
	"fmt"
	"mvdan.cc/sh/v3/syntax"
	"os"
)

func main() {
	if len(os.Args) == 1 {
		fmt.Fprintf(os.Stderr, "[usage] %s [script]\n", os.Args[0])
		os.Exit(1)
	}

	f, e := os.Open(os.Args[1])
	defer f.Close()
	if e != nil {
		fmt.Fprintf(os.Stderr, "cannot open file: %s, caused by `%s'\n", os.Args[1], e.Error())
		os.Exit(1)
	}

	b := bufio.NewReader(f)

	file, e := syntax.NewParser().Parse(b, "")
	if e != nil {
		fmt.Fprintf(os.Stderr, "[error] %s\n", e.Error())
		os.Exit(1)
	}
	syntax.NewPrinter().Print(os.Stdout, file)
}
