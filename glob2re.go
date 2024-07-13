package main

import "strings"

type Glob2RegexOption struct {
	startsWith bool // starts with pattern
	endsWith   bool // ends with pattern
}

type glob2RegexTranslator struct {
	runes  []rune
	length int
	index  int
	option Glob2RegexOption
}

func (g *glob2RegexTranslator) translateCharSet() string {
	// check balance of [ ]
	if g.runes[g.index] != '[' {
		return "" // normally unreachable
	}
	closeIndex := g.index + 1
	for i := g.index + 1; i < g.length; i++ {
		ch := g.runes[i]
		if ch == '\\' && i+1 < g.length {
			i++
			continue
		} else if ch == ']' && i > closeIndex {
			closeIndex = i
			break
		}
	}
	if closeIndex == g.index+1 { // unclosed char set and empty char set is invalid
		return "\\["
	}

	// translate char set
	sb := strings.Builder{}
	sb.Grow(closeIndex - g.index)
	sb.WriteRune('[')
	g.index++ // skip [
	switch g.runes[g.index] {
	case '!', '^':
		sb.WriteRune('^')
		g.index++
	}
	for ; g.index < closeIndex; g.index++ {
		ch := g.runes[g.index]
		if ch == '[' || ch == ']' {
			sb.WriteRune('\\')
		} else if ch == '\\' && g.index+1 < closeIndex {
			next := g.runes[g.index+1]
			if next == ']' || next == '[' {
				g.index++
				ch = next
				sb.WriteRune('\\')
			}
		}
		sb.WriteRune(ch)
	}
	sb.WriteRune(']')
	return sb.String()
}

func (g *glob2RegexTranslator) consumeStar() {
	for g.index+1 < g.length {
		if g.runes[g.index+1] == '*' {
			g.index++
		} else {
			break
		}
	}
}

func (g *glob2RegexTranslator) translate(glob string) string {
	g.runes = []rune(glob)
	g.length = len(g.runes)
	g.index = 0

	sb := strings.Builder{}
	sb.Grow(g.length)

	if g.option.startsWith {
		sb.WriteString("^")
	}
	for ; g.index < g.length; g.index++ {
		ch := g.runes[g.index]
		switch ch {
		case '\\':
			if g.index+1 < len(g.runes) {
				g.index++
				next := g.runes[g.index]
				switch next {
				case '?', '*', '[', ']', '\\', '/', '$', '^', '.', '+', '(', ')', '{', '}', '|':
					sb.WriteRune('\\')
					ch = next
				default:
					ch = next
				}
			} else {
				sb.WriteRune('\\')
			}
			sb.WriteRune(ch)
		case '\n':
			sb.WriteString("\\n")
		case '?':
			sb.WriteRune('.')
		case '*':
			sb.WriteString(".*")
			g.consumeStar()
		case '[':
			sb.WriteString(g.translateCharSet())
		case ']', '/', '$', '^', '.', '+', '(', ')', '{', '}', '|':
			sb.WriteRune('\\')
			sb.WriteRune(ch)
		default:
			sb.WriteRune(ch)
		}
	}
	if g.option.endsWith {
		sb.WriteString("$")
	}
	return sb.String()
}

func GlobToRegexWith(value string, option Glob2RegexOption) string {
	glob2regex := glob2RegexTranslator{option: option}
	return glob2regex.translate(value)
}

// GlobToRegex translate value (glob pattern) to regex
func GlobToRegex(value string) string {
	return GlobToRegexWith(value, Glob2RegexOption{startsWith: true, endsWith: true})
}

// LiteralGlobToRegex translate escaped command argument part to regex literal
func LiteralGlobToRegex(value string) string {
	value = UnescapeNonGlobMeta(value)
	sb := strings.Builder{}
	sb.WriteString("$/")
	sb.WriteString(GlobToRegex(value))
	sb.WriteRune('/')
	return sb.String()
}

// UnescapeNonGlobMeta unescape backslash (if not escape glob meta)
//
// value must be command argument part
func UnescapeNonGlobMeta(value string) string {
	runes := []rune(value)
	sb := strings.Builder{}
	size := len(runes)
	sb.Grow(size)
	for i := 0; i < size; i++ {
		ch := runes[i]
		if ch == '\\' && i+1 < size {
			i++
			next := runes[i]
			switch next {
			case '*', '?', '[', ']', '\\':
				sb.WriteRune('\\') // not skip backslash
				sb.WriteRune(next)
			case '\n':
				continue
			default:
				sb.WriteRune(next)
			}
		} else {
			sb.WriteRune(ch)
		}
	}
	return sb.String()
}
