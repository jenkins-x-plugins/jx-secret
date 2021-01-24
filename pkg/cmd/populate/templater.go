package populate

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/chrismellard/secretfacade/pkg/secretstore"
	jxcore "github.com/jenkins-x/jx-api/v4/pkg/apis/core/v4beta1"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/pkg/errors"
	"golang.org/x/crypto/bcrypt"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
)

// EvaluateTemplate evaluates the go template to create the value
func (o *Options) EvaluateTemplate(namespace, secretName, property, templateText string, retryTemplate bool) (string, error) {
	funcMap := sprig.TxtFuncMap()
	backOff := o.Backoff
	if backOff == nil {
		backOff = &retry.DefaultBackoff
	}

	if o.Requirements == nil {
		var requirementsResource *jxcore.Requirements
		requirementsResource, _, err := jxcore.LoadRequirementsConfig(o.Dir, false)
		if err != nil {
			return "", errors.Wrapf(err, "failed to load jx-requirements.yml in dir %s", o.Dir)
		}
		o.Requirements = &requirementsResource.Spec
	}

	// template function to lookup a value in a secret:
	//
	// use like this: `{{ secret "my-secret-name" "key-name" }}
	funcMap["extsecret"] = func(lookupSecretName, lookupKey string) string {
		var secret string
		lookupSecret, ns := ResolveResourceNames(lookupSecretName, namespace)
		storeType, err := storeTypeFromRequirements(o.Requirements)

		if err != nil {
			log.Logger().Warn("failed to determine store type from requirements")
			return ""
		}
		secretManager, err := o.SecretStoreManagerFactory.NewSecretManager(storeType)

		getSecretFunc := func() error {
			var err error
			secretLocation, err := storeLocationFromRequirements(o.Requirements, ns)
			if err != nil {
				return err
			}
			secret, err = secretManager.GetSecret(secretLocation, lookupSecret, lookupKey)
			return err
		}

		if retryTemplate {
			err = retry.OnError(*backOff, func(err error) bool {
				f := apierrors.IsNotFound(err)
				if f {
					log.Logger().Warnf("secret %s in namespace %s cannot be found - backoff loop", lookupSecret, ns)
				}
				return f
			}, getSecretFunc)
		} else {
			err = getSecretFunc()
		}
		if err != nil && !apierrors.IsNotFound(err) {
			log.Logger().Warnf("failed to find secret %s in namespace %s so cannot resolve secret %s property %s from template", lookupSecret, ns, secretName, property)
			return ""
		}
		answer := ""
		if secret != "" {
			return secret
		}
		return answer
	}

	funcMap["secret"] = func(lookupSecretName, lookupKey string) string {
		var secret *v1.Secret
		lookupSecret, ns := ResolveResourceNames(lookupSecretName, namespace)

		getSecretFunc := func() error {
			var err error
			secret, err = o.KubeClient.CoreV1().Secrets(ns).Get(context.TODO(), lookupSecret, metav1.GetOptions{})
			return err
		}

		var err error
		if retryTemplate {
			err = retry.OnError(*backOff, func(err error) bool {
				f := apierrors.IsNotFound(err)
				if f {
					log.Logger().Warnf("secret %s in namespace %s cannot be found - f backoff loop", lookupSecret, ns)
				}
				return f
			}, getSecretFunc)
		} else {
			err = getSecretFunc()
		}
		if err != nil && !apierrors.IsNotFound(err) {
			log.Logger().Warnf("failed to find secret %s in namespace %s so cannot resolve secret %s property %s from template", lookupSecret, ns, secretName, property)
			return ""
		}
		answer := ""
		if secret != nil && secret.Data != nil {
			return string(secret.Data[lookupKey])
		}
		return answer
	}

	// template function to lookup username + password in a Secret and then use that to make a htpasswd value
	//
	// use like this: `{{ htpasswdSecret "my-secret-name" "username" "password" }}
	funcMap["htpasswdSecret"] = func(lookupSecretName, usernameKey, passwordKey string) string {
		var secret *v1.Secret
		lookupSecret, ns := ResolveResourceNames(lookupSecretName, namespace)

		getSecretFunc := func() error {
			var err error
			secret, err = o.KubeClient.CoreV1().Secrets(ns).Get(context.TODO(), lookupSecret, metav1.GetOptions{})
			return err
		}

		var err error
		if retryTemplate {
			err = retry.OnError(*backOff, func(err error) bool {
				retry := apierrors.IsNotFound(err)
				if retry {
					log.Logger().Warnf("secret %s in namespace %s cannot be found - retry backoff loop", lookupSecret, ns)
				}
				return retry
			}, getSecretFunc)
		} else {
			err = getSecretFunc()
		}
		if err != nil && !apierrors.IsNotFound(err) {
			log.Logger().Warnf("failed to find secret %s in namespace %s so cannot resolve secret %s property %s from template", lookupSecret, ns, secretName, property)
			return ""
		}
		if secret == nil || secret.Data == nil {
			log.Logger().Warnf("failed to create htpasswd: no secret %s for namespace %s", lookupSecret, ns)
			return ""
		}
		username := string(secret.Data[usernameKey])
		if username == "" {
			log.Logger().Warnf("failed to create htpasswd: secret %s does not have username entry %s in namespace %s", lookupSecret, usernameKey, ns)
			return ""
		}
		if strings.Contains(username, ":") {
			log.Logger().Warnf("invalid username: %s from secret %s in namespace %s", username, lookupSecret, ns)
			return ""
		}

		password := string(secret.Data[passwordKey])
		if password == "" {
			log.Logger().Warnf("failed to create htpasswd: secret %s does not have password entry %s in namespace %s", lookupSecret, passwordKey, ns)
			return ""
		}

		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			log.Logger().Warnf("failed to create htpasswd: from secret %s in namespace %s: %s", lookupSecret, ns, err.Error())
			return ""
		}
		return fmt.Sprintf("%s:%s", username, hash)
	}

	funcMap["htpasswdExtSecret"] = func(lookupSecretName, usernameKey, passwordKey string) string {
		var usernameSecret string
		var passwordSecret string
		lookupSecret, ns := ResolveResourceNames(lookupSecretName, namespace)

		storeType, err := storeTypeFromRequirements(o.Requirements)

		if err != nil {
			log.Logger().Warn("failed to determine store type from requirements")
			return ""
		}
		secretManager, err := o.SecretStoreManagerFactory.NewSecretManager(storeType)

		getSecretFunc := func() error {
			var err error
			secretLocation, err := storeLocationFromRequirements(o.Requirements, ns)
			if err != nil {
				return err
			}
			usernameSecret, err = secretManager.GetSecret(secretLocation, lookupSecret, usernameKey)
			if err != nil {
				return err
			}
			passwordSecret, err = secretManager.GetSecret(secretLocation, lookupSecret, passwordKey)
			return err
		}

		if retryTemplate {
			err = retry.OnError(*backOff, func(err error) bool {
				retry := apierrors.IsNotFound(err)
				if retry {
					log.Logger().Warnf("secret %s in namespace %s cannot be found - retry backoff loop", lookupSecret, ns)
				}
				return retry
			}, getSecretFunc)
		} else {
			err = getSecretFunc()
		}
		if err != nil && !apierrors.IsNotFound(err) {
			log.Logger().Warnf("failed to find secret %s in namespace %s so cannot resolve secret %s property %s from template", lookupSecret, ns, secretName, property)
			return ""
		}
		username := usernameSecret
		if username == "" {
			log.Logger().Warnf("failed to create htpasswd: secret %s does not have username entry %s in namespace %s", lookupSecret, usernameKey, ns)
			return ""
		}
		if strings.Contains(username, ":") {
			log.Logger().Warnf("invalid username: %s from secret %s in namespace %s", username, lookupSecret, ns)
			return ""
		}

		password := passwordSecret
		if password == "" {
			log.Logger().Warnf("failed to create htpasswd: secret %s does not have password entry %s in namespace %s", lookupSecret, passwordKey, ns)
			return ""
		}

		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			log.Logger().Warnf("failed to create htpasswd: from secret %s in namespace %s: %s", lookupSecret, ns, err.Error())
			return ""
		}
		return fmt.Sprintf("%s:%s", username, hash)
	}

	// template function to lookup a user + password in a secret and concatenate in a string like `"username:password"`.
	//
	// use like this: `{{ auth "my-secret-name" "username-key" "password-key }}
	funcMap["auth"] = func(lookupSecretName, userKey, passwordKey string) string {
		var secret *v1.Secret
		lookupSecret, ns := ResolveResourceNames(lookupSecretName, namespace)

		getSecretFunc := func() error {
			var err error
			secret, err = o.KubeClient.CoreV1().Secrets(ns).Get(context.TODO(), lookupSecret, metav1.GetOptions{})
			return err
		}

		var err error
		if retryTemplate {
			err = retry.OnError(*backOff, func(err error) bool {
				f := apierrors.IsNotFound(err)
				if f {
					log.Logger().Warnf("secret %s in namespace %s cannot be found - f backoff loop", lookupSecret, ns)
				}
				return f
			}, getSecretFunc)
		} else {
			err = getSecretFunc()
		}
		if err != nil && !apierrors.IsNotFound(err) {
			log.Logger().Warnf("failed to find secret %s in namespace %s so cannot resolve secret %s property %s from template", lookupSecret, ns, secretName, property)
			return ""
		}
		answer := ""
		if secret != nil && secret.Data != nil {
			return string(secret.Data[userKey]) + ":" + string(secret.Data[passwordKey])
		}
		return answer
	}

	funcMap["extauth"] = func(lookupSecretName, userKey, passwordKey string) string {
		var usernameSecret string
		var passwordSecret string
		lookupSecret, ns := ResolveResourceNames(lookupSecretName, namespace)
		storeType, err := storeTypeFromRequirements(o.Requirements)

		if err != nil {
			log.Logger().Warn("failed to determine store type from requirements")
			return ""
		}
		secretManager, err := o.SecretStoreManagerFactory.NewSecretManager(storeType)

		getSecretFunc := func() error {
			var err error
			secretLocation, err := storeLocationFromRequirements(o.Requirements, ns)
			if err != nil {
				return err
			}
			usernameSecret, err = secretManager.GetSecret(secretLocation, lookupSecret, userKey)
			if err != nil {
				return err
			}
			passwordSecret, err = secretManager.GetSecret(secretLocation, lookupSecret, passwordKey)
			return err
		}

		if retryTemplate {
			err = retry.OnError(*backOff, func(err error) bool {
				f := apierrors.IsNotFound(err)
				if f {
					log.Logger().Warnf("secret %s in namespace %s cannot be found - f backoff loop", lookupSecret, ns)
				}
				return f
			}, getSecretFunc)
		} else {
			err = getSecretFunc()
		}
		if err != nil && !apierrors.IsNotFound(err) {
			log.Logger().Warnf("failed to find secret %s in namespace %s so cannot resolve secret %s property %s from template", lookupSecret, ns, secretName, property)
			return ""
		}

		return usernameSecret + ":" + passwordSecret
	}

	tmpl, err := template.New("value.gotmpl").Option("missingkey=error").Funcs(funcMap).Parse(templateText)
	if err != nil {
		return "", errors.Wrapf(err, "failed to parse Secret %s property %s with template: %s", secretName, property, templateText)
	}

	requirementsMap, err := o.Requirements.ToMap()
	if err != nil {
		return "", errors.Wrapf(err, "failed turn requirements into a map: %v", o.Requirements)
	}
	if requirementsMap["storage"] == nil {
		requirementsMap["storage"] = map[string]string{}
	}
	templateData := map[string]interface{}{
		"Requirements": requirementsMap,
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, templateData)
	if err != nil {
		return "", errors.Wrapf(err, "failed to evaluate template to create value of Secret %s property %s", secretName, property)
	}
	return buf.String(), nil
}

