package secretfacade

import (
	"sort"

	"github.com/jenkins-x-plugins/jx-secret/pkg/apis/schema/v1alpha1"
	"github.com/jenkins-x-plugins/jx-secret/pkg/secretmapping"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/pkg/errors"
)

// VerifyAndFilter loads the secrets and verifies which are valid to aid the edit/populate operations
// then filters out any duplicate entries which are using the same underlying secret mappings.
//
// e.g. if 2 secrets are populated to the same actual location then we can omit one of them since there's no need
// to write to the same location twice.
//
// We prefer the secrets which have schemas associated and that have the most entries.
func (o *Options) VerifyAndFilter() ([]*SecretPair, error) {
	secrets, err := o.Verify()
	if err != nil {
		return secrets, err
	}

	// lets filter out any secrets with similar secret mapping locations...
	secretMapping, _, err := secretmapping.LoadSecretMapping(o.Dir, false)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to load secret mappings in dir %s", o.Dir)
	}

	destinations := map[string][]*SecretPair{}

	if secretMapping != nil {
		// lets iterate through all objects + keep track of the properties and locations

		for _, s := range secrets {
			es := s.ExternalSecret
			_, err := s.SchemaObject()
			if err != nil {
				return secrets, errors.Wrapf(err, "failed to load the schema object for %s", es.Name)
			}

			secretRule := secretMapping.FindRule(es.Namespace, es.Name)
			if secretRule != nil {
				for i := range secretRule.Mappings {
					mapping := &secretRule.Mappings[i]
					destination := secretMapping.DestinationString(secretRule, mapping)

					destinations[destination] = append(destinations[destination], s)
				}
			}

		}
	}

	filterKeys := map[string]bool{}
	for destination, secrets := range destinations {
		if len(secrets) < 2 {
			continue
		}

		// lets filter out any unnecessary secrets
		SortSecretsInSchemaOrder(secrets)
		for i := 1; i < len(secrets); i++ {
			key := secrets[i].Key()
			if !filterKeys[key] {
				log.Logger().Debugf("filtering out Secret %s as %s is better for schema editing and it uses the same destination %s", secrets[i].Name(), secrets[0].Name(), destination)
				filterKeys[key] = true
			}
		}
	}

	// Sort schemas so those with templates appear last as they potentially operate on non templated secrets
	SortSecretsInSchemaTemplateOrder(secrets)
	var answer []*SecretPair
	for _, s := range secrets {
		key := s.Key()
		if !filterKeys[key] {
			answer = append(answer, s)
		}
	}
	return answer, nil
}

type SchemaOrder []*SecretPair

func (a SchemaOrder) Len() int      { return len(a) }
func (a SchemaOrder) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a SchemaOrder) Less(i, j int) bool {
	s1 := a[i]
	s2 := a[j]

	o1, _ := s1.SchemaObject()
	o2, _ := s2.SchemaObject()

	if o1 != nil && o2 == nil {
		return true
	}
	if o2 != nil && o1 == nil {
		return false
	}
	if o1 != nil && o2 != nil {
		if len(o1.Properties) > len(o2.Properties) {
			return true
		}
	}
	if len(s1.ExternalSecret.Spec.Data) > len(s2.ExternalSecret.Spec.Data) {
		return true
	}
	return s1.ExternalSecret.Name < s2.ExternalSecret.Name
}

// SortSecretsInSchemaOrder sorts the secrets in schema order with the entry with a schema with the most properties being first
func SortSecretsInSchemaOrder(resources []*SecretPair) {
	sort.Sort(SchemaOrder(resources))
}

type SchemaTemplateOrder []*SecretPair

func (a SchemaTemplateOrder) Len() int      { return len(a) }
func (a SchemaTemplateOrder) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a SchemaTemplateOrder) Less(i, j int) bool {
	s1 := a[i]
	s2 := a[j]

	o1, _ := s1.SchemaObject()
	o2, _ := s2.SchemaObject()

	if o1 != nil && o2 == nil {
		return true
	}
	if o2 != nil && o1 == nil {
		return false
	}
	if o1 != nil && o2 != nil {
		o1ContainsTemplate := containsTemplate(o1.Properties)
		o2ContainsTemplate := containsTemplate(o2.Properties)
		if !o1ContainsTemplate && o2ContainsTemplate {
			return true
		}
		if o1ContainsTemplate && !o2ContainsTemplate {
			return false
		}
		if len(o1.Properties) > len(o2.Properties) {
			return true
		}
	}
	if len(s1.ExternalSecret.Spec.Data) > len(s2.ExternalSecret.Spec.Data) {
		return true
	}
	return s1.ExternalSecret.Name < s2.ExternalSecret.Name
}

func containsTemplate(properties []v1alpha1.Property) bool {
	for _, p := range properties {
		if p.Template != "" {
			return true
		}
	}
	return false
}

// SortSecretsInSchemaTemplateOrder sorts the secrets in schema order with the entry without template functions being first
func SortSecretsInSchemaTemplateOrder(resources []*SecretPair) {
	sort.Sort(SchemaTemplateOrder(resources))
}
