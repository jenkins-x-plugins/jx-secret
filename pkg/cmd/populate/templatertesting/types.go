package templatertesting

import (
	"github.com/jenkins-x/jx-api/v3/pkg/config"
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

// TestCase represents a test case
type TestCase struct {
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
	Requirements *config.RequirementsConfig
}
