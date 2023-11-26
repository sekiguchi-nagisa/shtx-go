package main

import (
	"errors"
	"fmt"
	"github.com/jessevdk/go-flags"
	"os"
	"path/filepath"
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

var version = "" // for version embedding (specified like "-X main.version=v0.1.0")

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
		var v = info.Main.Version
		if version != "" { // set by "-X main.version=v0.1.0"
			v = version
		}
		return fmt.Sprintf("%s (%s)", v, rev)
	} else {
		return "(unknown)"
	}
}

func saveCrashDump(err error) {
	t := time.Now().Format(time.RFC3339)
	path := fmt.Sprintf("crash_shtx-go_%s.log", t)
	f, _ := os.Create(path)
	if f != nil {
		defer func(f *os.File) {
			_ = f.Close()
		}(f)
		header := fmt.Sprintf("+++++  build info  +++++\n%s\n\n", getVersion())
		_, _ = f.WriteString(header)
		_, _ = f.WriteString(err.Error())
	}
	p, e := filepath.Abs(path)
	if e == nil {
		path = p
	}
	_, _ = fmt.Fprintf(os.Stderr, "save crash dump:\n\t%s\n", path)
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
		var flagsErr *flags.Error
		if errors.As(e, &flagsErr) && flagsErr.Type == flags.ErrHelp {
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
		_, _ = fmt.Fprintln(os.Stderr, "the argument `SCRIPT` was not provided")
		p.WriteHelp(os.Stderr)
		os.Exit(1)
	}
	if script == "-" {
		script = "/dev/stdin"
	}

	buf, e := os.ReadFile(script)
	if e != nil {
		_, _ = fmt.Fprintln(os.Stderr, e.Error())
		os.Exit(1)
	}
	tx := NewTranslator(transTypes[options.Type])
	if len(options.DumpAST) != 0 {
		d, e := os.Create(options.DumpAST)
		defer func(d *os.File) {
			_ = d.Close()
		}(d)
		if e != nil {
			_, _ = fmt.Fprintln(os.Stderr, e.Error())
			os.Exit(1)
		}
		tx.SetDump(d)
	}

	if e := tx.Translate(buf, os.Stdout); e != nil {
		_, _ = fmt.Fprintln(os.Stderr, e.Error())
		if options.SaveCrashDump {
			saveCrashDump(e)
		}
		os.Exit(1)
	}
}
