package extsecrets

import (
	"github.com/jenkins-x/jx-secret/pkg/apis/external/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Interface interface {
	List(ns string, listOptions metav1.ListOptions) ([]*v1alpha1.ExternalSecret, error)
}