// ResolveResourceNames if the secret name contains a dot then assume its namespace.name otherwise return the name in the current namespace
func ResolveResourceNames(name string, currentNamespace string) (string, string) {
	idx := strings.Index(name, ".")
	if idx < 0 {
		return name, currentNamespace
	}
	return name[idx+1:], name[:idx]
}

func storeTypeFromRequirements(requirements *jxcore.RequirementsConfig) (secretstore.SecretStoreType, error) {
	switch requirements.SecretStorage {
	case "azure":
		return secretstore.SecretStoreTypeAzure, nil
	case "vault":
		return secretstore.SecretStoreTypeVault, nil
	case "gsm":
		return secretstore.SecretStoreTypeGoogle, nil
	case "local":
		return secretstore.SecretStoreTypeKubernetes, nil
	}
	return "", fmt.Errorf("unable to determine secret store type from %s", requirements.SecretStorage)
}

func storeLocationFromRequirements(requirements *jxcore.RequirementsConfig, namespace string) (string, error) {
	switch requirements.SecretStorage {
	case "azure":
		if requirements.Cluster.AzureConfig != nil && requirements.Cluster.AzureConfig.AzureSecretStorageConfig != nil {
			return requirements.Cluster.AzureConfig.AzureSecretStorageConfig.KeyVaultName, nil
		}
		return "", fmt.Errorf("unable to determine azure key vault name from requirements")
	case "vault":
		return os.Getenv("VAULT_ADDR"), nil
	case "gsm":
		return requirements.Cluster.ProjectID, nil
	case "local":
		return namespace, nil
	}
	return "", fmt.Errorf("unable to determine secret store type from %s", requirements.SecretStorage)
}
