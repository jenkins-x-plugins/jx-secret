package yamls

import (
	"io/ioutil"

	"github.com/jenkins-x/jx/v2/pkg/util"
	"sigs.k8s.io/yaml"

	"github.com/pkg/errors"
)

// LoadYAML loads the given YAML file
func LoadYAML(fileName string, dest interface{}) error {
	exists, err := util.FileExists(fileName)
	if err != nil {
		return errors.Wrapf(err, "failed to check if file exists  %s", fileName)
	}
	if !exists {
		return nil
	}

	data, err := ioutil.ReadFile(fileName)
	if err != nil {
		return errors.Wrapf(err, "failed to read file %s", fileName)
	}

	err = yaml.Unmarshal(data, dest)
	if err != nil {
		return errors.Wrapf(err, "failed to unmarshal file %s", fileName)
	}
	return nil
}

// SaveYAML saves the object as the given file name
func SaveYAML(obj interface{}, fileName string) error {
	data, err := yaml.Marshal(obj)
	if err != nil {
		return errors.Wrap(err, "failed to marshal to YAML")
	}
	err = ioutil.WriteFile(fileName, data, util.DefaultWritePermissions)
	if err != nil {
		return errors.Wrapf(err, "failed to save file %s", fileName)
	}
	return nil
}
