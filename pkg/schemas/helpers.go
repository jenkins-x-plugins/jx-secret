package schemas

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/jenkins-x/jx-helpers/pkg/files"
	"github.com/jenkins-x/jx-logging/pkg/log"
	"github.com/jenkins-x/jx-secret/pkg/apis/schema/v1alpha1"
	"github.com/pkg/errors"

	"gopkg.in/validator.v2"

	"gopkg.in/yaml.v1"
)

// LoadSchema loads the schema file(s) from the given directory
func LoadSchema(dir string) (*v1alpha1.Schema, error) {
	absolute, err := filepath.Abs(dir)
	if err != nil {
		return nil, errors.Wrap(err, "creating absolute path")
	}

	fileName := filepath.Join(absolute, ".jx", "secret", "schema", "secret-schema.yaml")
	return LoadSchemaFile(fileName)
}

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

	data, err := ioutil.ReadFile(fileName)
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

// LoadSchemaFiles loads a list of schema files returning an error if they conflict
func LoadSchemaFiles(fileNames []string) (*v1alpha1.Schema, error) {
	answer := &v1alpha1.Schema{}
	for _, f := range fileNames {
		schema, err := LoadSchemaFile(f)
		if err != nil {
			return answer, err
		}
		err = MergeSchemas(schema, answer, f, true)
		if err != nil {
			return answer, errors.Wrapf(err, "failed to merge schema from file %s", f)
		}
	}
	return answer, nil
}

// LoadSchemaObjectFromFiles loads a list of schema files and finds the schema object for the given name
func LoadSchemaObjectFromFiles(name string, fileNames []string) (*v1alpha1.Object, error) {
	var answer *v1alpha1.Object
	firstFileName := ""
	for _, f := range fileNames {
		schema, err := LoadSchemaFile(f)
		if err != nil {
			return answer, err
		}
		object := schema.Spec.FindObject(name)
		if object == nil {
			continue
		}
		if answer == nil {
			answer = object
			firstFileName = f
			continue
		}
		return nil, errors.Errorf("duplicate object definition from file %s and %s", firstFileName, f)
	}
	return answer, nil
}

// MergeSchemas merges values from the source into the destination
func MergeSchemas(source, dest *v1alpha1.Schema, path string, failOnDuplicate bool) error {
	for i := range source.Spec.Objects {
		so := &source.Spec.Objects[i]

		found := false
		for j := range dest.Spec.Objects {
			do := &dest.Spec.Objects[j]

			if do.Name != so.Name {
				continue
			}
			if failOnDuplicate {
				return errors.Errorf("duplicate object definition %s in file %s", do.Name, path)
			}
			dest.Spec.Objects[j] = *so
			found = true
		}
		if !found {
			dest.Spec.Objects = append(dest.Spec.Objects, *so)
		}
	}
	return nil
}

// ToAnnotationString converts the schema object to YAML so we can store it as an annotation
func ToAnnotationString(s interface{}) (string, error) {
	data, err := json.Marshal(s)
	if err != nil {
		return "", errors.Wrapf(err, "failed to marshal object %v to JSON", s)
	}
	return string(data), nil
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
func FindObjectProperty(s *v1alpha1.Schema, objectName, property string) (*v1alpha1.Object, *v1alpha1.Property, error) {
	if s == nil {
		return nil, nil, nil
	}
	for i := range s.Spec.Objects {
		o := &s.Spec.Objects[i]
		if o.Name == objectName {
			for i := range o.Properties {
				p := &o.Properties[i]
				if p.Name == property {
					return o, p, nil
				}
			}
		}
	}
	return nil, nil, nil
}
