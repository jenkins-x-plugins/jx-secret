FROM ghcr.io/jenkins-x/jx-boot:latest

ENTRYPOINT ["jx-secret"]

COPY ./build/linux/jx-secret /usr/bin/jx-secret
