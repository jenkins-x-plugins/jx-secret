# jx-secret

[![Documentation](https://godoc.org/github.com/jenkins-x/jx-secret?status.svg)](https://pkg.go.dev/mod/github.com/jenkins-x/jx-secret)
[![Go Report Card](https://goreportcard.com/badge/github.com/jenkins-x/jx-secret)](https://goreportcard.com/report/github.com/jenkins-x/jx-secret)
[![Releases](https://img.shields.io/github/release-pre/jenkins-x/jx-secret.svg)](https://github.com/jenkins-x/jx-secret/releases)
[![LICENSE](https://img.shields.io/github/license/jenkins-x/jx-secret.svg)](https://github.com/jenkins-x/jx-secret/blob/master/LICENSE)
[![Slack Status](https://img.shields.io/badge/slack-join_chat-white.svg?logo=slack&style=social)](https://slack.k8s.io/)

`jx-secret` is a small command line tool working with [Kubernetes External Secrets](https://github.com/godaddy/kubernetes-external-secrets).

## Getting Started

Download the [jx-secret binary](https://github.com/jenkins-x/jx-secret/releases) for your operating system and add it to your `$PATH`.

There will be an `app` you can install soon too...

## Commands

See the [jx-secret command reference](https://github.com/jenkins-x/jx-secret/blob/master/docs/cmd/jx-secret.md)


### Mappings

When using the [jx-secret convert](cmd/jx-secret_convert.md) command to generate [ExternalSecret](https://github.com/godaddy/kubernetes-external-secrets) CRDs you may wish to use a custom mapping of `Secret` names and data keys to key/properties in Vault.

To do this just create a [.jx/secret/mapping/config.yaml](https://github.com/jenkins-x/jx-gitops/blob/master/.jx/gitops/secret-mappings.yaml) file in your directory tree when running the command. 

You can then customise the `key` and/or `property` values that are used in the generated [ExternalSecret](https://github.com/godaddy/kubernetes-external-secrets) CRDs

For more details see the [Mapping Configuration Reference](mapping.md)

## Configuration References

The configuration file formats and schema references are here:

* [ExternalSecret](docs/external.md#kubernetes-client.io/v1.ExternalSecret)
* [SecretMapping](docs/mapping.md#secret.jenkins-x.io/v1alpha1.SecretMapping)
* [Schema](docs/schema.md#secret.jenkins-x.io/v1alpha1.Schema)
