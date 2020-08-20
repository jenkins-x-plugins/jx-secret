FROM gcr.io/jenkinsxio/jx-cli-base:0.0.10

ENTRYPOINT ["jx-secret"]

COPY ./build/linux/jx-secret /usr/bin/jx-secret