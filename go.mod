module github.com/jenkins-x-plugins/jx-secret

go 1.15

require (
	github.com/Azure/go-autorest/autorest v0.11.21 // indirect
	github.com/Azure/go-autorest/autorest/adal v0.9.16 // indirect
	github.com/Masterminds/sprig/v3 v3.2.2
	github.com/cpuguy83/go-md2man v1.0.10
	github.com/google/go-cmp v0.5.5
	github.com/jenkins-x-plugins/secretfacade v0.2.0
	github.com/jenkins-x/go-scm v1.10.10
	github.com/jenkins-x/jx-api/v4 v4.1.5
	github.com/jenkins-x/jx-helpers/v3 v3.0.130
	github.com/jenkins-x/jx-kube-client/v3 v3.0.2
	github.com/jenkins-x/jx-logging/v3 v3.0.6
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/pkg/errors v0.9.1
	github.com/sethvargo/go-password v0.2.0
	github.com/spf13/cobra v1.2.0
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.7.0
	golang.org/x/crypto v0.0.0-20210513164829-c07d793c2f9a
	golang.org/x/text v0.3.7 // indirect
	gopkg.in/validator.v2 v2.0.0-20200605151824-2b28d334fa05
	gopkg.in/yaml.v1 v1.0.0-20140924161607-9f9df34309c0
	k8s.io/api v0.21.5
	k8s.io/apimachinery v0.21.5
	k8s.io/client-go v11.0.0+incompatible
	sigs.k8s.io/kustomize/kyaml v0.10.15
	sigs.k8s.io/structured-merge-diff/v4 v4.1.2 // indirect
	sigs.k8s.io/yaml v1.2.0
)

replace (
	k8s.io/api => k8s.io/api v0.20.6
	k8s.io/apimachinery => k8s.io/apimachinery v0.20.6
	k8s.io/client-go => k8s.io/client-go v0.20.6
)
