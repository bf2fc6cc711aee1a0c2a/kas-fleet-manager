# Deploying Observability Remote Write Proxy in OpenShift

This document explains how to configure and deploy Observability Remote Write
Proxy [Observability Remote Write Proxy](https://github.com/bf2fc6cc711aee1a0c2a/observability-remote-write-proxy)
in OpenShift.

This document assumes that the reader is familiar with Observability
Remote Write Proxy as well as with its configuration. For information about it please go to the
[Observability Remote Write Proxy repository](https://github.com/bf2fc6cc711aee1a0c2a/observability-remote-write-proxy).

## Configuring and Deploying Observability Remote Write Proxy

To send metrics to Observatorium from the Data Plane side, using
[Observability Remote Write Proxy](https://github.com/bf2fc6cc711aee1a0c2a/observability-remote-write-proxy)
is required. This avoids needing to store Observatorium credentials in the
Data Planes managed by KAS Fleet Manager.

Observability Remote Write Proxy is usually deployed together with the KAS
Fleet Manager deployment.

The requirement is that the proxy needs to be reachable at network level by
the Data Planes being managed.

To configure and deploy Observability Remote Write Prox follow the next steps:

If you want to enable Observability Remote Write Proxy OIDC Token Retrieval
the OIDC configuration secret needs to be created. To do so execute:
```sh
make deploy/observability-remote-write-proxy/secrets OBSERVABILITY_REMOTE_WRITE_PROXY_OIDC_CONFIG_FILEPATH=<file-containing-obsrwproxy-config>
```

Where OBSERVABILITY_REMOTE_WRITE_PROXY_OIDC_CONFIG_FILEPATH contains the
Observability Remote Write Proxy OIDC configuration file. See a
[sample Observability Remote Write Proxy OIDC configuration file](https://github.com/bf2fc6cc711aee1a0c2a/observability-remote-write-proxy/blob/main/secrets/oidc-config.yaml.sample) to see the available attributes.

Then, an OpenShift route to expose the Observability Remote Write Proxy has to
be created. To do so, execute the following command:
```sh
make deploy/observability-remote-write-proxy/route
```

Finally, deploy the Observability Remote Write Proxy by running:
```sh
make deploy/observability-remote-write-proxy \
  OBSERVABILITY_REMOTE_WRITE_PROXY_FORWARD_URL=<proxy-forward-url>
  OBSERVABILITY_REMOTE_WRITE_PROXY_OIDC_ENABLED=<bool>
  OBSERVABILITY_REMOTE_WRITE_PROXY_TOKEN_VERIFICATION_ENABLED=<bool>
  OBSERVABILITY_REMOTE_WRITE_PROXY_TOKEN_VERIFICATION_URL=<token-verification-url>
```

The description of the previously shown parameters are:
* OBSERVABILITY_REMOTE_WRITE_PROXY_FORWARD_URL: URL of a Prometheus Remote
  Write Receiver Endpoint to which requests of the Observability Remote Write
  Proxy will be forwarded to after they are successfully verified. The
  Observatorium Remote Write endpoint URL can be used as a value here.
* OBSERVABILITY_REMOTE_WRITE_PROXY_OIDC_ENABLED: Enables OIDC token retrieval.
  If enabled, the retrieved OIDC token is provided when
  Observability Remote Write Proxy forwards the request to the
  Prometheus Remote Write Receiver Endpoint specified in
  OBSERVABILITY_REMOTE_WRITE_PROXY_FORWARD_URL.
* OBSERVABILITY_REMOTE_WRITE_PROXY_TOKEN_VERIFICATION_ENABLED: Enables Token
  verification on requests that arrive to the Observability Remote Write Proxy.
  If enabled, Observability Remote Write Proxy performs
  verification of the token provided as part of the 'Authorization' HTTP
  Header of the requests that receives by using
  OBSERVABILITY_REMOTE_WRITE_PROXY_TOKEN_VERIFICATION_URL
* OBSERVABILITY_REMOTE_WRITE_PROXY_TOKEN_VERIFICATION_URL: URL endpoint used as
  part of the Token Verification on requests that arrive to the Observability
  Remote Write Proxy.
  If OBSERVABILITY_REMOTE_WRITE_PROXY_TOKEN_VERIFICATION_ENABLED is enabled
  it must be set and not empty

After this, an Observability Remote Write Proxy should be up and running.