FROM gcr.io/jenkinsxio-labs-private/jxl-base:0.0.52

ENTRYPOINT ["jx-secret"]

COPY ./build/linux/jx-secret /usr/bin/jx-secret