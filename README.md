# File System HTTP Server

[![Build Status](https://github.com/bryk-io/serve/workflows/ci/badge.svg?branch=main)](https://github.com/bryk-io/serve/actions)
[![Version](https://img.shields.io/github/tag/bryk-io/serve.svg)](https://github.com/bryk-io/serve/releases)
[![Software License](https://img.shields.io/badge/license-BSD3-red.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/bryk-io/serve?style=flat)](https://goreportcard.com/report/github.com/bryk-io/serve)
[![Contributor Covenant](https://img.shields.io/badge/Contributor%20Covenant-v2.0-ff69b4.svg)](.github/CODE_OF_CONDUCT.md)

Easily deploy a robust HTTP(S) server from contents on your file system.

Features include:

- OpenTelemetry support
- Error reporting (using Sentry)
- TLS
- Gzip compression
- Cache
- Robust logging
- PROXY protocol detection
- Flexible CORS capabilities

## Quick start

Run a server instance on port `8080` with default settings from contents on your `/var/www` local directory.

```shell
serve run -p 8080 /var/www
```

## Configuration

You can adjust the settings available using a YAML configuration file. The file is loaded automatically if available at:

- `pwd`/config.yaml
- ${HOME}/.config.yaml
- /etc/serve/config.yaml

```yaml
otel:
  service_name: "serve"
  service_version: "0.1.0"
  collector: "" # OTEL collector endpoint, if not provided output will be discarded
  attributes:
    environment: dev
    host: "my-local-host"
  sentry:
    dsn: "" # Sentry DNS, if not provided output will be discarded
    environment: dev
server:
  port: 9090
  cache: 3600
  proxy_protocol: false
  tls:
    enabled: false
    system_ca: true
    cert: /etc/serve/tls/tls.crt
    key: /etc/serve/tls/tls.key
    custom_ca: []
  middleware:
    gzip: 7
    metadata:
      headers:
        - authorization
        - x-api-key
    cors:
      max_age: 300
      options_status_code: 200
      allow_credentials: true
      ignore_options: false
      allowed_headers:
        - authorization
        - content-type
        - x-api-key
      allowed_methods:
        - get
        - head
        - post
        - options
      allowed_origins:
        - "*"
      exposed_headers:
        - authorization
        - x-api-key
```
