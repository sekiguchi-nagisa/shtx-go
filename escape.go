package main

import "strings"

func unescapeCmdName(name string) string {
	sb := strings.Builder{}
	runes := []rune(name)
	sb.Grow(len(runes))
	for i := 0; i < len(runes); i++ {
		c := runes[i]
		if c == '\\' {
			i++
			next := runes[i]
			switch next {
			case '\n', '\r':
				continue
			default:
				c = next
			}
		}
		sb.WriteRune(c)
	}
	return sb.String()
}

func quoteCmdArgAsGlobStr(value string) string {
	sb := strings.Builder{}
	runes := []rune(value)
	sb.Grow(len(runes))
	index := 0
	if len(runes) > 0 && runes[0] == '~' {
		sb.WriteRune('~')
		index++
	}
	sb.WriteRune('"')
	for ; index < len(runes); index++ {
		ch := runes[index]
		if ch == '\\' {
			if index+1 < len(runes) {
				index++
				next := runes[index]
				switch next {
				case '?', '*', '[', ']', '\\', '`', '"':
					sb.WriteRune('\\')
				}
				ch = next
			} else {
				sb.WriteRune('\\')
			}
		}
		sb.WriteRune(ch)
	}
	sb.WriteRune('"')
	return sb.String()
}