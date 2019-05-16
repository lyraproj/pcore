package types

import (
	"bytes"
	"errors"
	"fmt"
	"unicode"
	"unicode/utf8"

	"github.com/lyraproj/pcore/utils"
)

type tokenType int

const (
	end = iota
	name
	identifier
	integer
	float
	regexpLiteral
	stringLiteral
	leftBracket
	rightBracket
	leftCurlyBrace
	rightCurlyBrace
	leftParen
	rightParen
	comma
	dot
	rocket
	equal
)

func (t tokenType) String() (s string) {
	switch t {
	case end:
		s = "end"
	case name:
		s = "name"
	case identifier:
		s = "identifier"
	case integer:
		s = "integer"
	case float:
		s = "float"
	case regexpLiteral:
		s = "regexp"
	case stringLiteral:
		s = "string"
	case leftBracket:
		s = "leftBracket"
	case rightBracket:
		s = "rightBracket"
	case leftCurlyBrace:
		s = "leftCurlyBrace"
	case rightCurlyBrace:
		s = "rightCurlyBrace"
	case leftParen:
		s = "leftParen"
	case rightParen:
		s = "rightParen"
	case comma:
		s = "comma"
	case dot:
		s = "dot"
	case rocket:
		s = "rocket"
	case equal:
		s = "equal"
	default:
		s = "*UNKNOWN TOKEN*"
	}
	return
}

type token struct {
	s string
	i tokenType
}

func (t token) String() string {
	return fmt.Sprintf("%s: '%s'", t.i.String(), t.s)
}

func badToken(r rune) error {
	return fmt.Errorf("unexpected character '%c'", r)
}

func nextToken(sr *utils.StringReader) (t *token) {
	for {
		r := sr.Next()
		if r == utf8.RuneError {
			panic(errors.New("unicode error"))
		}
		if r == 0 {
			return &token{``, end}
		}

		switch r {
		case ' ', '\t', '\n':
			continue
		case '#':
			consumeLineComment(sr)
			continue
		case '\'', '"':
			t = &token{consumeString(sr, r), stringLiteral}
		case '/':
			t = &token{consumeRegexp(sr), regexpLiteral}
		case '{':
			t = &token{`{`, leftCurlyBrace}
		case '}':
			t = &token{`}`, rightCurlyBrace}
		case '[':
			t = &token{`[`, leftBracket}
		case ']':
			t = &token{`]`, rightBracket}
		case '(':
			t = &token{`(`, leftParen}
		case ')':
			t = &token{`)`, rightParen}
		case ',':
			t = &token{`,`, comma}
		case '.':
			t = &token{`.`, dot}
		case '=':
			r = sr.Peek()
			if r == '>' {
				sr.Next()
				t = &token{`=>`, rocket}
			} else {
				t = &token{`=`, equal}
			}
		case '-', '+':
			n := sr.Next()
			if n < '0' || n > '9' {
				panic(badToken(r))
			}
			buf := bytes.NewBufferString(string(r))
			tkn := consumeNumber(sr, n, buf, integer)
			t = &token{buf.String(), tkn}
		default:
			var tkn tokenType
			buf := bytes.NewBufferString(``)
			if r >= '0' && r <= '9' {
				tkn = consumeNumber(sr, r, buf, integer)
			} else if r >= 'A' && r <= 'Z' {
				consumeTypeName(sr, r, buf)
				tkn = name
			} else if r >= 'a' && r <= 'z' {
				consumeIdentifier(sr, r, buf)
				tkn = identifier
			} else {
				panic(badToken(r))
			}
			t = &token{buf.String(), tkn}
		}
		break
	}

	return t
}

func consumeLineComment(sr *utils.StringReader) {
	for {
		switch sr.Next() {
		case 0, '\n':
			return
		case utf8.RuneError:
			panic(errors.New("unicode error"))
		}
	}
}

func consumeUnsignedInteger(sr *utils.StringReader, buf *bytes.Buffer) {
	for {
		r := sr.Peek()
		switch r {
		case utf8.RuneError:
			panic(errors.New("unicode error"))
		case 0:
		case '.':
			panic(badToken(r))
		default:
			if r >= '0' && r <= '9' {
				sr.Next()
				buf.WriteRune(r)
				continue
			}
			if unicode.IsLetter(r) {
				sr.Next()
				panic(badToken(r))
			}
			return
		}
	}
}

func consumeExponent(sr *utils.StringReader, buf *bytes.Buffer) {
	for {
		r := sr.Next()
		switch r {
		case 0:
			panic(errors.New("unexpected end"))
		case '+', '-':
			buf.WriteRune(r)
			r = sr.Next()
			fallthrough
		default:
			if r >= '0' && r <= '9' {
				buf.WriteRune(r)
				consumeUnsignedInteger(sr, buf)
				return
			}
			panic(badToken(r))
		}
	}
}

