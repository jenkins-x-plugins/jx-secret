## Working with External Secrets

### Mappings

When using the [jx-secret convert](cmd/jx-secret_convert.md) command to generate [ExternalSecret](https://github.com/godaddy/kubernetes-external-secrets) CRDs you may wish to use a custom mapping of `Secret` names and data keys to key/properties in Vault.

To do this just create a [.jx/secret/mapping/config.yaml](https://github.com/jenkins-x/jx-gitops/blob/master/.jx/gitops/secret-mappings.yaml) file in your directory tree when running the command. 

You can then customise the `key` and/or `property` values that are used in the generated [ExternalSecret](https://github.com/godaddy/kubernetes-external-secrets) CRDs

For more details see the [Mapping Configuration Reference](mapping.md)

### References

The configuration file formats and schema references are here:

* [ExternalSecret](external.md#skubernetes-client.io%2fv1)
* [SecretMapping](mapping.md#secret.jenkins-x.io/v1alpha1.SecretMapping)
* [Schema](schema.md#secret.jenkins-x.io/v1alpha1.Schema)
