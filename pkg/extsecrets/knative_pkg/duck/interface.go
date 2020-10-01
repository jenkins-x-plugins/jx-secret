package duck

import "github.com/jenkins-x/jx-secret/pkg/extsecrets/knative_pkg/kmeta"

// OneOfOurs is the union of our Accessor interface and the OwnerRefable interface
// that is implemented by our resources that implement the kmeta.Accessor.
type OneOfOurs interface {
	kmeta.Accessor
	kmeta.OwnerRefable
}
