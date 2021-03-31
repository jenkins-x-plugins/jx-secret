package vault_test

import (
	"strings"
	"testing"

	"github.com/jenkins-x-plugins/jx-secret/pkg/extsecrets/editor/vault"
	"github.com/stretchr/testify/assert"
)

func TestPopulate(t *testing.T) {
	testCases := []struct {
		args     []string
		expected []string
	}{
		{
			args:     []string{"kv", "list"},
			expected: []string{"kv", "list"},
		},
		{
			args:     []string{"kv", "secret/jx/pipelineUser", "token=dummyPipelineToken"},
			expected: []string{"kv", "secret/jx/pipelineUser", "token=****"},
		},
		{
			args:     []string{"kv", "secret/jx/pipelineUser", "username=dummyPipelineUser", "token=dummyPipelineToken"},
			expected: []string{"kv", "secret/jx/pipelineUser", "username=****", "token=****"},
		},
	}

	for _, tc := range testCases {
		actual := vault.MastSecretArgs(tc.args)
		argLine := strings.Join(tc.args, " ")
		assert.Equal(t, tc.expected, actual, "for input arguments: %s", argLine)

		t.Logf("%s => %s\n", argLine, strings.Join(actual, " "))
	}
}
