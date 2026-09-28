package convert

import (
	"bytes"
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTemplateLiteral(t *testing.T) {
	testCases := []struct {
		name          string
		value         string
		base64Encoded bool
		want          string
	}{
		{name: "data values are decoded", value: "cGlua2llcGll", base64Encoded: true, want: "pinkiepie"},
		{name: "stringData values are kept", value: "pinkiepie", want: "pinkiepie"},
		{name: "template actions are escaped", value: "hello {{ .Name }}", want: "hello {{ .Name }}"},
		{name: "decoded template actions are escaped", value: "e3sgLk5hbWUgfX0=", base64Encoded: true, want: "{{ .Name }}"},
		{name: "empty data values stay empty", value: "", base64Encoded: true, want: ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			literal, err := templateLiteral(tc.value, tc.base64Encoded)
			require.NoError(t, err)

			// ESO renders template data with text/template, so check what it would store
			tmpl, err := template.New("value").Parse(literal)
			require.NoError(t, err)
			var rendered bytes.Buffer
			require.NoError(t, tmpl.Execute(&rendered, map[string]string{"Name": "rendered"}))
			assert.Equal(t, tc.want, rendered.String())
		})
	}
}

func TestTemplateLiteralRejectsWhatItCannotCarry(t *testing.T) {
	_, err := templateLiteral("pinkiepie", true)
	assert.ErrorContains(t, err, "not valid base64")

	_, err = templateLiteral("/w==", true)
	assert.ErrorContains(t, err, "binary")
}
