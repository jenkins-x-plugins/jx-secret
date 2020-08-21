package templatertesting

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/jenkins-x/jx-helpers/pkg/cmdrunner/fakerunner"
	"github.com/jenkins-x/jx-secret/pkg/cmd/populate"
	"github.com/jenkins-x/jx-secret/pkg/extsecrets/testsecrets"
	"github.com/jenkins-x/jx-secret/pkg/schemas"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes/fake"
)

// Run runs the test cases
func (r *Runner) Run(t *testing.T) {
	require.NotEmpty(t, r.TestCases, "no TestCases supplied")
	require.NotEmpty(t, r.SchemaFile, "no SchemaFile supplied")

	schema, err := schemas.LoadSchemaFile(r.SchemaFile)
	require.NoError(t, err, "failed to load schema file %s", r.SchemaFile)

	_, o := populate.NewCmdPopulate()
	o.NoWait = true

	kubeObjects := r.KubeObjects
	o.Namespace = r.Namespace
	o.KubeClient = fake.NewSimpleClientset(testsecrets.AddVaultSecrets(kubeObjects...)...)

	runner := &fakerunner.FakeRunner{}
	o.CommandRunner = runner.Run

	for _, tc := range r.TestCases {
		objName := tc.ObjectName
		if tc.Requirements != nil {
			o.Requirements = tc.Requirements
		}
		if o.Requirements == nil {
			if tc.Dir != "" {
				o.Dir = tc.Dir
			}
			if o.Dir == "" {
				o.Dir = r.Dir
			}
			require.NotEmpty(t, o.Dir, "you must either specify Requirements or a Dir on the Runner or TestCase to be able to detect the Requirements to use the the template generation")
		}
		object := schema.Spec.FindObject(objName)
		require.NotNil(t, object, "could not find schema for object name %s", objName)

		propName := tc.Property
		property := object.FindProperty(propName)
		require.NotNil(t, property, "could not find property for object %s name %s", objName, propName)

		templateText := property.Template
		require.NotEmpty(t, templateText, "no template defined for object %s property name %s", objName, propName)

		text, err := o.EvaluateTemplate(objName, propName, templateText)
		require.NoError(t, err, "failed to evaluate template for object %s property name %s", objName, propName)

		message := fmt.Sprintf("test %s for object %s property name %s", tc.TestName, objName, propName)
		require.NotEmpty(t, text, message)

		format := tc.Format
		if format == "" {
			format = "txt"
		}
		switch format {
		case "xml":
			AssertValidXML(t, text, message)
		case "json":
			AssertValidJSON(t, text, message)
		}

		expectedFile := filepath.Join("test_data", "expected", objName, propName, tc.TestName+"."+format)
		require.FileExists(t, expectedFile, "for expected output")

		expected, err := ioutil.ReadFile(expectedFile)
		require.NoError(t, err, "failed to load %s", expectedFile)

		if assert.Equal(t, string(expected), text, "generated template for %s", message) {
			t.Logf("got expected file %s for %s\n", expectedFile, message)
		} else {
			t.Logf("generated for %s:\n%s\n", message, text)
		}
	}
}
