### Linux

```shell
curl -L https://github.com/jenkins-x-plugins/jx-secret/releases/download/v{{.Version}}/jx-secret-linux-amd64.tar.gz | tar xzv 
sudo mv jx-secret /usr/local/bin
```

### macOS

```shell
curl -L  https://github.com/jenkins-x-plugins/jx-secret/releases/download/v{{.Version}}/jx-secret-darwin-amd64.tar.gz | tar xzv
sudo mv jx-secret /usr/local/bin
```

