package schema

import (
	"fmt"
	"io/ioutil"

	"github.com/jenkins-x/jx-helpers/pkg/files"
	"github.com/jenkins-x/jx-logging/pkg/log"
	"github.com/jenkins-x/jx-secret/pkg/apis/schema/v1alpha1"
	"github.com/pkg/errors"

	"gopkg.in/validator.v2"

	"gopkg.in/yaml.v1"
)

// LoadSchema loads a specific secret mapping YAML file
func LoadSchema(fileName string) (*v1alpha1.Schema, error) {
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
		return nil, fmt.Errorf("failed to unmarshal YAML file %s due to %s", fileName, err)
	}

	err = validator.Validate(config)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to validate secret mapping YAML file %s", fileName)
	}
	return config, nil
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
