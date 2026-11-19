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
| `Login` | [tidegate.authority.v1.LoginRequest](#tidegate.authority.v1.LoginRequest) | [tidegate.authority.v1.LoginResponse](#tidegate.authority.v1.LoginResponse) | Login checks a user's local password and returns a session token. All wrong credentials get the same error. |
| `Logout` | [tidegate.authority.v1.LogoutRequest](#tidegate.authority.v1.LogoutRequest) | [tidegate.authority.v1.LogoutResponse](#tidegate.authority.v1.LogoutResponse) | Logout revokes the session token in the call's `Authorization: Bearer` header, so later calls with it fail. A missing, unknown or expired token fails with unauthenticated. |

## tidegate.authority.v1.PingService { #tidegate.authority.v1.PingService }

PingService checks that a client can reach the authority.

| Method | Request | Response | Description |
| ------ | ------- | -------- | ----------- |
| `Ping` | [tidegate.authority.v1.PingRequest](#tidegate.authority.v1.PingRequest) | [tidegate.authority.v1.PingResponse](#tidegate.authority.v1.PingResponse) | Ping answers with pong. |

## tidegate.authority.v1.ProjectService { #tidegate.authority.v1.ProjectService }

ProjectService creates and lists projects. A project holds assets, roles and
bindings.

| Method | Request | Response | Description |
| ------ | ------- | -------- | ----------- |
| `CreateProject` | [tidegate.authority.v1.CreateProjectRequest](#tidegate.authority.v1.CreateProjectRequest) | [tidegate.authority.v1.CreateProjectResponse](#tidegate.authority.v1.CreateProjectResponse) | CreateProject creates a project and returns it. A name that another project has fails with already_exists. |
| `ListProjects` | [tidegate.authority.v1.ListProjectsRequest](#tidegate.authority.v1.ListProjectsRequest) | [tidegate.authority.v1.ListProjectsResponse](#tidegate.authority.v1.ListProjectsResponse) | ListProjects returns every project. |

## tidegate.authority.v1.RoleService { #tidegate.authority.v1.RoleService }

RoleService creates roles. A role is a named list of permissions in one
project.

| Method | Request | Response | Description |
| ------ | ------- | -------- | ----------- |
| `CreateRole` | [tidegate.authority.v1.CreateRoleRequest](#tidegate.authority.v1.CreateRoleRequest) | [tidegate.authority.v1.CreateRoleResponse](#tidegate.authority.v1.CreateRoleResponse) | CreateRole creates a role in a project and returns it. A project that doesn't exist fails with not_found. A name that another role of the project has fails with already_exists. |

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
| `all_sessions` | `bool` | AllSessions revokes every session token of the caller's user, not only the one the call carries. |  |

### tidegate.authority.v1.LogoutResponse { #tidegate.authority.v1.LogoutResponse }

LogoutResponse is the result of LoginService.Logout.

### tidegate.authority.v1.PingRequest { #tidegate.authority.v1.PingRequest }

PingRequest is the input of PingService.Ping. It has no fields.

### tidegate.authority.v1.PingResponse { #tidegate.authority.v1.PingResponse }

PingResponse is the result of PingService.Ping.

| Field | Type | Description | Rules |
| ----- | ---- | ----------- | ----- |
| `message` | `string` | Message is always "pong". |  |

### tidegate.authority.v1.CreateProjectRequest { #tidegate.authority.v1.CreateProjectRequest }

CreateProjectRequest is the input of ProjectService.CreateProject.

| Field | Type | Description | Rules |
| ----- | ---- | ----------- | ----- |
| `name` | `string` | Name is the name of the new project. | Starts with a lowercase letter, goes on with lowercase letters, digits and dashes, and ends with a letter or digit. At most 63 characters. |

### tidegate.authority.v1.CreateProjectResponse { #tidegate.authority.v1.CreateProjectResponse }

CreateProjectResponse is the result of ProjectService.CreateProject.

| Field | Type | Description | Rules |
| ----- | ---- | ----------- | ----- |
| `project` | `tidegate.authority.v1.Project` | Project is the new project. |  |

### tidegate.authority.v1.ListProjectsRequest { #tidegate.authority.v1.ListProjectsRequest }

ListProjectsRequest is the input of ProjectService.ListProjects.

### tidegate.authority.v1.ListProjectsResponse { #tidegate.authority.v1.ListProjectsResponse }

ListProjectsResponse is the result of ProjectService.ListProjects.

| Field | Type | Description | Rules |
| ----- | ---- | ----------- | ----- |
| `projects` | `repeated tidegate.authority.v1.Project` | Projects are all projects. |  |

### tidegate.authority.v1.Project { #tidegate.authority.v1.Project }

Project is a project as the authority stores it.

| Field | Type | Description | Rules |
| ----- | ---- | ----------- | ----- |
| `id` | `string` | ID identifies the project. It is a UUID. |  |
| `name` | `string` | Name is the project's name. |  |

### tidegate.authority.v1.CreateRoleRequest { #tidegate.authority.v1.CreateRoleRequest }

CreateRoleRequest is the input of RoleService.CreateRole.

| Field | Type | Description | Rules |
| ----- | ---- | ----------- | ----- |
| `project` | `string` | Project is the name of the project that gets the role. | Starts with a lowercase letter, goes on with lowercase letters, digits and dashes, and ends with a letter or digit. At most 63 characters. |
| `name` | `string` | Name is the name of the new role. | Starts with a lowercase letter, goes on with lowercase letters, digits and dashes, and ends with a letter or digit. At most 63 characters. |
| `permissions` | `tidegate.authority.v1.ProjectPermissions` | Permissions are the permissions of the new role. A role without them gives its holders nothing. |  |

### tidegate.authority.v1.CreateRoleResponse { #tidegate.authority.v1.CreateRoleResponse }

CreateRoleResponse is the result of RoleService.CreateRole.

| Field | Type | Description | Rules |
| ----- | ---- | ----------- | ----- |
| `role` | `tidegate.authority.v1.Role` | Role is the new role. |  |

### tidegate.authority.v1.ProjectPermissions { #tidegate.authority.v1.ProjectPermissions }

ProjectPermissions says what the holders of a role may do with the roles
and bindings of its project.

| Field | Type | Description | Rules |
| ----- | ---- | ----------- | ----- |
| `create_roles` | `bool` | CreateRoles means the holders may create roles in the project. |  |
| `escalate` | `bool` | Escalate means the holders may create roles with permissions that they don't have themselves. |  |
| `create_bindings` | `bool` | CreateBindings means the holders may create bindings in the project. |  |
| `bind` | `bool` | Bind means the holders may bind roles with permissions that they don't have themselves. |  |

### tidegate.authority.v1.Role { #tidegate.authority.v1.Role }

Role is a role as the authority stores it.

| Field | Type | Description | Rules |
| ----- | ---- | ----------- | ----- |
| `id` | `string` | ID identifies the role. It is a UUID. |  |
| `project_id` | `string` | ProjectID identifies the project that the role belongs to. It is a UUID. |  |
| `name` | `string` | Name is the role's name. |  |
| `permissions` | `tidegate.authority.v1.ProjectPermissions` | Permissions are the permissions that the role gives its holders. |  |

