package populate

import (
	"bytes"
	"context"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/jenkins-x/jx-api/v3/pkg/config"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EvaluateTemplate evaluates the go template to create the value
func (o *Options) EvaluateTemplate(ns, secretName, property, templateText string) (string, error) {
	funcMap := sprig.TxtFuncMap()

	// template function to lookup a value in a secret:
	//
	// use like this: `{{ secret "my-secret-name" "key-name" }}
	funcMap["secret"] = func(lookupSecret, lookupKey string) string {
		secret, err := o.KubeClient.CoreV1().Secrets(ns).Get(context.TODO(), lookupSecret, metav1.GetOptions{})
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

	// template function to lookup a user + password in a secret and concatenate in a string like `"username:password"`.
	//
	// use like this: `{{ auth "my-secret-name" "username-key" "password-key }}
	funcMap["auth"] = func(lookupSecret, userKey, passwordKey string) string {
		secret, err := o.KubeClient.CoreV1().Secrets(ns).Get(context.TODO(), lookupSecret, metav1.GetOptions{})
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

	tmpl, err := template.New("value.gotmpl").Option("missingkey=error").Funcs(funcMap).Parse(templateText)
	if err != nil {
		return "", errors.Wrapf(err, "failed to parse Secret %s property %s with template: %s", secretName, property, templateText)
	}

	if o.Requirements == nil {
		o.Requirements, _, err = config.LoadRequirementsConfig(o.Dir, false)
		if err != nil {
			return "", errors.Wrapf(err, "failed to load jx-requirements.yml in dir %s", o.Dir)
		}
	}
	requirementsMap, err := o.Requirements.ToMap()
	if err != nil {
		return "", errors.Wrapf(err, "failed turn requirements into a map: %v", o.Requirements)
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
