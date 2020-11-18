module github.com/jenkins-x/jx-secret

go 1.15

require (
	github.com/Masterminds/sprig/v3 v3.1.0
	github.com/alecthomas/assert v0.0.0-20170929043011-405dbfeb8e38
	github.com/alecthomas/colour v0.1.0 // indirect
	github.com/alecthomas/repr v0.0.0-20201103221029-55c485bd663f // indirect
	github.com/cpuguy83/go-md2man v1.0.10
	github.com/google/go-cmp v0.5.2
	github.com/jenkins-x/jx-api/v3 v3.0.3
	github.com/jenkins-x/jx-helpers/v3 v3.0.23
	github.com/jenkins-x/jx-kube-client/v3 v3.0.1
	github.com/jenkins-x/jx-logging/v3 v3.0.2
	github.com/pkg/errors v0.9.1
	github.com/sethvargo/go-password v0.2.0
	github.com/spf13/cobra v1.0.0
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.6.1
	golang.org/x/crypto v0.0.0-20200728195943-123391ffb6de
	gopkg.in/validator.v2 v2.0.0-20200605151824-2b28d334fa05
	gopkg.in/yaml.v1 v1.0.0-20140924161607-9f9df34309c0
	k8s.io/api v0.19.2
	k8s.io/apimachinery v0.19.2
	k8s.io/client-go v0.19.2
	sigs.k8s.io/kustomize/kyaml v0.6.1
	sigs.k8s.io/yaml v1.2.0
)
