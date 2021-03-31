package testsecrets

import (
	"github.com/jenkins-x-plugins/jx-secret/pkg/vaults"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func AddVaultSecrets(objects ...runtime.Object) []runtime.Object {
	ns := vaults.DefaultVaultNamespace
	return append(objects,
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "vault-unseal-keys",
				Namespace: ns,
			},
			Data: map[string][]byte{
				"vault-root": []byte("dummyVaultToken"),
			},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "vault-tls",
				Namespace: ns,
			},
			Data: map[string][]byte{
				"ca.crt": []byte("dummyVaultCaCert"),
			},
		})
}
