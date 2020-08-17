module github.com/jenkins-x/jx-secret

go 1.13

require (
	github.com/Azure/go-autorest/autorest/validation v0.2.0 // indirect
	github.com/alecthomas/assert v0.0.0-20170929043011-405dbfeb8e38
	github.com/alecthomas/colour v0.1.0 // indirect
	github.com/alecthomas/repr v0.0.0-20200325044227-4184120f674c // indirect
	github.com/coreos/bbolt v1.3.3 // indirect
	github.com/coreos/etcd v3.3.17+incompatible // indirect
	github.com/cpuguy83/go-md2man v1.0.10
	github.com/docker/docker v1.13.1 // indirect
	github.com/gobuffalo/envy v1.7.1 // indirect
	github.com/google/go-cmp v0.4.0
	github.com/googleapis/gnostic v0.4.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway v1.12.1 // indirect
	github.com/jenkins-x/gen-crd-api-reference-docs v0.1.6 // indirect
	github.com/jenkins-x/jx-api v0.0.17
	github.com/jenkins-x/jx-helpers v1.0.44
	github.com/jenkins-x/jx-logging v0.0.11
	github.com/klauspost/cpuid v1.2.2 // indirect
	github.com/pkg/errors v0.9.1
	github.com/prometheus/procfs v0.0.11 // indirect
	github.com/sethvargo/go-password v0.2.0
	github.com/spf13/cobra v1.0.0
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.6.1
	google.golang.org/genproto v0.0.0-20200326112834-f447254575fd // indirect
	gopkg.in/validator.v2 v2.0.0-20200605151824-2b28d334fa05
	gopkg.in/yaml.v1 v1.0.0-20140924161607-9f9df34309c0
	k8s.io/api v0.17.6
	k8s.io/apimachinery v0.17.6
	k8s.io/client-go v11.0.1-0.20190805182717-6502b5e7b1b5+incompatible
	k8s.io/kubernetes v1.14.7 // indirect
	k8s.io/utils v0.0.0-20200124190032-861946025e34 // indirect
	knative.dev/pkg v0.0.0-20200626182828-bce16cf78661
	sigs.k8s.io/kustomize/kyaml v0.6.0
	sigs.k8s.io/yaml v1.2.0
)

replace (
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v13.3.1+incompatible

	k8s.io/api => k8s.io/api v0.17.2
	k8s.io/apimachinery => k8s.io/apimachinery v0.17.2
	k8s.io/client-go => k8s.io/client-go v0.16.5
)
