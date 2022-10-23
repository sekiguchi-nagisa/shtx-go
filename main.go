package main

import (
	"fmt"
	flags "github.com/jessevdk/go-flags"
	"os"
	"runtime/debug"
	"time"
)

type Options struct {
	Version       bool   `short:"v" long:"version" description:"Show version info"`
	Type          string `short:"t" long:"type" description:"Type of translation" choice:"eval" choice:"source" choice:"none" default:"eval"`
	DumpAST       string `short:"d" long:"dump" description:"Dump internal ast to specified file (default to stderr)" optional:"true" optional-value:"/dev/stderr"`
	SaveCrashDump bool   `long:"crash-dump" description:"Save crash dump to file"`
	Args          struct {
		SCRIPT string
	} `positional-args:"yes"`
}

func getVersion() string {
	info, ok := debug.ReadBuildInfo()
	if ok {
		rev := "unknown"
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" {
				rev = setting.Value
				break
			}
		}
		return fmt.Sprintf("%s (%s)", info.Main.Version, rev)
	} else {
		return "(unknown)"
	}
}

func saveCrashDump(err error) {
	t := time.Now().Format(time.RFC3339)
	name := fmt.Sprintf("crash_shtx-go_%s.log", t)
	f, _ := os.Create(name)
	if f != nil {
		header := fmt.Sprintf("+++++  build info  +++++\n%s\n\n", getVersion())
		f.WriteString(header)
		f.WriteString(err.Error())
	}
	fmt.Fprintf(os.Stderr, "save crash dump:\n\t%s\n", name)
}

var transTypes = map[string]TranslationType{
	"none":   TranslateNone,
	"eval":   TranslateEval,
	"source": TranslateSource,
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

	buf, e := os.ReadFile(script)
	if e != nil {
		fmt.Fprintln(os.Stderr, e.Error())
		os.Exit(1)
	}
	tx := NewTranslator(transTypes[options.Type])
	if len(options.DumpAST) != 0 {
		d, e := os.Create(options.DumpAST)
		defer d.Close()
		if e != nil {
			fmt.Fprintln(os.Stderr, e.Error())
			os.Exit(1)
		}
		tx.SetDump(d)
	}

	if e := tx.Translate(buf, os.Stdout); e != nil {
		fmt.Fprintln(os.Stderr, e.Error())
		if options.SaveCrashDump {
			saveCrashDump(e)
		}
		os.Exit(1)
	}
}
