package populate

import (
	"bytes"
	"text/template"

	"github.com/Masterminds/sprig"
	"github.com/jenkins-x/jx-api/pkg/config"
	"github.com/jenkins-x/jx-logging/pkg/log"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EvaluateTemplate evaluates the go template to create the value
func (o *Options) EvaluateTemplate(secretName, property, templateText string) (string, error) {
	funcMap := sprig.TxtFuncMap()

	// represents the helm template function
	// which can be used like: `{{ secret "name" "key" }}
	funcMap["secret"] = func(lookupSecret, lookupKey string) string {
		secret, err := o.KubeClient.CoreV1().Secrets(o.Namespace).Get(lookupSecret, metav1.GetOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			log.Logger().Warnf("failed to find secret %s in namespace %s so cannot resolve secret %s property %s from template", lookupSecret, o.Namespace, secretName, property)
			return ""
		}
		answer := ""
		if secret != nil && secret.Data != nil {
			return string(secret.Data[lookupKey])
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
