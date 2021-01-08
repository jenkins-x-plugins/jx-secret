package extsecrets

import (
	v1 "github.com/jenkins-x/jx-secret/pkg/apis/external/v1"
)

type Interface interface {
	List(ns string) ([]*v1.ExternalSecret, error)
}
