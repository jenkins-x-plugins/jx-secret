package extsecrets

import (
	v1 "github.com/jenkins-x/jx-secret/pkg/apis/external/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Interface interface {
	List(ns string, listOptions metav1.ListOptions) ([]*v1.ExternalSecret, error)
}
