package main

import (
	"sort"
	"strings"

	"github.com/jenkins-x-plugins/jx-secret/pkg/masker"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kube"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
)

func main() {
	masker.ShowMaskedPasswords = true

	kubeClient, ns, err := kube.LazyCreateKubeClientAndNamespace(nil, "")
	if err != nil {
		log.Logger().Fatalf("failed to create kube client: %s", err.Error())
		return
	}

	m, err := masker.NewMasker(kubeClient, ns)
	if err != nil {
		log.Logger().Fatalf("failed to create masker: %s", err.Error())
		return
	}

	var words []string
	for w := range m.ReplaceWords {
		words = append(words, w)
	}
	sort.Strings(words)

	log.Logger().Infof("\nreplacing secret words:\n%s", strings.Join(words, "\n"))
}
