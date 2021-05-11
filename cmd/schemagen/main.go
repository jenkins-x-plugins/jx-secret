package main

import (
	v1 "github.com/jenkins-x-plugins/jx-secret/pkg/apis/external/v1"
	"github.com/jenkins-x-plugins/jx-secret/pkg/apis/mapping/v1alpha1"
	schemav1alpha1 "github.com/jenkins-x-plugins/jx-secret/pkg/apis/schema/v1alpha1"
	"github.com/jenkins-x/jx-api/v4/pkg/schemagen"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"os"
)

var (
	resourceKinds = []schemagen.ResourceKind{
		{
			APIVersion: "kubernetes-client.io/v1",
			Name:       "externalsecrets",
			Resource:   &v1.ExternalSecret{},
		},
		{
			APIVersion: "secret.jenkins-x.io/v1alpha1",
			Name:       "secretmappings",
			Resource:   &v1alpha1.SecretMapping{},
		},
		{
			APIVersion: "secret.jenkins-x.io/v1alpha1",
			Name:       "schemas",
			Resource:   &schemav1alpha1.Schema{},
		},
	}
)

func main() {
	out := "schema"
	if len(os.Args) > 1 {
		out = os.Args[1]
	}
	err := schemagen.GenerateSchemas(resourceKinds, out)
	if err != nil {
		log.Logger().Errorf("failed: %v", err)
		os.Exit(1)
	}
	log.Logger().Infof("completed the plugin generator")
	os.Exit(0)
}
