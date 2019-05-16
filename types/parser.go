package types

import (
	"fmt"
	"strconv"

	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/utils"
)

// States:
const (
	exElement     = 0 // Expect value literal
	exParam       = 1 // Expect value literal
	exKey         = 2 // Expect exKey or end of hash
	exValue       = 3 // Expect value
	exEntryValue  = 4 // Expect value
	exRocket      = 5 // Expect rocket
	exListComma   = 6 // Expect comma or end of array
	exParamsComma = 7 // Expect comma or end of parameter list
	exHashComma   = 8 // Expect comma or end of hash
	exName        = 9
	exEqual       = 10
	exListFirst   = 11
	exHashFirst   = 12
	exParamsFirst = 13
	exEnd         = 14
)

const ParseError = `PARSE_ERROR`

func init() {
	issue.Hard(ParseError, `%{message}`)
}

func expect(state int) (s string) {
	switch state {
	case exElement, exParam:
		s = `a literal`
	case exKey:
		s = `a hash key`
	case exValue, exEntryValue:
		s = `a hash value`
	case exRocket:
		s = `'=>'`
	case exListComma:
		s = `one of ',' or ']'`
	case exParamsComma:
		s = `one of ',' or ')'`
	case exHashComma:
		s = `one of ',' or '}'`
	case exName:
		s = `a type name`
	case exEqual:
		s = `'='`
	case exListFirst:
		s = `']' or a literal`
	case exParamsFirst:
		s = `')' or a literal`
	case exHashFirst:
		s = `'}' or a hash key`
	case exEnd:
		s = `end of expression`
	}
	return
}

func badSyntax(t *token, state int) error {
	var ts string
	if t.i == 0 {
		ts = `EOF`
	} else {
		ts = t.s
	}
	return fmt.Errorf(`expected %s, got '%s'`, expect(state), ts)
}

// Parse calls ParseFile with the string "<pcore type expression>" as the fileName
func Parse(content string) px.Value {
	return ParseFile(`<pcore type expression>`, content)
}

// ParseFile parses the given content into a px.Value. The content must be a string representation
// of a Puppet literal or a type assignment expression. Valid literals are float, integer, string,
// boolean, undef, array, hash, parameter lists, and type expressions.
//
// Double quoted strings containing interpolation expressions will be parsed into a string verbatim
// without resolving the interpolations.
func ParseFile(fileName, content string) px.Value {
	d := NewCollector()
	sr := utils.NewStringReader(content)
	p := &parser{d, sr, nil, nil}

	defer func() {
		if r := recover(); r != nil {
			if err, ok := r.(error); ok {
				panic(issue.NewReported(ParseError, issue.SeverityError, issue.H{`message`: err.Error()}, p.location(fileName)))
			}
			panic(r)
		}
	}()

	// deal with alias syntax "type X = <the rest>"
	typeName := ``
	t := p.nextToken()
	if t.i == identifier && t.s == `type` {
		t = p.nextToken()
		switch t.i {
		case name:
			typeName = t.s
			t = p.nextToken()
			if t.i != equal {
				panic(badSyntax(t, exEqual))
			}
			p.parse(p.nextToken())
		case rocket:
			// allow type => <something> as top level expression
			p.parse(p.nextToken())
			d.Add(singletonMap(`type`, d.PopLast()))
		default:
			panic(badSyntax(t, exName))
		}
	} else {
		p.parse(t)
	}

	dv := d.Value()
	if typeName != `` {
		dv = NamedType(px.RuntimeNameAuthority, typeName, dv)
	}
	return dv
}

type parser struct {
	d  px.Collector
	sr *utils.StringReader
	v  *DeferredType
	lt *token
}

func (p *parser) location(fileName string) issue.Location {
	return issue.NewLocation(fileName, p.sr.Line(), p.sr.Column()-len(p.lt.s))
}

func (p *parser) nextToken() *token {
	t := nextToken(p.sr)
	p.lt = t
	return t
}

func (p *parser) parse(t *token) {
	tk := p.element(t)
	if tk != nil {
		if tk.i != end {
			panic(badSyntax(tk, exListComma))
		}
		p.d.Add(undef)
	} else {
		tk = p.handleTypeArgs()
		if tk.i == rocket {
			// Accept top level x => y expression as a singleton hash
			key := p.d.PopLast()
			tk = p.element(p.nextToken())
			if tk == nil {
				tk = p.handleTypeArgs()
				if tk.i == end {
					p.d.Add(singleMap(key, p.d.PopLast()))
				}
			}
		}
		if tk.i != end {
			panic(badSyntax(tk, exEnd))
		}
	}
}

func (p *parser) array() {
	arrayHash := false
	d := p.d
	d.AddArray(0, func() {
		var tk *token
		var rockLhs px.Value
		for {
			tk = p.element(p.nextToken())
			if tk != nil {
				// Right bracket instead of element indicates an empty array or an extraneous comma. Both are OK
				if tk.i == rightBracket {
					return
				}
				panic(badSyntax(tk, exListFirst))
			}
			tk = p.handleTypeArgs()

			if rockLhs != nil {
				// Last two elements is a hash entry
				d.Add(WrapHashEntry(rockLhs, d.PopLast()))
				rockLhs = nil
				arrayHash = true
			}

			// Comma, rocket, or right bracket must follow element
			switch tk.i {
			case rightBracket:
				return
			case comma:
				continue
			case rocket:
				rockLhs = d.PopLast()
			default:
				panic(badSyntax(tk, exListComma))
			}
		}
	})

	if arrayHash {
		// there's at least one hash entry in the array
		d.Add(convertHashEntries(d.PopLast().(*Array)))
	}
}

