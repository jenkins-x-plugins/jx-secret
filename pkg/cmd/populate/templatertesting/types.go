package templatertesting

import (
	corev1 "k8s.io/api/core/v1"
	"testing"

	v1 "github.com/jenkins-x-plugins/jx-secret/pkg/apis/external/v1"
	"github.com/jenkins-x-plugins/secretfacade/pkg/secretstore"
	jxcore "github.com/jenkins-x/jx-api/v4/pkg/apis/core/v4beta1"
	"k8s.io/apimachinery/pkg/runtime"
)

// Runner runs the test harness
type Runner struct {
	// TestCases the test cases to run
	TestCases []TestCase

	// SchemaFile the schema file to load for the templates
	SchemaFile string

	// Namespace optional namespace - defaults to 'jx'
	Namespace string

	// Dir the directory used to detect the jx-requirements.yml file if none is supplied on the test case
	Dir string

	// KubeObjects so you can define default secrets
	KubeObjects []runtime.Object
}

type ExternalSecret struct {
	Location       string
	Name           string
	Value          secretstore.SecretValue
	ExternalSecret v1.ExternalSecret
}

// TestCase represents a test case
type TestCase struct {
	// GenerateTestOutput to regenerate the expected output
	GenerateTestOutput bool

	// ObjectName name of the object in the schema
	ObjectName string

	// Property name to render; this name is used to find the template in the schema
	Property string

	// TestName is the name of the test
	TestName string

	// ExpectedFile expected file name to compare the generated results against
	ExpectedFile string

	// Format the text format of the output used to perform additional validation
	Format string

	// Dir the directory used to detect the jx-requirements.yml file if none is supplied
	Dir string

	// Requirements the jx-requirements.yml used to parameterize the template
	Requirements *jxcore.RequirementsConfig

	// KubeObjects extra kubernetes resources such as Secrets for the test case
	KubeObjects []runtime.Object

	// VerifyFn performs a custom verify of the generated value
	VerifyFn func(*testing.T, string)

	ExternalSecrets []ExternalSecret

	ExternalSecretStorageType secretstore.SecretStoreType

	// Secret is the underlying secret for the first external secret if using Populate loop testing
	Secret *corev1.Secret
}
