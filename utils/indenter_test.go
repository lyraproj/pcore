package utils

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIndenter_AppendIndented_onlyNewline(t *testing.T) {
	i := NewIndenter().Indent()
	i.AppendIndented("\n")
	require.Equal(t, "\n", i.String())
}

func TestIndenter_AppendIndented_twoNewline(t *testing.T) {
	i := NewIndenter().Indent()
	i.AppendIndented("\n\n")
	require.Equal(t, "\n\n", i.String())
}

func TestIndenter_AppendIndented_twoLines(t *testing.T) {
	i := NewIndenter().Indent()
	i.AppendIndented("\na\n")
	require.Equal(t, "\n  a\n", i.String())

	i.Reset()
	i.AppendIndented("\na\nb")
	require.Equal(t, "\n  a\n  b", i.String())
}
