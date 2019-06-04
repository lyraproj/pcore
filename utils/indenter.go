package utils

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// An Indenter helps building strings where all newlines are supposed to be followed by
// a sequence of zero or many spaces that reflect an indent level.
type Indenter struct {
	b *bytes.Buffer
	i int
}

// An Indentable can create build a string representation of itself using an Indenter
type Indentable interface {
	fmt.Stringer

	// AppendTo appends a string representation of the Node to the Indenter
	AppendTo(w *Indenter)
}

// IndentedString will produce a string from an Indentable using an Indenter
func IndentedString(ia Indentable) string {
	i := NewIndenter()
	ia.AppendTo(i)
	return i.String()
}

// NewIndenter creates a new Indenter for indent level zero
func NewIndenter() *Indenter {
	return &Indenter{b: &bytes.Buffer{}, i: 0}
}

// NewIndenterWithLevel creates a new Indenter for the given level
func NewIndenterWithLevel(level int) *Indenter {
	return &Indenter{b: &bytes.Buffer{}, i: level}
}

// Len returns the current number of bytes that has been appended to the indenter
func (i *Indenter) Len() int {
	return i.b.Len()
}

// Level returns the indent level for the indenter
func (i *Indenter) Level() int {
	return i.i
}

// Reset resets the internal buffer. It does not reset the indent
func (i *Indenter) Reset() {
	i.b.Reset()
}

// String returns the current string that has been built using the indenter. Trailing whitespaces
// are deleted from all lines.
func (i *Indenter) String() string {
	n := bytes.NewBuffer(make([]byte, 0, i.b.Len()))
	wb := &bytes.Buffer{}
	for {
		r, _, err := i.b.ReadRune()
		if err == io.EOF {
			break
		}
		if r == ' ' || r == '\t' {
			// Defer whitespace output
			wb.WriteByte(byte(r))
			continue
		}
		if r == '\n' {
			// Truncate trailing space
			wb.Reset()
		} else {
			if wb.Len() > 0 {
				n.Write(wb.Bytes())
				wb.Reset()
			}
		}
		n.WriteRune(r)
	}
	return n.String()
}

// WriteString appends a string to the internal buffer without checking for newlines
func (i *Indenter) WriteString(s string) (n int, err error) {
	return i.b.WriteString(s)
}

// Write appends a slice of bytes to the internal buffer without checking for newlines
func (i *Indenter) Write(p []byte) (n int, err error) {
	return i.b.Write(p)
}

// AppendRune appends a rune to the internal buffer without checking for newlines
func (i *Indenter) AppendRune(r rune) {
	i.b.WriteRune(r)
}

// Append appends a string to the internal buffer without checking for newlines
func (i *Indenter) Append(s string) {
	WriteString(i.b, s)
}

// AppendIndented is like Append but replaces all occurrences of newline with an indented newline
func (i *Indenter) AppendIndented(s string) {
	for ni := strings.IndexByte(s, '\n'); ni >= 0; ni = strings.IndexByte(s, '\n') {
		if ni > 0 {
			WriteString(i.b, s[:ni])
		}
		i.NewLine()
		ni++
		if ni >= len(s) {
			return
		}
		s = s[ni:]
	}
	if len(s) > 0 {
		WriteString(i.b, s)
	}
}

// AppendBool writes the string "true" or "false" to the internal buffer
func (i *Indenter) AppendBool(b bool) {
	var s string
	if b {
		s = `true`
	} else {
		s = `false`
	}
	WriteString(i.b, s)
}

// AppendInt writes the result of calling strconf.Itoa() in the given argument
func (i *Indenter) AppendInt(b int) {
	WriteString(i.b, strconv.Itoa(b))
}

// Indent returns a new Indenter instance that shares the same buffer but has an
// indent level that is increased by one.
func (i *Indenter) Indent() *Indenter {
	return &Indenter{b: i.b, i: i.i + 1}
}

// Printf formats according to a format specifier and writes to the internal buffer.
func (i *Indenter) Printf(s string, args ...interface{}) {
	Fprintf(i.b, s, args...)
}

// NewLine writes a newline followed by the current indent after trimming trailing whitespaces
func (i *Indenter) NewLine() {
	i.b.WriteByte('\n')
	for n := 0; n < i.i; n++ {
		WriteString(i.b, `  `)
	}
}
