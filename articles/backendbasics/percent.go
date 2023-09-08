package backendbasics

import (
	"strings"
)

func Escape(s string) string {
	// check if s is already url-escaped
	for i := range s {
		switch {
		case s[i] == '%' && i > len(s)-2: // not enough characters left to be a valid escape sequence
			return newEscaped(s)
		case 'a' <= s[i] && s[i] <= 'z', // lowercase ascii letters
			'A' <= s[i] && s[i] <= 'Z',                         // uppercase ascii letters
			'0' <= s[i] && s[i] <= '9',                         // digits
			s[i] == '-', s[i] == '.', s[i] == '_', s[i] == '~', // unreserved characters
			s[i] == '%' && percentToByte[s[i+1:i+3]] != 0: // valid escape sequence
		default:
			return newEscaped(s)
		}
	}
	return s // s is already url-escaped
}

func Unescape(s string) string {
	// check if s actually needs to be unescaped
	for i := range s {
		// if we find a %, and there are at least two characters left, and the next two characters are valid hex digits, then s needs to be unescaped
		if s[i] == '%' && i < len(s)-2 {
			return newUnescaped(s)
		}
	}
	return s // s is already url-unescaped
}

func newUnescaped(s string) string {
	buf := new(strings.Builder)
	buf.Grow(len(s))
	for i := 0; i < len(s); {
		if s[i] == '%' && i < len(s)-2 {
			if b, ok := percentToByte[s[i+1:i+3]]; ok {
				buf.WriteByte(b)
				i += 3
				continue
			}
		}
		buf.WriteByte(s[i])
		i++
	}
	return buf.String()
}

func newEscaped(s string) string {
	buf := new(strings.Builder)
	buf.Grow(len(s))
	for i := range s {
		switch {
		case s[i] == '%' && i > len(s)-2: // not enough characters left to be a valid escape sequence
			// we can't escape this character, so just write it to the buffer
			buf.WriteByte(s[i])
		case 'a' <= s[i] && s[i] <= 'z', // lowercase ascii letters
			'A' <= s[i] && s[i] <= 'Z', // uppercase ascii letters
			'0' <= s[i] && s[i] <= '9', // digits
			s[i] == '-', s[i] == '.', s[i] == '_', s[i] == '~':
			buf.WriteByte(s[i]) // unreserved characters
		default:
			buf.WriteString(byteToPercent[s[i]]) // escape this character
		}
	}
	return buf.String()
}
