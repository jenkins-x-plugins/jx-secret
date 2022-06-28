package schemas

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/jenkins-x-plugins/jx-secret/pkg/apis/schema/v1alpha1"
	"github.com/jenkins-x-plugins/jx-secret/pkg/extsecrets"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"gopkg.in/validator.v2"

	"gopkg.in/yaml.v1"
)

// LoadSchemaFile loads a specific secret mapping YAML file
func LoadSchemaFile(fileName string) (*v1alpha1.Schema, error) {
	config := &v1alpha1.Schema{}

	exists, err := files.FileExists(fileName)
	if err != nil {
		return config, errors.Wrapf(err, "failed to check file exists %s", fileName)
	}
	if !exists {
		log.Logger().Warnf("no schema file found at %s", fileName)
		return config, nil
	}

	data, err := os.ReadFile(fileName)
	if err != nil {
		return nil, fmt.Errorf("failed to load file %s due to %s", fileName, err)
	}

	err = yaml.Unmarshal(data, config)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to %s unmarshal YAML file", fileName)
	}
	err = validator.Validate(config)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to validate secret mapping YAML file %s", fileName)
	}
	return config, nil
}

// LoadSchemaObjectFromFiles loads a list of schema files and finds the schema object for the given name
func LoadSchemaObjectFromFiles(name string, fileNames []string) (*v1alpha1.Object, error) {
	var answer *v1alpha1.Object
	for _, f := range fileNames {
		schema, err := LoadSchemaFile(f)
		if err != nil {
			return answer, err
		}
		object := schema.Spec.FindObject(name)
		if object == nil {
			continue
		}
		answer = object
	}
	return answer, nil
}

// MergeSchemas merges values from the source into the destination

// ToAnnotationString converts the schema object to YAML so we can store it as an annotation
func ToAnnotationString(s interface{}) (string, error) {
	data, err := json.Marshal(s)
	if err != nil {
		return "", errors.Wrapf(err, "failed to marshal object %v to JSON", s)
	}
	return string(data), nil
}

// ObjectFromObjectMeta returns the schema object for the given object metadata
func ObjectFromObjectMeta(m *metav1.ObjectMeta) (*v1alpha1.Object, error) {
	if m == nil {
		return nil, nil
	}
	ann := m.Annotations
	if ann == nil {
		return nil, nil
	}
	text := ann[extsecrets.SchemaObjectAnnotation]
	if text == "" {
		return nil, nil
	}
	return ObjectFromAnnotationString(text)
}

// ObjectFromAnnotationString converts the string to a schema object
func ObjectFromAnnotationString(text string) (*v1alpha1.Object, error) {
	if text == "" {
		return nil, nil
	}
	object := &v1alpha1.Object{}
	err := json.Unmarshal([]byte(text), object)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal JSON %s", text)
	}
	return object, nil
}

// FindObjectProperty finds the schema property for the given object
func FindObjectProperty(s *v1alpha1.Schema, objectName, property string) (*v1alpha1.Object, *v1alpha1.Property) {
	o := s.Spec.FindObject(objectName)
	p := o.FindProperty(property)
	return o, p
}
