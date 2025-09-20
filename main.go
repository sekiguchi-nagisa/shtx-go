package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"time"

	"github.com/alecthomas/kong"
)

var CLI struct {
	Version       bool    `short:"v" help:"Show version info"`
	Type          string  `short:"t" help:"Set type of translation (eval, source, pattern, none)" enum:"eval,source,pattern,none" default:"eval"`
	String        *string `short:"c" placeholder:"string" help:"Use string as input"`
	DumpAST       *string `name:"dump" short:"d" placeholder:"file" help:"Dump internal ast to specified file (default to stderr)" optional:""`
	SaveCrashDump bool    `name:"crash-dump" help:"Save crash dump to file"`
	PatternType   string  `name:"pattern-type" short:"p" help:"Set type of pattern (whole, partial, start, end, forward-short, forward-long, backward-short, backward-long)" enum:"whole,partial,start,end,forward-short,forward-long,backward-short,backward-long" default:"whole"`
	Script        string  `arg:"" optional:""`
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
	"none":    TranslateNone,
	"eval":    TranslateEval,
	"source":  TranslateSource,
	"pattern": TranslatePattern,
}

var glob2RegexOptions = map[string]Glob2RegexOption{
	"whole":   {startsWith: true, endsWith: true},
	"partial": {startsWith: false, endsWith: false},
	"start":   {startsWith: true, endsWith: false},
	"end":     {startsWith: false, endsWith: true},
	// for `##`, `%%` op
	"forward-short":  {startsWith: true, endsWith: false, reluctant: true},                  // '#'
	"forward-long":   {startsWith: true, endsWith: false},                                   // '##'
	"backward-short": {startsWith: false, endsWith: false, backward: true},                  // '%'
	"backward-long":  {startsWith: false, endsWith: false, reluctant: true, backward: true}, // '%%'
}

func main() {
	ctx := kong.Parse(&CLI, kong.UsageOnError())
	if CLI.Version {
		fmt.Println(getVersion())
		os.Exit(0)
	}

	var buf []byte = nil
	if CLI.String != nil {
		buf = []byte(*CLI.String)
	} else {
		script := CLI.Script
		if script == "" {
			ctx.Fatalf("the argument `SCRIPT` was not provided")
		}
		if script == "-" {
			script = "/dev/stdin"
		}

		b, e := os.ReadFile(script)
		if e != nil {
			_, _ = fmt.Fprintln(os.Stderr, e.Error())
			os.Exit(1)
		}
		buf = b
	}

	// resolve features
	v, err := ParseVersion(os.Getenv("ARSH_VERSION"))
	if err != nil { // set to version limit (enable all versionRequire)
		tmp := NewDummyVersion()
		v = &tmp
	}
	featureSet := NewFeatureSetFromVersion(*v)

	tx := NewTranslatorWithFeatures(transTypes[CLI.Type], featureSet)
	if CLI.DumpAST != nil {
		dump := *CLI.DumpAST
		if dump == "" {
			dump = "/dev/stderr"
		}
		d, e := os.Create(dump)
		defer func(d *os.File) {
			_ = d.Close()
		}(d)
		if e != nil {
			_, _ = fmt.Fprintln(os.Stderr, e.Error())
			os.Exit(1)
		}
		tx.SetDump(d)
	}
	tx.glob2RegexOption = glob2RegexOptions[CLI.PatternType]

	var txError error
	tx.errorCallback = func(e error) {
		txError = e
	}
	out := bytes.Buffer{}
	if e := tx.Translate(buf, &out); e != nil {
		if txError != nil {
			_, _ = fmt.Fprintln(os.Stderr, txError.Error())
		} else {
			_, _ = fmt.Fprintln(os.Stderr, e.Error())
		}
		if CLI.SaveCrashDump {
			saveCrashDump(e)
		}
		os.Exit(1)
	}
	_, _ = os.Stdout.Write(out.Bytes())
}
