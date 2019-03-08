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

func scan(sr *utils.StringReader, tf func(t token) error) (err error) {
	buf := bytes.NewBufferString(``)
	next := rune(-1)

	for {
		r := next
		if r == -1 {
			r = sr.Next()
			if r == utf8.RuneError {
				return errors.New("unicode error")
			}
		} else {
			next = -1
		}

		if r == 0 {
			break
		}

		switch r {
		case ' ', '\t', '\n':
		case '\'', '"':
			if err = consumeString(sr, r, buf); err != nil {
				return err
			}
			if err = tf(token{buf.String(), stringLiteral}); err != nil {
				return err
			}
			buf.Reset()
		case '/':
			if err = consumeRegexp(sr, buf); err != nil {
				return err
			}
			if err = tf(token{buf.String(), regexpLiteral}); err != nil {
				return err
			}
			buf.Reset()
		case '#':
			if err = consumeLineComment(sr); err != nil {
				return err
			}
		case '{':
			if err = tf(token{string(r), leftCurlyBrace}); err != nil {
				return err
			}
		case '}':
			if err = tf(token{string(r), rightCurlyBrace}); err != nil {
				return err
			}
		case '[':
			if err = tf(token{string(r), leftBracket}); err != nil {
				return err
			}
		case ']':
			if err = tf(token{string(r), rightBracket}); err != nil {
				return err
			}
		case '(':
			if err = tf(token{string(r), leftParen}); err != nil {
				return err
			}
		case ')':
			if err = tf(token{string(r), rightParen}); err != nil {
				return err
			}
		case ',':
			if err = tf(token{string(r), comma}); err != nil {
				return err
			}
		case '.':
			if err = tf(token{string(r), dot}); err != nil {
				return err
			}
		case '=':
			r = sr.Next()
			if r == '>' {
				if err = tf(token{`=>`, rocket}); err != nil {
					return err
				}
				continue
			}
			return badToken(r)
		case '-', '+':
			if r >= '0' && r <= '9' {
				var tkn tokenType
				tkn, next, err = consumeNumber(sr, r, buf, integer)
				if err != nil {
					return err
				}
				if err = tf(token{buf.String(), tkn}); err != nil {
					return err
				}
				buf.Reset()
				continue
			}
			return badToken(r)
		default:
			var tkn tokenType
			if r >= '0' && r <= '9' {
				tkn, next, err = consumeNumber(sr, r, buf, integer)
			} else if r >= 'A' && r <= 'Z' {
				next, err = consumeTypeName(sr, r, buf)
				tkn = name
			} else if r >= 'a' && r <= 'z' {
				next, err = consumeIdentifier(sr, r, buf)
				tkn = identifier
			} else {
				return badToken(r)
			}
			if err != nil {
				return err
			}
			if err = tf(token{buf.String(), tkn}); err != nil {
				return err
			}
			buf.Reset()
		}
	}
	if err = tf(token{``, end}); err != nil {
		return err
	}
	return nil
}

func consumeLineComment(sr *utils.StringReader) error {
	for {
		switch sr.Next() {
		case 0, '\n':
			return nil
		case utf8.RuneError:
			return errors.New("unicode error")
		}
	}
}

func consumeUnsignedInteger(sr *utils.StringReader, buf *bytes.Buffer) (rune, error) {
	for {
		r := sr.Next()
		switch r {
		case utf8.RuneError:
			return 0, errors.New("unicode error")
		case 0:
			return 0, nil
		case '.':
			return 0, badToken(r)
		default:
			if r >= '0' && r <= '9' {
				buf.WriteRune(r)
				continue
			}
			if unicode.IsLetter(r) {
				return 0, badToken(r)
			}
			return r, nil
		}
	}
}

func consumeExponent(sr *utils.StringReader, buf *bytes.Buffer) (rune, error) {
	for {
		r := sr.Next()
		switch r {
		case 0:
			return 0, errors.New("unexpected end")
		case '+', '-':
			buf.WriteRune(r)
			r = sr.Next()
			fallthrough
		default:
			if r >= '0' && r <= '9' {
				buf.WriteRune(r)
				return consumeUnsignedInteger(sr, buf)
			}
			return 0, badToken(r)
		}
	}
}

