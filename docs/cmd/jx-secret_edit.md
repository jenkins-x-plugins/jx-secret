## jx-secret edit

Edits secret values in the underlying secret stores for ExternalSecrets

### Usage

```
jx-secret edit
```

### Synopsis

Edits secret values in the underlying secret stores for ExternalSecrets

### Examples

  # edit any missing mandatory secrets
  jx-secret edit
  
  # edit any secrets with a given filter
  jx-secret edit --filter nexus

### Options

```
      --all                     for interactive mode do you want to select all of the properties to edit by default. Otherwise none are selected and you choose to select the properties to change
  -d, --dir string              the directory to look for the .jx/secret/mapping/secret-mappings.yaml file (default ".")
      --external-vault string   specify whether we are using external vault or not
  -f, --filter string           filter on the Secret / ExternalSecret names to enter
  -h, --help                    help for edit
  -i, --interactive             interactive mode asks the user for the Secret name and the properties to edit
  -m, --multiple                for interactive mode do you want to select multiple secrets to edit. If not defaults to just picking a single secret
  -n, --ns string               the namespace to filter the ExternalSecret resources
```

### SEE ALSO

* [jx-secret](jx-secret.md)	 - commands for working with Secrets, ExternalSecrets and external secret stores

###### Auto generated by spf13/cobra on 10-Oct-2024
