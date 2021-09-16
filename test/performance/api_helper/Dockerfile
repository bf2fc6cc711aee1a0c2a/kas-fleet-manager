FROM golang:1.15.7

RUN curl -Lso ocm https://github.com/openshift-online/ocm-cli/releases/download/v0.1.42/ocm-linux-amd64 && \
    chmod +x ocm && \
    mv ocm /usr/local/bin

RUN go get github.com/dgrijalva/jwt-go