// params is like array but using () instead of []
func (p *parser) params() {
	arrayHash := false
	d := p.d
	d.AddArray(0, func() {
		var tk *token
		var rockLhs px.Value
		for {
			tk = p.element(p.nextToken())
			if tk != nil {
				// Right parenthesis instead of element indicates an empty parameter list or an extraneous comma. Both are OK
				if tk.i == rightParen {
					return
				}
				panic(badSyntax(tk, exParamsFirst))
			}
			tk = p.handleTypeArgs()

			if rockLhs != nil {
				// Last two elements is a hash entry
				d.Add(WrapHashEntry(rockLhs, d.PopLast()))
				rockLhs = nil
				arrayHash = true
			}

			// Comma, rocket, or right bracket must follow element
			switch tk.i {
			case rightParen:
				return
			case comma:
				continue
			case rocket:
				rockLhs = d.PopLast()
			default:
				panic(badSyntax(tk, exParamsComma))
			}
		}
	})

	if arrayHash {
		// there's at least one hash entry in the array
		d.Add(convertHashEntries(d.PopLast().(*Array)))
	}
}

func (p *parser) hash() {
	p.d.AddHash(0, func() {
		var tk *token
		for {
			tk = p.element(p.nextToken())
			if tk != nil {
				// Right curly brace instead of element indicates an empty hash or an extraneous comma. Both are OK
				if tk.i == rightCurlyBrace {
					return
				}
				panic(badSyntax(tk, exHashFirst))
			}
			tk = p.handleTypeArgs()

			// rocket must follow key
			if tk.i != rocket {
				panic(badSyntax(tk, exRocket))
			}

			tk = p.element(p.nextToken())
			if tk != nil {
				panic(badSyntax(tk, exValue))
			}
			tk = p.handleTypeArgs()

			// Comma or right curly brace must follow value
			switch tk.i {
			case rightCurlyBrace:
				return
			case comma:
				continue
			default:
				panic(badSyntax(tk, exHashComma))
			}
		}
	})
}

func (p *parser) element(t *token) (tk *token) {
	switch t.i {
	case leftCurlyBrace:
		p.hash()
	case leftBracket:
		p.array()
	case leftParen:
		p.params()
	case integer:
		i, err := strconv.ParseInt(t.s, 0, 64)
		if err != nil {
			panic(err)
		}
		p.d.Add(WrapInteger(i))
	case float:
		f, err := strconv.ParseFloat(t.s, 64)
		if err != nil {
			panic(err)
		}
		p.d.Add(WrapFloat(f))
	case identifier:
		switch t.s {
		case `true`:
			p.d.Add(BooleanTrue)
		case `false`:
			p.d.Add(BooleanFalse)
		case `default`:
			p.d.Add(WrapDefault())
		case `undef`:
			p.d.Add(undef)
		default:
			p.d.Add(WrapString(t.s))
		}
	case stringLiteral:
		p.d.Add(WrapString(t.s))
	case regexpLiteral:
		p.d.Add(WrapRegexp(t.s))
	case name:
		p.v = &DeferredType{tn: t.s}
	default:
		tk = t
	}
	return
}

// convertHashEntries converts consecutive HashEntry elements found in an array to a Hash. This is
// to permit the x => y notation inside an array.
func convertHashEntries(av *Array) *Array {
	es := make([]px.Value, 0, av.Len())

	var en []*HashEntry
	av.Each(func(v px.Value) {
		if he, ok := v.(*HashEntry); ok {
			if en == nil {
				en = []*HashEntry{he}
			} else {
				en = append(en, he)
			}
		} else {
			if en != nil {
				es = append(es, WrapHash(en))
				en = nil
			}
			es = append(es, v)
		}
	})
	if en != nil {
		es = append(es, WrapHash(en))
	}
	return WrapValues(es)
}

// handleTypeArgs deals with type names that are followed by an array, a hash, or a parameter list,
// e.g. Integer[0,8], Person{...}, Point(23, 14).
func (p *parser) handleTypeArgs() *token {
	tk := p.nextToken()
	if p.v == nil {
		return tk
	}

	sv := p.v
	p.v = nil
	switch tk.i {
	case leftBracket:
		p.array()
		ll := p.d.PopLast().(*Array)
		n := ll.Len()
		if n == 0 {
			// Empty type parameter list is not permitted
			panic(badSyntax(&token{i: rightBracket, s: `]`}, exElement))
		}
		sv.params = ll.AppendTo(make([]px.Value, 0, n))
		p.d.Add(sv)
	case leftCurlyBrace:
		p.hash()
		sv.params = []px.Value{p.d.PopLast()}
		p.d.Add(sv)
	case leftParen:
		p.params()
		ll := p.d.PopLast().(*Array)
		dt := sv.tn
		if dt != `Deferred` {
			params := append(make([]px.Value, 0, ll.Len()+1), WrapString(dt))
			p.d.Add(NewDeferred(`new`, ll.AppendTo(params)...))
		} else {
			params := ll.Slice(1, ll.Len()).AppendTo(make([]px.Value, 0, ll.Len()-1))
			p.d.Add(NewDeferred(ll.At(0).String(), params...))
		}
	default:
		p.d.Add(sv)
		return tk
	}
	return p.nextToken()
}
