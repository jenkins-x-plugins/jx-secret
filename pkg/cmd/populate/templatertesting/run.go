package templatertesting

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/jenkins-x-plugins/jx-secret/pkg/extsecrets/secretfacade"
	"github.com/jenkins-x-plugins/jx-secret/pkg/extsecrets/testsecrets"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"

	v1 "github.com/jenkins-x-plugins/jx-secret/pkg/apis/external/v1"
	"github.com/jenkins-x-plugins/jx-secret/pkg/cmd/populate"
	"github.com/jenkins-x-plugins/jx-secret/pkg/schemas"
	secretstorefake "github.com/jenkins-x-plugins/secretfacade/testing/fake"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner/fakerunner"
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

	o.Namespace = r.Namespace

	runner := &fakerunner.FakeRunner{}
	o.CommandRunner = runner.Run

	for k := range r.TestCases {
		testcase := r.TestCases[k]
		r.KubeObjects = append(r.KubeObjects, testcase.KubeObjects...)
		o.KubeClient = fake.NewSimpleClientset(testsecrets.AddVaultSecrets(r.KubeObjects...)...)

		fakeFactory := &secretstorefake.SecretManagerFactory{}
		o.SecretStoreManagerFactory = fakeFactory

		_, err = fakeFactory.NewSecretManager(testcase.ExternalSecretStorageType)
		assert.NoError(t, err)

		fakeStore := fakeFactory.GetSecretStore()

		o.Options.ExternalSecrets = []*v1.ExternalSecret{}
		for k := range testcase.ExternalSecrets {
			p := testcase.ExternalSecrets[k]
			es := p.ExternalSecret
			o.Options.ExternalSecrets = append(o.Options.ExternalSecrets, &es)
			err := fakeStore.SetSecret(p.Location, p.Name, &p.Value)
			assert.NoError(t, err)
		}

		objName := testcase.ObjectName
		if testcase.Requirements != nil {
			o.Requirements = testcase.Requirements
		}
		if o.Requirements == nil {
			if testcase.Dir != "" {
				o.Dir = testcase.Dir
			}
			if o.Dir == "" {
				o.Dir = r.Dir
			}
			require.NotEmpty(t, o.Dir, "you must either specify Requirements or a Dir on the Runner or TestCase to be able to detect the Requirements to use the the template generation")
		}
		object := schema.Spec.FindObject(objName)
		require.NotNil(t, object, "could not find schema for object name %s", objName)

		propName := testcase.Property
		property := object.FindProperty(propName)
		require.NotNil(t, property, "could not find property for object %s name %s", objName, propName)

		templateText := property.Template
		require.NotEmpty(t, templateText, "no template defined for object %s property name %s", objName, propName)

		text, err := o.EvaluateTemplate(r.Namespace, objName, propName, templateText, false)
		require.NoError(t, err, "failed to evaluate template for object %s property name %s", objName, propName)

		message := fmt.Sprintf("test %s for object %s property name %s", testcase.TestName, objName, propName)
		require.NotEmpty(t, text, message)

		format := testcase.Format
		if format == "" {
			format = "txt"
		}
		switch format {
		case "xml":
			AssertValidXML(t, text, message)
		case "json":
			AssertValidJSON(t, text, message)
		}

		expectedDir := filepath.Join("test_data", "template", "expected", objName, propName)
		expectedFile := filepath.Join(expectedDir, testcase.TestName+"."+format)

		generateTestOut(t, &testcase, expectedDir, expectedFile, text, message)
	}
}

// Populate simulates the populate loop
func (r *Runner) Populate(t *testing.T) {
	require.NotEmpty(t, r.TestCases, "no TestCases supplied")
	require.NotEmpty(t, r.SchemaFile, "no SchemaFile supplied")

	schema, err := schemas.LoadSchemaFile(r.SchemaFile)
	require.NoError(t, err, "failed to load schema file %s", r.SchemaFile)

	_, o := populate.NewCmdPopulate()
	o.NoWait = true

	o.Namespace = r.Namespace

	runner := &fakerunner.FakeRunner{}
	o.CommandRunner = runner.Run

	for k := range r.TestCases {
		tc := r.TestCases[k]
		r.KubeObjects = append(r.KubeObjects, tc.KubeObjects...)
		o.KubeClient = fake.NewSimpleClientset(testsecrets.AddVaultSecrets(r.KubeObjects...)...)

		fakeFactory := &secretstorefake.SecretManagerFactory{}
		o.SecretStoreManagerFactory = fakeFactory

		_, err = fakeFactory.NewSecretManager(tc.ExternalSecretStorageType)
		assert.NoError(t, err)

		fakeStore := fakeFactory.GetSecretStore()

		o.Options.ExternalSecrets = []*v1.ExternalSecret{}
		for k := range tc.ExternalSecrets {
			p := tc.ExternalSecrets[k]
			es := p.ExternalSecret
			o.Options.ExternalSecrets = append(o.Options.ExternalSecrets, &es)
			err := fakeStore.SetSecret(p.Location, p.Name, &p.Value)
			assert.NoError(t, err)
		}

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

		require.NotEmpty(t, tc.ExternalSecrets, "should have at least 1 ExternalSecret")
		secret := tc.Secret
		require.NotNil(t, secret, "requires a Secret resource for the populate loop test")

		p := tc.ExternalSecrets[0]

		secretPair := &secretfacade.SecretPair{
			ExternalSecret: p.ExternalSecret,
			Secret:         secret,
		}
		secretPair.SetSchemaObject(object)
		results := []*secretfacade.SecretPair{secretPair}
		wait := map[string]bool{}
		err := o.PopulateLoop(results, wait)
		require.NoError(t, err, "populating loop for test %s", tc.TestName)

		propName := tc.Property
		property := object.FindProperty(propName)
		require.NotNil(t, property, "could not find property for object %s name %s", objName, propName)

		text, err := fakeStore.GetSecret(p.Location, p.Name, tc.Property)
		require.NoError(t, err, "failed to get secret value for %s", tc.TestName)

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

		expectedDir := filepath.Join("test_data", "template", "expected", objName, propName)
		expectedFile := filepath.Join(expectedDir, tc.TestName+"."+format)

		generateTestOut(t, &tc, expectedDir, expectedFile, text, message)
	}
}

func generateTestOut(t *testing.T, tc *TestCase, expectedDir, expectedFile, text, message string) {
	var err error
	if tc.GenerateTestOutput {
		err = os.MkdirAll(expectedDir, files.DefaultDirWritePermissions)
		require.NoError(t, err, "failed to create dir %s", expectedDir)

		err = os.WriteFile(expectedFile, []byte(text), files.DefaultFileWritePermissions)
		require.NoError(t, err, "failed to save file %s", expectedFile)

		t.Logf("generated %s\n", expectedFile)

	} else {
		require.FileExists(t, expectedFile, "for expected output")

		expected, err := os.ReadFile(expectedFile)
		require.NoError(t, err, "failed to load %s", expectedFile)

		if tc.VerifyFn != nil {
			tc.VerifyFn(t, text)
		} else if assert.Equal(t, string(expected), text, "generated template for %s", message) {
			t.Logf("got expected file %s for %s\n", expectedFile, message)
		} else {
			t.Logf("generated for %s:\n%s\n", message, text)
		}
	}
}
