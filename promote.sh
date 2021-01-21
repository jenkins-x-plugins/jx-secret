#!/bin/bash

echo "promoting the new version ${VERSION} to downstream repositories"

# TODO - lets disable promoting the new jx-secret release to jx-cli until we've got the new secret facade code working
# jx step create pr regex --regex '\s+SecretVersion = "(?P<version>.*)"' --version ${VERSION} --files pkg/plugins/versions.go --repo https://github.com/jenkins-x/jx-cli.git
