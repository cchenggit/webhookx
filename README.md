# WebhookX [![release](https://img.shields.io/github/v/release/webhookx-io/webhookx?color=green)](https://github.com/webhookx-io/webhookx/releases) [![test-workflow](https://github.com/webhookx-io/webhookx/actions/workflows/test.yml/badge.svg)](https://github.com/webhookx-io/webhookx/actions/workflows/test.yml) [![lint-workflow](https://github.com/webhookx-io/webhookx/actions/workflows/lint.yml/badge.svg)](https://github.com/webhookx-io/webhookx/actions/workflows/lint.yml) [![go-report](https://goreportcard.com/badge/github.com/webhookx-io/webhookx)](https://goreportcard.com/report/github.com/webhookx-io/webhookx) [![codecov](https://codecov.io/gh/webhookx-io/webhookx/graph/badge.svg?token=O4AQNRBJRF)](https://codecov.io/gh/webhookx-io/webhookx) 

[![Join Slack](https://img.shields.io/badge/Slack-4285F4?logo=slack&logoColor=white)](https://join.slack.com/t/webhookx/shared_invite/zt-2o4b6hv45-mWm6_WUcQP9qEf1nOxhrrg) [![Follow on Twitter](https://img.shields.io/badge/twitter-1DA1F2?logo=twitter&logoColor=white)](https://twitter.com/webhookx)

WebhookX is an open-source webhooks gateway for message receiving, processing, and delivering.


## Features

- **Admin API:** The admin API(:8080) provides a RESTful API for webhooks entities management.
- **Retries:** WebhookX automatically retries unsuccessful deliveries at configurable delays.
- **Fan out:** Events can be fan out to multiple destinations.
- **Declarative configuration(WIP):**  Managing your configuration through declarative configuration file, and be DevOps compliant.
- **Multi-tenancy:** Multi-tenancy is supported with workspaces. Workspaces provide an isolation of configuration entites.
- **Plugins:**
  - `webhookx-signature`: signing outbound requests with HMAC(SHA-256) by adding `Webhookx-Signature` and `Webhookx-Timestamp` to request header.
  - `transformer`(WIP): transform request before sending outbound requests.
- **Observability:** Metrics and Tracing(WIP).


## Roadmap

- [ ] Data retention policy
- [ ] Insight admin APIs
- [ ] Declarative configuration management

#### Inbound

- [ ] Middlewares/Plugins
- [ ] Authentication
- [ ] Event Transformer

## Installation

```shell
$ docker compose up
```

```shell
$ curl http://localhost:8080
```


## Getting started

#### 1. Create an endpoint that subscribes to specific events

> **Endpoint** represents the event's destination.

```
$ curl -X POST http://localhost:8080/workspaces/default/endpoints \
  --header 'Content-Type: application/json' \
  --data '{
      "request": {
          "url": "https://httpbin.org/anything",
          "method": "POST",
          "headers": {
              "api-key": "secret"
          }
      },
      "events": [
          "charge.succeeded"
      ]
  }'
```

#### 2. Create a source that is used on the Proxy for receiving events

> **Source** represents the ingress of events

```
$ curl -X POST http://localhost:8080/workspaces/default/sources \
  --header 'accept: application/json' \
  --header 'Content-Type: application/json' \
  --data '{
    "path": "/",
    "methods": ["POST"]
  }'
```

#### 3. Send an event to the Proxy (port 8081)

```
$ curl -X POST http://localhost:8081 \
--header 'Content-Type: application/json' \
--data '{
    "event_type": "charge.succeeded",
    "data": {
        "key": "value"
    }
}'
```

#### 4. Retrieve delivery attempts

> Attempt represents an event delivery attempt, and contains inspection information of a delivery. 

```
$ curl http://localhost:8080/workspaces/default/attempts
```

<details>
<summary>See response</summary>

```json
{
  "total": 1,
  "data": [
    {
      "id": "2mYwlR8U5FS6VfK3AHLrYZL75MD",
      "event_id": "2mYwlQZgpNSHTuDr9ApNgvL95x3",
      "endpoint_id": "2mYwjjwRGCwDhtdTtOrVQYETzVt",
      "status": "SUCCESSFUL",
      "attempt_number": 1,
      "scheduled_at": 1727266967962,
      "attempted_at": 1727266968826,
      "trigger_mode": "INITIAL",
      "exhausted": false,
      "error_code": null,
      "request": {
        "method": "POST",
        "url": "https://httpbin.org/anything",
        "headers": null,
        "body": null
      },
      "response": {
        "status": 200,
        "latency": 8573,
        "headers": null,
        "body": null
      },
      "created_at": 1727238167962,
      "updated_at": 1727238167962
    }
  ]
}
```
</details>

Explore more API at [openapi.yml](/openapi.yml).

## CLI

[CLI Reference](https://webhookx.io/docs/cli)


## Runtime dependencies

The gateway requires the following runtime dependencies to work:

- PostgreSQL(>=13): Lower versions of PostgreSQL may work, but have not been fully tested.
- Redis(>=6.2)

## Status and Compatibility

The project is currently under active development, hence breaking changes may be introduced in minor releases.

The public API will strictly follow semantic versioning after `v1.0.0`.

## Sponsoring

## Contributing

We ❤️ pull requests, and we’re continually working hard to make it as easy as possible for developers to contribute.

Thank you for your contribution to WebhookX!

[![Contributors](https://contrib.rocks/image?repo=webhookx-io/webhookx)](https://github.com/webhookx-io/webhookx/graphs/contributors)

[![Star History Chart](https://api.star-history.com/svg?repos=webhookx-io/webhookx&type=Date)](https://api.star-history.com/svg?repos=webhookx-io/webhookx&type=Date)

## License

```
Copyright 2024 WebhookX

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
```
