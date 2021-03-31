package secretfacade

import (
	"fmt"

	"github.com/jenkins-x-plugins/secretfacade/pkg/secretstore"
	"github.com/jenkins-x-plugins/secretfacade/pkg/secretstore/factory"
	v1 "github.com/jenkins-x-plugins/jx-secret/pkg/apis/external/v1"
	schema "github.com/jenkins-x-plugins/jx-secret/pkg/apis/schema/v1alpha1"
	"github.com/jenkins-x-plugins/jx-secret/pkg/extsecrets"
	"github.com/jenkins-x-plugins/jx-secret/pkg/extsecrets/editor"
	"github.com/jenkins-x-plugins/jx-secret/pkg/schemas"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

// Options options for verifying secrets
type Options struct {
	Dir                       string
	Namespace                 string
	SecretNamespace           string
	Filter                    string
	SecretClient              extsecrets.Interface
	KubeClient                kubernetes.Interface
	Source                    string
	SecretStoreManagerFactory secretstore.FactoryInterface

	// ExternalSecrets the loaded secrets
	ExternalSecrets []*v1.ExternalSecret

	// EditorCache the optional cache of editors
	EditorCache map[string]editor.Interface
}

type ExternalSecretLocation string

const FileSystem string = "filesystem"
const Kubernetes string = "kubernetes"

func (o *Options) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&o.Filter, "filter", "f", "", "the filter to filter on ExternalSecret names")
	cmd.Flags().StringVarP(&o.Source, "source", "s", "kubernetes", "the source location for the ExternalSecrets, valid values include filesystem or kubernetes")
}

func (o *Options) Validate() error {
	var err error
	if o.SecretClient == nil && (o.Source == Kubernetes || o.Source == "") {
		o.SecretClient, err = extsecrets.NewClient(nil)
		if err != nil {
			return errors.Wrap(err, "error initialising kubernetes external secrets client")
		}
	} else if o.SecretClient == nil && o.Source == FileSystem {
		o.SecretClient = extsecrets.NewFileClient(o.Dir)
	}
	if o.SecretClient == nil {
		return fmt.Errorf("secret client required to read external secrets")
	}
	if o.SecretStoreManagerFactory == nil {
		o.SecretStoreManagerFactory = &factory.SecretManagerFactory{}
	}
	return nil
}

func (o *Options) ExternalSecretByName(secretName string) (*v1.ExternalSecret, error) {
	for _, s := range o.ExternalSecrets {
		if s.ObjectMeta.Name == secretName {
			return s, nil
		}
	}
	return nil, fmt.Errorf("unable to find External Secret with name %s", secretName)
}

// SecretError returns an error for a secret
type SecretError struct {
	// ExternalSecret the external secret which is not valid
	ExternalSecret v1.ExternalSecret

	// EntryErrors the errors for each secret entry
	EntryErrors []*EntryError
}

// EntryError represents the missing entries
type EntryError struct {
	// Key the secret key
	Key string

	// Properties property names for the key
	Properties []string
}

// SecretPair the external secret and the associated Secret an error for a secret
type SecretPair struct {
	// ExternalSecret the external secret which is not valid
	ExternalSecret v1.ExternalSecret

	// Secret the secret if there is one
	Secret *corev1.Secret

	// Error last validation error at last check
	Error *SecretError

	// schemaObject caches the schema object
	schemaObject *schema.Object
}

// IsInvalid returns true if the validation failed
func (p *SecretPair) IsInvalid() bool {
	return p.Error != nil && len(p.Error.EntryErrors) > 0
}

// IsMandatory returns true if the secret is a mandatory secret
func (p *SecretPair) IsMandatory() bool {
	obj, err := p.SchemaObject()
	if err == nil && obj != nil {
		return obj.Mandatory
	}
	return false
}

// SetSchemaObject sets the cached schema object: typically used for testing
func (p *SecretPair) SetSchemaObject(schemaObject *schema.Object) {
	p.schemaObject = schemaObject
}

// SchemaObject returns the optional schema object from the annotation
func (p *SecretPair) SchemaObject() (*schema.Object, error) {
	if p.schemaObject != nil {
		return p.schemaObject, nil
	}
	var err error
	p.schemaObject, err = schemas.ObjectFromObjectMeta(&p.ExternalSecret.ObjectMeta)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to load schema object from ExternalSecret annotation for %s", p.ExternalSecret.Name)
	}
	return p.schemaObject, nil
}

// Name returns the name of the secret
func (p *SecretPair) Name() string {
	return p.ExternalSecret.Name
}

// Namespace returns the namespace of the secret
func (p *SecretPair) Namespace() string {
	return p.ExternalSecret.Namespace
}

// Key returns the unique key of the secret
func (p *SecretPair) Key() string {
	return p.Namespace() + "/" + p.Name()
}
