package main

import "strings"

type glob2RegexTranslator struct {
	runes  []rune
	length int
	index  int
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
		} else if ch == ']' {
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
		sb.WriteRune(ch)
	}
	sb.WriteRune(']')
	return sb.String()
}

func (g *glob2RegexTranslator) translate(glob string) string {
	g.runes = []rune(glob)
	g.length = len(g.runes)
	g.index = 0

	sb := strings.Builder{}
	sb.Grow(g.length)
	sb.WriteString("$/^")
	for ; g.index < g.length; g.index++ {
		ch := g.runes[g.index]
		switch ch {
		case '\\':
			if g.index+1 < len(g.runes) {
				g.index++
				next := g.runes[g.index]
				switch next {
				case '?', '*', '\\', '[', ']', '/', '$', '^', '.', '+', '(', ')', '{', '}', '|':
					sb.WriteRune('\\')
					ch = next
				case '\n':
					sb.WriteRune('\\')
					ch = 'n'
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
		case '[':
			sb.WriteString(g.translateCharSet())
		case ']', '/', '$', '^', '.', '+', '(', ')', '{', '}', '|':
			sb.WriteRune('\\')
			sb.WriteRune(ch)
		default:
			sb.WriteRune(ch)
		}
	}
	sb.WriteString("$/")
	return sb.String()
}

func GlobToRegex(glob string) string {
	glob2regex := glob2RegexTranslator{}
	return glob2regex.translate(glob)
}