func consumeHexInteger(sr *utils.StringReader, buf *bytes.Buffer) {
	for {
		r := sr.Peek()
		switch r {
		case 0:
			return
		default:
			if r >= '0' && r <= '9' || r >= 'A' && r <= 'F' || r >= 'a' && r <= 'f' {
				sr.Next()
				buf.WriteRune(r)
				continue
			}
			return
		}
	}
}

func consumeNumber(sr *utils.StringReader, start rune, buf *bytes.Buffer, t tokenType) tokenType {
	buf.WriteRune(start)
	firstZero := t != float && start == '0'
	for {
		r := sr.Peek()
		switch r {
		case 0:
			return 0
		case '0':
			sr.Next()
			buf.WriteRune(r)
			continue
		case 'e', 'E':
			sr.Next()
			buf.WriteRune(r)
			consumeExponent(sr, buf)
			return float
		case 'x', 'X':
			if firstZero {
				sr.Next()
				buf.WriteRune(r)
				r = sr.Next()
				if r >= '0' && r <= '9' || r >= 'A' && r <= 'F' || r >= 'a' && r <= 'f' {
					buf.WriteRune(r)
					consumeHexInteger(sr, buf)
					return t
				}
			}
			panic(badToken(r))
		case '.':
			if t == float {
				panic(badToken(r))
			}
			sr.Next()
			buf.WriteRune(r)
			r = sr.Next()
			if r >= '0' && r <= '9' {
				return consumeNumber(sr, r, buf, float)
			}
			panic(badToken(r))
		default:
			if r >= '0' && r <= '9' {
				sr.Next()
				buf.WriteRune(r)
				continue
			}
			return t
		}
	}
}

func consumeRegexp(sr *utils.StringReader) string {
	buf := bytes.NewBufferString(``)
	for {
		r := sr.Next()
		switch r {
		case utf8.RuneError:
			panic(badToken(r))
		case '/':
			return buf.String()
		case '\\':
			r = sr.Next()
			switch r {
			case 0:
				panic(errors.New("unterminated regexp"))
			case utf8.RuneError:
				panic(badToken(r))
			case '/': // Escape is removed
			default:
				buf.WriteByte('\\')
			}
			buf.WriteRune(r)
		case 0, '\n':
			panic(errors.New("unterminated regexp"))
		default:
			buf.WriteRune(r)
		}
	}
}

func consumeString(sr *utils.StringReader, end rune) string {
	buf := bytes.NewBufferString(``)
	for {
		r := sr.Next()
		if r == end {
			return buf.String()
		}
		switch r {
		case 0:
			panic(errors.New("unterminated string"))
		case utf8.RuneError:
			panic(badToken(r))
		case '\\':
			r := sr.Next()
			switch r {
			case 0:
				panic(errors.New("unterminated string"))
			case utf8.RuneError:
				panic(badToken(r))
			case 'n':
				r = '\n'
			case 'r':
				r = '\r'
			case 't':
				r = '\t'
			case '\\':
			default:
				if r != end {
					panic(fmt.Errorf("illegal escape '\\%c'", r))
				}
			}
			buf.WriteRune(r)
		case '\n':
			panic(errors.New("unterminated string"))
		default:
			buf.WriteRune(r)
		}
	}
}

func consumeIdentifier(sr *utils.StringReader, start rune, buf *bytes.Buffer) {
	buf.WriteRune(start)
	for {
		r := sr.Peek()
		switch r {
		case 0:
			return
		case ':':
			sr.Next()
			buf.WriteRune(r)
			r = sr.Next()
			if r == ':' {
				buf.WriteRune(r)
				r = sr.Next()
				if r >= 'a' && r <= 'z' || r == '_' {
					buf.WriteRune(r)
					continue
				}
			}
			panic(badToken(r))
		default:
			if r == '_' || r >= '0' && r <= '9' || r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' {
				sr.Next()
				buf.WriteRune(r)
				continue
			}
			return
		}
	}
}

func consumeTypeName(sr *utils.StringReader, start rune, buf *bytes.Buffer) {
	buf.WriteRune(start)
	for {
		r := sr.Peek()
		switch r {
		case 0:
			return
		case ':':
			sr.Next()
			buf.WriteRune(r)
			r = sr.Next()
			if r == ':' {
				buf.WriteRune(r)
				r = sr.Next()
				if r >= 'A' && r <= 'Z' {
					buf.WriteRune(r)
					continue
				}
			}
			panic(badToken(r))
		default:
			if r == '_' || r >= '0' && r <= '9' || r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' {
				sr.Next()
				buf.WriteRune(r)
				continue
			}
			return
		}
	}
}
