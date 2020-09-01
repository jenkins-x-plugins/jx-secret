package populate_test

import (
	"path/filepath"
	"testing"

	"github.com/jenkins-x/jx-api/pkg/config"
	"github.com/jenkins-x/jx-secret/pkg/cmd/populate/templatertesting"
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
