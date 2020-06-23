FROM gcr.io/jenkinsxio-labs-private/jxl-base:0.0.52

ENTRYPOINT ["jx-alpha-extsecret"]

COPY ./build/linux/jx-alpha-extsecret /usr/bin/jx-alpha-extsecret