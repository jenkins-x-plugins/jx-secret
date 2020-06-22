FROM gcr.io/jenkinsxio-labs-private/jxl-base:0.0.52

ENTRYPOINT ["jx-extsecret"]

COPY ./build/linux/jx-extsecret /usr/bin/jx-extsecret