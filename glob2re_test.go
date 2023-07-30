package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

var testCases = []struct {
	before string
	after  string
}{
	// base
	{`abc`, `$/^abc$/`},
	{`abc*`, `$/^abc.*$/`},
	{`*`, `$/^.*$/`},
	{`abc?1`, `$/^abc.1$/`},
	{`?`, `$/^.$/`},

	// char set
	{`[abc]`, `$/^[abc]$/`},
	{`[a-zA-Z_]*`, `$/^[a-zA-Z_].*$/`},
	{`[[]`, `$/^[[]$/`},
	{`[\]]`, `$/^[\]]$/`},
	{`[!!a-bc]`, `$/^[^!a-bc]$/`},
	{`[^!a-bc^]`, `$/^[^!a-bc^]$/`},

	// escape meta character for pcre
	{`\`, `$/^\\$/`},
	{`\\`, `$/^\\$/`},
	{`\\\`, `$/^\\\\$/`},
	{`\b`, `$/^b$/`},
	{`$^`, `$/^\$\^$/`},
	{`/`, `$/^\/$/`},
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
`, `$/^\n$/`},
	{`(34)`, `$/^\(34\)$/`},
	{`\(34\)\\)`, `$/^\(34\)\\\)$/`},
	{`abc|123`, `$/^abc\|123$/`},
	{`abc\|123`, `$/^abc\|123$/`},
	{`a{2}`, `$/^a\{2\}$/`},
	{`a\{2\}`, `$/^a\{2\}$/`},
}

func TestGlobToRegex(t *testing.T) {
	for _, testCase := range testCases {
		actual := GlobToRegex(testCase.before)
		assert.Equal(t, testCase.after, actual)
	}
}
