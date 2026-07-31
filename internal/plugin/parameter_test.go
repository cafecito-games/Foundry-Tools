package plugin

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseParameterAcceptsJSON(t *testing.T) {
	options, err := parseParameter("json")
	require.NoError(t, err)
	require.True(t, options.JSON)
}

func TestParseParameterAcceptsAnEmptyString(t *testing.T) {
	options, err := parseParameter("")
	require.NoError(t, err)
	require.False(t, options.JSON)
}

func TestParseParameterIgnoresSurroundingWhitespaceAndEmptyEntries(t *testing.T) {
	options, err := parseParameter(" json , ")
	require.NoError(t, err)
	require.True(t, options.JSON)
}

func TestParseParameterRejectsAnUnknownKey(t *testing.T) {
	_, err := parseParameter("json,jsonn")
	require.Error(t, err)
	require.Contains(t, err.Error(), "jsonn")
}
