package main

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

var unescapeNoGlobMetaTestCases = []struct {
	before string
	after  string
}{
	{`~root`, `~root`},
	{`\~root`, `~root`},
	{`\`, `\`},
	{`\s`, `s`},
	{`\\s`, `\\s`},
	{`\\\s`, `\\s`},
	{`\ `, ` `},
	{`*`, `*`},
	{`\*`, `\*`},
	{`\\*`, `\\*`},
	{`?`, `?`},
	{`\?`, `\?`},
	{`\\?`, `\\?`},
	{`[`, `[`},
	{`\[`, `\[`},
	{`\\[`, `\\[`},
	{`]`, `]`},
	{`\]`, `\]`},
	{`\\]`, `\\]`},
}

func TestUnescape(t *testing.T) {
	for _, testCase := range unescapeNoGlobMetaTestCases {
		actual := UnescapeNonGlobMeta(testCase.before)
		assert.Equal(t, testCase.after, actual)
	}
}

var literalGlobTestCases = []struct {
	before string
	after  string
}{
	// base
	{`abc`, `$/^abc$/`},
	{`abc*`, `$/^abc.*$/`},
	{`*`, `$/^.*$/`},
	{`*2`, `$/^.*2$/`},
	{`1*2`, `$/^1.*2$/`},
	{`abc?1`, `$/^abc.1$/`},
	{`?`, `$/^.$/`},
	{`??*2`, `$/^...*2$/`},

	// char set
	{`[abc]`, `$/^[abc]$/`},
	{`[a-zA-Z_]*`, `$/^[a-zA-Z_].*$/`},
	{`[[]`, `$/^[\[]$/`},
	{`[\[]`, `$/^[\[]$/`},
	{`[]]`, `$/^[\]]$/`},
	{`[\]]`, `$/^[\]]$/`},
	{`[!!a-bc]`, `$/^[^!a-bc]$/`},
	{`[^!a-bc^]`, `$/^[^!a-bc^]$/`},

	// escape meta character for pcre
	{`\`, `$/^\\$/`},
	{`\s`, `$/^s$/`},
	{`\\s`, `$/^\\s$/`},
	{`\\\s`, `$/^\\s$/`},
	{`\b`, `$/^b$/`},
	{`$^`, `$/^\$\^$/`},
	{`/`, `$/^\/$/`},
	{`\/`, `$/^\/$/`},
	{`\\/`, `$/^\\\/$/`},
	{`[`, `$/^\[$/`},
	{`]`, `$/^\]$/`},
	{`[]`, `$/^\[\]$/`},
	{`.+`, `$/^\.\+$/`},
	{`\?`, `$/^\?$/`},
	{`\+\*`, `$/^\+\*$/`},
	{`\^\[\]\/\$`, `$/^\^\[\]\/\$$/`},
	{`
`, `$/^\n$/`},
	{`\
`, `$/^$/`},
	{`(34)`, `$/^\(34\)$/`},
	{`\(34\)\\)`, `$/^\(34\)\\\)$/`},
	{`abc|123`, `$/^abc\|123$/`},
	{`abc\|123`, `$/^abc\|123$/`},
	{`a{2}`, `$/^a\{2\}$/`},
	{`{1..3}`, `$/^\{1\.\.3\}$/`},
	{`\{1..3}`, `$/^\{1\.\.3\}$/`},
	{`\\{1..3}`, `$/^\\\{1\.\.3\}$/`},
	{`a\{2\}`, `$/^a\{2\}$/`},
	{`\*`, `$/^\*$/`},
	{`\\*`, `$/^\\.*$/`},

	// optimize **
	{`**`, `$/^.*$/`},
	{`***`, `$/^.*$/`},
	{`****`, `$/^.*$/`},
	{`**2`, `$/^.*2$/`},
}

func TestLiteralGlobToRegex(t *testing.T) {
	for i, testCase := range literalGlobTestCases {
		actual := LiteralGlobToRegex(testCase.before)
		assert.Equal(t, testCase.after, actual, fmt.Sprintf("index=%d, before=%s", i, testCase.before))
	}
}

var globTestCases = []struct {
	before string
	after  string
}{
	{``, `^$`},
	{`{}`, `^\{\}$`},
	{`[[]`, `^[\[]$`},
	{`[]]`, `^[\]]$`},
	{`[\[]`, `^[\[]$`},
	{`[\]]`, `^[\]]$`},
	{`[\\12]`, `^[\\12]$`},
	{`\?`, `^\?$`},
	{`\[`, `^\[$`},
	{`\]`, `^\]$`},
	{`\*`, `^\*$`},
	{`\\*`, `^\\.*$`},
	{`\\\*`, `^\\\*$`},
	{`\\\\*`, `^\\\\.*$`},
	{"\n", `^\n$`},
	{`{1..3}`, `^\{1\.\.3\}$`},
	{`\{1..3\}`, `^\{1\.\.3\}$`},
	{`(23)`, `^\(23\)$`},
	{`\(23\)`, `^\(23\)$`},
	{`\`, `^\\$`},
	{`/`, `^\/$`},
	{`\/`, `^\/$`},
	{`\a`, `^a$`},
	{`\\a`, `^\\a$`},
	{`\\\a`, `^\\a$`},
	{`\\\\a`, `^\\\\a$`},
	{`+`, `^\+$`},
	{`\+`, `^\+$`},
	{`|`, `^\|$`},
	{`\|`, `^\|$`},
	{`\\`, `^\\$`},
	{`\\\`, `^\\\\$`},
	{`\\\\`, `^\\\\$`},
	{`_[a-zA-Z_]*`, `^_[a-zA-Z_].*$`},
}

func TestGlobToRegex(t *testing.T) {
	for i, testCase := range globTestCases {
		actual := GlobToRegex(testCase.before)
		assert.Equal(t, testCase.after, actual, fmt.Sprintf("index=%d, before=%s", i, testCase.before))
	}
}
