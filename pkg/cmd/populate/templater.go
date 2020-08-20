package populate

import (
	"bytes"
	"text/template"

	"github.com/Masterminds/sprig"
	"github.com/jenkins-x/jx-logging/pkg/log"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// evaluateTemplate evaluates the go template to create the value
func (o *Options) evaluateTemplate(secretName string, property string, templateText string) (string, error) {
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
	templateData := map[string]interface{}{}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, templateData)
	if err != nil {
		return "", errors.Wrapf(err, "failed to evaluate template to create value of Secret %s property %s", secretName, property)
	}
	return string(buf.Bytes()), nil
}