func consumeHexInteger(sr *utils.StringReader, buf *bytes.Buffer) (rune, error) {
	for {
		r := sr.Next()
		switch r {
		case 0:
			return r, nil
		default:
			if r >= '0' && r <= '9' || r >= 'A' && r <= 'F' || r >= 'a' && r <= 'f' {
				buf.WriteRune(r)
				continue
			}
			return r, nil
		}
	}
}

func consumeNumber(sr *utils.StringReader, start rune, buf *bytes.Buffer, t tokenType) (tokenType, rune, error) {
	buf.WriteRune(start)
	firstZero := t != float && start == '0'
	for {
		r := sr.Next()
		switch r {
		case 0:
			return t, 0, nil
		case '0':
			buf.WriteRune(r)
			continue
		case 'e', 'E':
			buf.WriteRune(r)
			n, err := consumeExponent(sr, buf)
			return float, n, err
		case 'x', 'X':
			if firstZero {
				buf.WriteRune(r)
				r = sr.Next()
				if r >= '0' && r <= '9' || r >= 'A' && r <= 'F' || r >= 'a' && r <= 'f' {
					buf.WriteRune(r)
					n, err := consumeHexInteger(sr, buf)
					return t, n, err
				}
			}
			return t, 0, badToken(r)
		case '.':
			if t == float {
				return t, 0, badToken(r)
			}
			buf.WriteRune(r)
			r = sr.Next()
			if r >= '0' && r <= '9' {
				return consumeNumber(sr, r, buf, float)
			}
			return t, 0, badToken(r)
		default:
			if r >= '0' && r <= '9' {
				buf.WriteRune(r)
				continue
			}
			return t, r, nil
		}
	}
}

func consumeRegexp(sr *utils.StringReader, buf *bytes.Buffer) error {
	for {
		r := sr.Next()
		switch r {
		case utf8.RuneError:
			return badToken(r)
		case '/':
			return nil
		case '\\':
			r = sr.Next()
			switch r {
			case 0:
				return errors.New("unterminated regexp")
			case utf8.RuneError:
				return badToken(r)
			case '/': // Escape is removed
			default:
				buf.WriteByte('\\')
			}
			buf.WriteRune(r)
		case 0, '\n':
			return errors.New("unterminated regexp")
		default:
			buf.WriteRune(r)
		}
	}
}

func consumeString(sr *utils.StringReader, end rune, buf *bytes.Buffer) error {
	for {
		r := sr.Next()
		if r == end {
			return nil
		}
		switch r {
		case 0:
			return errors.New("unterminated string")
		case utf8.RuneError:
			return badToken(r)
		case '\\':
			r := sr.Next()
			switch r {
			case 0:
				return errors.New("unterminated string")
			case utf8.RuneError:
				return badToken(r)
			case 'n':
				r = '\n'
			case 'r':
				r = '\r'
			case 't':
				r = '\t'
			case '\\':
			default:
				if r != end {
					return fmt.Errorf("illegal escape '\\%c'", r)
				}
			}
			buf.WriteRune(r)
		case '\n':
			return errors.New("unterminated string")
		default:
			buf.WriteRune(r)
		}
	}
}

func consumeIdentifier(sr *utils.StringReader, start rune, buf *bytes.Buffer) (rune, error) {
	buf.WriteRune(start)
	for {
		r := sr.Next()
		switch r {
		case 0:
			return 0, nil
		case ':':
			buf.WriteRune(r)
			r = sr.Next()
			if r == ':' {
				buf.WriteRune(r)
				r = sr.Next()
				if r >= 'a' && r <= 'z' || r == '_' {
					buf.WriteRune(r)
					continue
				}
				return 0, badToken(r)
			}
			return r, nil
		default:
			if r == '_' || r >= '0' && r <= '9' || r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' {
				buf.WriteRune(r)
				continue
			}
			return r, nil
		}
	}
}

func consumeTypeName(sr *utils.StringReader, start rune, buf *bytes.Buffer) (rune, error) {
	buf.WriteRune(start)
	for {
		r := sr.Next()
		switch r {
		case 0:
			return 0, nil
		case ':':
			buf.WriteRune(r)
			r = sr.Next()
			if r == ':' {
				buf.WriteRune(r)
				r = sr.Next()
				if r >= 'A' && r <= 'Z' {
					buf.WriteRune(r)
					continue
				}
				return 0, badToken(r)
			}
			return r, nil
		default:
			if r == '_' || r >= '0' && r <= '9' || r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' {
				buf.WriteRune(r)
				continue
			}
			return r, nil
		}
	}
}
