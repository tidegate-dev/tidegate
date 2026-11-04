# API reference

This page is generated from the comments in the `.proto` files in `proto/`. Clients call the API with [Connect](https://connectrpc.com/), gRPC or gRPC-Web.

## tidegate.authority.v1.DummyService { #tidegate.authority.v1.DummyService }

DummyService stores and reads texts, to check the path from a client through
the authority into the database and back.

| Method | Request | Response | Description |
| ------ | ------- | -------- | ----------- |
| `Write` | [tidegate.authority.v1.WriteRequest](#tidegate.authority.v1.WriteRequest) | [tidegate.authority.v1.WriteResponse](#tidegate.authority.v1.WriteResponse) | Write stores a text and returns its ID. |
| `Read` | [tidegate.authority.v1.ReadRequest](#tidegate.authority.v1.ReadRequest) | [tidegate.authority.v1.ReadResponse](#tidegate.authority.v1.ReadResponse) | Read returns the text with the given ID. |

## tidegate.authority.v1.PingService { #tidegate.authority.v1.PingService }

PingService checks that a client can reach the authority.

| Method | Request | Response | Description |
| ------ | ------- | -------- | ----------- |
| `Ping` | [tidegate.authority.v1.PingRequest](#tidegate.authority.v1.PingRequest) | [tidegate.authority.v1.PingResponse](#tidegate.authority.v1.PingResponse) | Ping answers with pong. |

## Types

### tidegate.authority.v1.ReadRequest { #tidegate.authority.v1.ReadRequest }

ReadRequest is the input of DummyService.Read.

| Field | Type | Description |
| ----- | ---- | ----------- |
| `id` | `int64` | ID is the one DummyService.Write returned for the text. |

### tidegate.authority.v1.ReadResponse { #tidegate.authority.v1.ReadResponse }

ReadResponse is the result of DummyService.Read.

| Field | Type | Description |
| ----- | ---- | ----------- |
| `text` | `string` | Text is the stored text. |

### tidegate.authority.v1.WriteRequest { #tidegate.authority.v1.WriteRequest }

WriteRequest is the input of DummyService.Write.

| Field | Type | Description |
| ----- | ---- | ----------- |
| `text` | `string` | Text is the text to store. |

### tidegate.authority.v1.WriteResponse { #tidegate.authority.v1.WriteResponse }

WriteResponse is the result of DummyService.Write.

| Field | Type | Description |
| ----- | ---- | ----------- |
| `id` | `int64` | ID identifies the stored text for DummyService.Read. |

### tidegate.authority.v1.PingRequest { #tidegate.authority.v1.PingRequest }

PingRequest is the input of PingService.Ping. It has no fields.

### tidegate.authority.v1.PingResponse { #tidegate.authority.v1.PingResponse }

PingResponse is the result of PingService.Ping.

| Field | Type | Description |
| ----- | ---- | ----------- |
| `message` | `string` | Message is always "pong". |

