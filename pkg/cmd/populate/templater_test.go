package populate_test

import (
	"path/filepath"
	"testing"

	"github.com/jenkins-x/jx-api/v3/pkg/config"
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
				TestName:   "jx-basic-auth-htpasswd",
				ObjectName: "jx-basic-auth-htpasswd",
				Property:   "auth",
				VerifyFn: func(t *testing.T, text string) {
					assert.NotEmpty(t, text, "should have created a valid htpasswd value from the Secret")
					t.Logf("generated jx-basic-auth-htpasswd auth value: %s\n", text)
				},
				Requirements: &config.RequirementsConfig{
					Repository: "nexus",
					Cluster: config.ClusterConfig{
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
				Requirements: &config.RequirementsConfig{
					Repository: "nexus",
					Cluster: config.ClusterConfig{
						Provider:    "gke",
						ProjectID:   "myproject",
						ClusterName: "mycluster",
					},
				},
			},
			{
				TestName:   "nexus",
				ObjectName: "jenkins-maven-settings",
				Property:   "settings.xml",
				Format:     "xml",
				Requirements: &config.RequirementsConfig{
					Repository: "nexus",
					Cluster: config.ClusterConfig{
						Provider:    "gke",
						ProjectID:   "myproject",
						ClusterName: "mycluster",
					},
				},
			},
			{
				TestName:   "bucketrepo",
				ObjectName: "jenkins-maven-settings",
				Property:   "settings.xml",
				Format:     "xml",
				Requirements: &config.RequirementsConfig{
					Repository: "nexus",
					Cluster: config.ClusterConfig{
						Provider:    "minikube",
						ProjectID:   "myproject",
						ClusterName: "mycluster",
					},
				},
			},
			{
				TestName:   "none",
				ObjectName: "jenkins-maven-settings",
				Property:   "settings.xml",
				Format:     "xml",
				Requirements: &config.RequirementsConfig{
					Repository: "nexus",
					Cluster: config.ClusterConfig{
						Provider:    "docker",
						ProjectID:   "myproject",
						ClusterName: "mycluster",
					},
				},
			},
		},
		SchemaFile:  filepath.Join("test_data", "secret-schema.yaml"),
		Namespace:   ns,
		KubeObjects: testSecrets,
	}
	runner.Run(t)
}


func TestResolveNames(t *testing.T) {
	testCases := []struct {
		input []string
		expected []string

	} {
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
