package populate_test

import (
	"path/filepath"
	"testing"

	"github.com/jenkins-x-plugins/secretfacade/pkg/secretstore"
	jxcore "github.com/jenkins-x/jx-api/v4/pkg/apis/core/v4beta1"
	"github.com/jenkins-x/jx-secret/pkg/cmd/populate"
	"github.com/jenkins-x/jx-secret/pkg/cmd/populate/templatertesting"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestTemplater(t *testing.T) {
	ns := "jx"

	testSecrets := []runtime.Object{
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "jx-boot",
				Namespace: ns,
			},
			Data: map[string][]byte{
				"username": []byte("gitoperatorUsername"),
				"password": []byte("gitoperatorpassword"),
			},
		},

		// some other secrets used for templating the jenkins-maven-settings Secret
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "jx-basic-auth-user-password",
				Namespace: ns,
			},
			Data: map[string][]byte{
				"username": []byte("my-basic-auth-user"),
				"password": []byte("my-basic-auth-password"),
			},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "nexus",
				Namespace: ns,
			},
			Data: map[string][]byte{
				"password": []byte("my-nexus-password"),
			},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "sonatype",
				Namespace: ns,
			},
			Data: map[string][]byte{
				"username": []byte("my-sonatype-username"),
				"password": []byte("my-sonatype-password"),
			},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "gpg",
				Namespace: ns,
			},
			Data: map[string][]byte{
				"passphrase": []byte("my-secret-gpg-passphrase"),
			},
		},
	}

	runner := templatertesting.Runner{
		TestCases: []templatertesting.TestCase{
			{
				TestName:   "jx-basic-auth-htpasswd-external",
				ObjectName: "jx-basic-auth-htpasswd-external",
				Property:   "auth",
				VerifyFn: func(t *testing.T, text string) {
					assert.NotEmpty(t, text, "should have created a valid htpasswd value from the external secret")
					t.Logf("generated jx-basic-auth-htpasswd auth value: %s\n", text)
				},
				Requirements: &jxcore.RequirementsConfig{
					Repository: "nexus",
					Cluster: jxcore.ClusterConfig{
						Provider:    "gke",
						ProjectID:   "myproject",
						ClusterName: "mycluster",
					},
					SecretStorage: "gsm",
				},
				ExternalSecrets: []templatertesting.ExternalSecret{
					{
						Location: "myproject",
						Name:     "jx-basic-auth-user-password",
						Value: secretstore.SecretValue{
							PropertyValues: map[string]string{
								"username": "my-basic-auth-user",
								"password": "my-basic-auth-password",
							},
						},
					},
				},
				ExternalSecretStorageType: secretstore.SecretStoreTypeGoogle,
			},
			{
				TestName:   "jx-basic-auth-htpasswd",
				ObjectName: "jx-basic-auth-htpasswd",
				Property:   "auth",
				VerifyFn: func(t *testing.T, text string) {
					assert.NotEmpty(t, text, "should have created a valid htpasswd value from the Secret")
					t.Logf("generated jx-basic-auth-htpasswd auth value: %s\n", text)
				},
				Requirements: &jxcore.RequirementsConfig{
					Repository: "nexus",
					Cluster: jxcore.ClusterConfig{
						Provider:    "gke",
						ProjectID:   "myproject",
						ClusterName: "mycluster",
					},
				},
			},
			{
				TestName:   "docker-gke",
				ObjectName: "jenkins-docker-cfg",
				Property:   "config.json",
				Format:     "json",
				Requirements: &jxcore.RequirementsConfig{
					Repository: "nexus",
					Cluster: jxcore.ClusterConfig{
						Provider:    "gke",
						ProjectID:   "myproject",
						ClusterName: "mycluster",
					},
				},
			},
			{
				TestName:   "docker-gke",
				ObjectName: "jenkins-docker-cfg-external",
				Property:   "config.json",
				Format:     "json",
				Requirements: &jxcore.RequirementsConfig{
					Repository: "nexus",
					Cluster: jxcore.ClusterConfig{
						Provider:    "gke",
						ProjectID:   "myproject",
						ClusterName: "mycluster",
					},
					SecretStorage: "gsm",
				},
				ExternalSecretStorageType: secretstore.SecretStoreTypeGoogle,
				ExternalSecrets: []templatertesting.ExternalSecret{
					{
						Location: "myproject",
						Name:     "docker-hub",
						Value: secretstore.SecretValue{
							PropertyValues: map[string]string{
								"url":      "mydockerhub.com",
								"username": "dockeruser",
								"password": "dockerpassword",
								"email":    "dockeremail",
							},
						},
					},
				},
			},
			{
				TestName:   "nexus",
				ObjectName: "jenkins-maven-settings",
				Property:   "settings.xml",
				Format:     "xml",
				Requirements: &jxcore.RequirementsConfig{
					Repository: "nexus",
					Cluster: jxcore.ClusterConfig{
						Provider:    "gke",
						ProjectID:   "myproject",
						ClusterName: "mycluster",
					},
				},
			},
			{
				TestName:   "nexus",
				ObjectName: "jenkins-maven-settings-external",
				Property:   "settings.xml",
				Format:     "xml",
				Requirements: &jxcore.RequirementsConfig{
					Repository: "nexus",
					Cluster: jxcore.ClusterConfig{
						Provider:    "gke",
						ProjectID:   "myproject",
						ClusterName: "mycluster",
					},
					SecretStorage: "gsm",
				},
				ExternalSecrets: []templatertesting.ExternalSecret{
					{
						Location: "myproject",
						Name:     "nexus",
						Value: secretstore.SecretValue{
							PropertyValues: map[string]string{
								"password": "my-nexus-password",
							},
						},
					},
					{
						Location: "myproject",
						Name:     "sonatype",
						Value: secretstore.SecretValue{
							PropertyValues: map[string]string{
								"username": "my-sonatype-username",
								"password": "my-sonatype-password",
							},
						},
					},
					{
						Location: "myproject",
						Name:     "gpg",
						Value: secretstore.SecretValue{
							PropertyValues: map[string]string{
								"passphrase": "my-secret-gpg-passphrase",
							},
						},
					},
				},
				ExternalSecretStorageType: secretstore.SecretStoreTypeGoogle,
			},
			{
				TestName:   "bucketrepo",
				ObjectName: "jenkins-maven-settings",
				Property:   "settings.xml",
				Format:     "xml",
				Requirements: &jxcore.RequirementsConfig{
					Repository: "nexus",
					Cluster: jxcore.ClusterConfig{
						Provider:    "minikube",
						ProjectID:   "myproject",
						ClusterName: "mycluster",
					},
				},
			},
			{
				TestName:   "bucketrepo",
				ObjectName: "jenkins-maven-settings-external",
				Property:   "settings.xml",
				Format:     "xml",
				Requirements: &jxcore.RequirementsConfig{
					Repository: "nexus",
					Cluster: jxcore.ClusterConfig{
						Provider:    "minikube",
						ProjectID:   "myproject",
						ClusterName: "mycluster",
					},
					SecretStorage: "gsm",
				},
				ExternalSecrets: []templatertesting.ExternalSecret{
					{
						Location: "myproject",
						Name:     "nexus",
						Value: secretstore.SecretValue{
							PropertyValues: map[string]string{
								"password": "my-nexus-password",
							},
						},
					},
					{
						Location: "myproject",
						Name:     "sonatype",
						Value: secretstore.SecretValue{
							PropertyValues: map[string]string{
								"username": "my-sonatype-username",
								"password": "my-sonatype-password",
							},
						},
					},
					{
						Location: "myproject",
						Name:     "gpg",
						Value: secretstore.SecretValue{
							PropertyValues: map[string]string{
								"passphrase": "my-secret-gpg-passphrase",
							},
						},
					},
				},
				ExternalSecretStorageType: secretstore.SecretStoreTypeGoogle,
			},
			{
				TestName:   "none",
				ObjectName: "jenkins-maven-settings",
				Property:   "settings.xml",
				Format:     "xml",
				Requirements: &jxcore.RequirementsConfig{
					Repository: "nexus",
					Cluster: jxcore.ClusterConfig{
						Provider:    "docker",
						ProjectID:   "myproject",
						ClusterName: "mycluster",
					},
				},
			},
			{
				TestName:   "none",
				ObjectName: "jenkins-maven-settings-external",
				Property:   "settings.xml",
				Format:     "xml",
				Requirements: &jxcore.RequirementsConfig{
					Repository: "nexus",
					Cluster: jxcore.ClusterConfig{
						Provider:    "docker",
						ProjectID:   "myproject",
						ClusterName: "mycluster",
					},
					SecretStorage: "gsm",
				},
				ExternalSecrets: []templatertesting.ExternalSecret{
					{
						Location: "myproject",
						Name:     "nexus",
						Value: secretstore.SecretValue{
							PropertyValues: map[string]string{
								"password": "my-nexus-password",
							},
						},
					},
					{
						Location: "myproject",
						Name:     "sonatype",
						Value: secretstore.SecretValue{
							PropertyValues: map[string]string{
								"username": "my-sonatype-username",
								"password": "my-sonatype-password",
							},
						},
					},
					{
						Location: "myproject",
						Name:     "gpg",
						Value: secretstore.SecretValue{
							PropertyValues: map[string]string{
								"passphrase": "my-secret-gpg-passphrase",
							},
						},
					},
				},
				ExternalSecretStorageType: secretstore.SecretStoreTypeGoogle,
			},
		},
		SchemaFile:  filepath.Join("test_data", "template", "secret-schema.yaml"),
		Namespace:   ns,
		KubeObjects: testSecrets,
	}
	runner.Run(t)
}

func TestResolveNames(t *testing.T) {
	testCases := []struct {
		input    []string
		expected []string
	}{
		{
			input: []string{
				"jx-boot", "jx",
			},
			expected: []string{
				"jx-boot", "jx",
			},
		},
		{
			input: []string{
				"jx-git-operator.jx-boot", "jx",
			},
			expected: []string{
				"jx-boot", "jx-git-operator",
			},
		},
	}

	for _, tc := range testCases {
		name, ns := populate.ResolveResourceNames(tc.input[0], tc.input[1])

		assert.Equal(t, tc.expected[0], name, "name for input %v", tc.input)
		assert.Equal(t, tc.expected[1], ns, "namespace for input %v", tc.input)
	}
}
