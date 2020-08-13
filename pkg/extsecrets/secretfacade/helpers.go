package secretfacade

import (
	v1 "github.com/jenkins-x/jx-secret/pkg/apis/external/v1"
	corev1 "k8s.io/api/core/v1"
)

// VerifySecret verifies the secret
func VerifySecret(es *v1.ExternalSecret, secret *corev1.Secret) (*SecretError, error) {
	var answer []*EntryError
	for _, d := range es.Spec.Data {
		valid := false
		if secret != nil && secret.Data != nil {
			value := secret.Data[d.Name]
			if len(value) > 0 {
				valid = true
			}
		}
		if !valid {
			var entry *EntryError

			for _, e := range answer {
				if e.Key == d.Key {
					entry = e
					break
				}
			}
			if entry == nil {
				entry = &EntryError{
					Key: d.Key,
				}
				answer = append(answer, entry)
			}
			entry.Properties = append(entry.Properties, d.Property)
		}
	}
	if len(answer) == 0 {
		return nil, nil
	}
	return &SecretError{
		ExternalSecret: *es,
		EntryErrors:    answer,
	}, nil
}
