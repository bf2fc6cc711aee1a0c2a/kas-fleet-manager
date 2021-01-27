FROM grubykarol/locust:1.2.3-python3.9-alpine3.12

RUN apk --no-cache add curl

RUN pip install pyjwt

RUN pip install --upgrade locust

RUN curl -Lso ocm https://github.com/openshift-online/ocm-cli/releases/download/v0.1.42/ocm-linux-amd64 && \
    chmod +x ocm && \
    mv ocm /usr/local/bin
