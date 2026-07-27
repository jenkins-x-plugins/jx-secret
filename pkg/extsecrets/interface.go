package extsecrets

import (
	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

type Interface interface {
	List(ns string) ([]*esv1.ExternalSecret, error)
}
