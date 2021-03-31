package templatertesting

import (
	"encoding/json"
	"encoding/xml"
	"io"
	"strings"
	"testing"

	"github.com/jenkins-x-plugins/jx-secret/pkg/apis/schema/v1alpha1"
	"github.com/jenkins-x-plugins/jx-secret/pkg/extsecrets"
	"github.com/jenkins-x-plugins/jx-secret/pkg/schemas"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

// AssertValidXML asserts that the given text is valid XML
func AssertValidXML(t *testing.T, text, message string) {
	require.NotEmpty(t, text, message)

	decoder := xml.NewDecoder(strings.NewReader(text))
	for {
		err := decoder.Decode(new(interface{}))
		if err != nil && err == io.EOF {
			return
		}
		if err != nil {
			t.Logf("failed to parse XML: %s\n", text)
		}
		require.NoError(t, err, "failed to parse XML for %s", message)
	}
}

// AssertValidJSON asserts that the given text is valid JSON
func AssertValidJSON(t *testing.T, text, message string) {
	var j map[string]interface{}
	err := json.Unmarshal([]byte(text), &j)
	require.NoError(t, err, "failed to parse JSON for %s", message)
}

// AddSchemaAnnotations adds the schema annotation to the external secret Unstructured objects
func AddSchemaAnnotations(t *testing.T, schema *v1alpha1.Schema, dynObjects []runtime.Object) error {
	var err error
	for _, r := range dynObjects {
		u, ok := r.(*unstructured.Unstructured)
		if ok && u != nil {
			name := u.GetName()
			obj := schema.Spec.FindObject(name)
			if obj != nil {
				ann := u.GetAnnotations()
				if ann == nil {
					ann = map[string]string{}
				}
				value := ""
				value, err = schemas.ToAnnotationString(obj)
				require.NoError(t, err, "failed to create annotation value for schema %#v on secret %s", obj, name)
				ann[extsecrets.SchemaObjectAnnotation] = value
				u.SetAnnotations(ann)
			}
		}
	}
	return err
}
