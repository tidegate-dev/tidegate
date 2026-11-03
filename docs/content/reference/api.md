# API reference

This page is generated from the comments in the `.proto` files in `proto/`. Clients call the API with [Connect](https://connectrpc.com/), gRPC or gRPC-Web.

## DummyService

DummyService stores and reads texts, to check the path from a client through
the authority into the database and back.

| Method | Request | Response | Description |
| ------ | ------- | -------- | ----------- |
| `Write` | [WriteRequest](#writerequest) | [WriteResponse](#writeresponse) | Write stores a text and returns its ID. |
| `Read` | [ReadRequest](#readrequest) | [ReadResponse](#readresponse) | Read returns the text with the given ID. |

## PingService

PingService checks that a client can reach the authority.

| Method | Request | Response | Description |
| ------ | ------- | -------- | ----------- |
| `Ping` | [PingRequest](#pingrequest) | [PingResponse](#pingresponse) | Ping answers with pong. |

## Types

### ReadRequest

ReadRequest is the input of DummyService.Read.

| Field | Type | Description |
| ----- | ---- | ----------- |
| `id` | `int64` | ID is the one DummyService.Write returned for the text. |

### ReadResponse

ReadResponse is the result of DummyService.Read.

| Field | Type | Description |
| ----- | ---- | ----------- |
| `text` | `string` | Text is the stored text. |

### WriteRequest

WriteRequest is the input of DummyService.Write.

| Field | Type | Description |
| ----- | ---- | ----------- |
| `text` | `string` | Text is the text to store. |

### WriteResponse

WriteResponse is the result of DummyService.Write.

| Field | Type | Description |
| ----- | ---- | ----------- |
| `id` | `int64` | ID identifies the stored text for DummyService.Read. |

### PingRequest

PingRequest is the input of PingService.Ping. It has no fields.

### PingResponse

PingResponse is the result of PingService.Ping.

| Field | Type | Description |
| ----- | ---- | ----------- |
| `message` | `string` | Message is always "pong". |

