# API reference

This page is generated from the comments in the `.proto` files in `proto/`. Clients call the API with [Connect](https://connectrpc.com/), gRPC or gRPC-Web.

## tidegate.authority.v1.DummyService { #tidegate.authority.v1.DummyService }

DummyService stores and reads texts, to check the path from a client through
the authority into the database and back.

| Method | Request | Response | Description |
| ------ | ------- | -------- | ----------- |
| `Write` | [tidegate.authority.v1.WriteRequest](#tidegate.authority.v1.WriteRequest) | [tidegate.authority.v1.WriteResponse](#tidegate.authority.v1.WriteResponse) | Write stores a text and returns its ID. |
| `Read` | [tidegate.authority.v1.ReadRequest](#tidegate.authority.v1.ReadRequest) | [tidegate.authority.v1.ReadResponse](#tidegate.authority.v1.ReadResponse) | Read returns the text with the given ID. |

## tidegate.authority.v1.LoginService { #tidegate.authority.v1.LoginService }

LoginService signs users in to the authority.

| Method | Request | Response | Description |
| ------ | ------- | -------- | ----------- |
| `Login` | [tidegate.authority.v1.LoginRequest](#tidegate.authority.v1.LoginRequest) | [tidegate.authority.v1.LoginResponse](#tidegate.authority.v1.LoginResponse) | Login checks a user's local password and returns a login token. All wrong credentials get the same error. |
| `Logout` | [tidegate.authority.v1.LogoutRequest](#tidegate.authority.v1.LogoutRequest) | [tidegate.authority.v1.LogoutResponse](#tidegate.authority.v1.LogoutResponse) | Logout revokes the login token in the call's `Authorization: Bearer` header, so later calls with it fail. A missing, unknown or expired token fails with unauthenticated. |

## tidegate.authority.v1.PingService { #tidegate.authority.v1.PingService }

PingService checks that a client can reach the authority.

| Method | Request | Response | Description |
| ------ | ------- | -------- | ----------- |
| `Ping` | [tidegate.authority.v1.PingRequest](#tidegate.authority.v1.PingRequest) | [tidegate.authority.v1.PingResponse](#tidegate.authority.v1.PingResponse) | Ping answers with pong. |

## Types

### tidegate.authority.v1.ReadRequest { #tidegate.authority.v1.ReadRequest }

ReadRequest is the input of DummyService.Read.

| Field | Type | Description | Rules |
| ----- | ---- | ----------- | ----- |
| `id` | `int64` | ID is the one DummyService.Write returned for the text. |  |

### tidegate.authority.v1.ReadResponse { #tidegate.authority.v1.ReadResponse }

ReadResponse is the result of DummyService.Read.

| Field | Type | Description | Rules |
| ----- | ---- | ----------- | ----- |
| `text` | `string` | Text is the stored text. |  |

### tidegate.authority.v1.WriteRequest { #tidegate.authority.v1.WriteRequest }

WriteRequest is the input of DummyService.Write.

| Field | Type | Description | Rules |
| ----- | ---- | ----------- | ----- |
| `text` | `string` | Text is the text to store. | At most 1024 characters. |

### tidegate.authority.v1.WriteResponse { #tidegate.authority.v1.WriteResponse }

WriteResponse is the result of DummyService.Write.

| Field | Type | Description | Rules |
| ----- | ---- | ----------- | ----- |
| `id` | `int64` | ID identifies the stored text for DummyService.Read. |  |

### tidegate.authority.v1.LoginRequest { #tidegate.authority.v1.LoginRequest }

LoginRequest is the input of LoginService.Login.

| Field | Type | Description | Rules |
| ----- | ---- | ----------- | ----- |
| `user_name` | `string` | UserName is the user's name, in any letter case. | Not empty. |
| `password` | `string` | Password is the user's local password. | Not empty, at most 1024 bytes. |

### tidegate.authority.v1.LoginResponse { #tidegate.authority.v1.LoginResponse }

LoginResponse is the result of LoginService.Login.

| Field | Type | Description | Rules |
| ----- | ---- | ----------- | ----- |
| `token` | `string` | Token is valid for the authority's token lifetime, 12 hours by default. |  |

### tidegate.authority.v1.LogoutRequest { #tidegate.authority.v1.LogoutRequest }

LogoutRequest is the input of LoginService.Logout.

| Field | Type | Description | Rules |
| ----- | ---- | ----------- | ----- |
| `all_sessions` | `bool` | AllSessions revokes every login token of the caller's user, not only the one the call carries. |  |

### tidegate.authority.v1.LogoutResponse { #tidegate.authority.v1.LogoutResponse }

LogoutResponse is the result of LoginService.Logout.

### tidegate.authority.v1.PingRequest { #tidegate.authority.v1.PingRequest }

PingRequest is the input of PingService.Ping. It has no fields.

### tidegate.authority.v1.PingResponse { #tidegate.authority.v1.PingResponse }

PingResponse is the result of PingService.Ping.

| Field | Type | Description | Rules |
| ----- | ---- | ----------- | ----- |
| `message` | `string` | Message is always "pong". |  |

