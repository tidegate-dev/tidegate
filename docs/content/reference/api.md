# API reference

This page is generated from the comments in the `.proto` files in `proto/`. Clients call the API with [Connect](https://connectrpc.com/), gRPC or gRPC-Web.

## phemera.authority.v1.AssetService { #phemera.authority.v1.AssetService }

AssetService creates and lists assets. An asset is a target in one project
that phemera gives access to, like an SSH host.

| Method | Request | Response | Description |
| ------ | ------- | -------- | ----------- |
| `CreateAsset` | [phemera.authority.v1.CreateAssetRequest](#phemera.authority.v1.CreateAssetRequest) | [phemera.authority.v1.CreateAssetResponse](#phemera.authority.v1.CreateAssetResponse) | CreateAsset creates an asset in a project and returns it. A project that doesn't exist fails with not_found. A name that another asset of the project has fails with already_exists. |
| `ListAssets` | [phemera.authority.v1.ListAssetsRequest](#phemera.authority.v1.ListAssetsRequest) | [phemera.authority.v1.ListAssetsResponse](#phemera.authority.v1.ListAssetsResponse) | ListAssets returns every asset of a project. A project that doesn't exist fails with not_found. |

## phemera.authority.v1.BindingService { #phemera.authority.v1.BindingService }

BindingService creates and lists bindings. A binding gives a role of one
project to a user.

| Method | Request | Response | Description |
| ------ | ------- | -------- | ----------- |
| `CreateBinding` | [phemera.authority.v1.CreateBindingRequest](#phemera.authority.v1.CreateBindingRequest) | [phemera.authority.v1.CreateBindingResponse](#phemera.authority.v1.CreateBindingResponse) | CreateBinding gives a role of a project to a user. It fails with not_found if the project, the role or the user doesn't exist, and with already_exists if the user already has the role. |
| `ListBindings` | [phemera.authority.v1.ListBindingsRequest](#phemera.authority.v1.ListBindingsRequest) | [phemera.authority.v1.ListBindingsResponse](#phemera.authority.v1.ListBindingsResponse) | ListBindings returns every binding of a project. It fails with not_found if the project doesn't exist. |

## phemera.authority.v1.LoginService { #phemera.authority.v1.LoginService }

LoginService signs users in to the authority.

| Method | Request | Response | Description |
| ------ | ------- | -------- | ----------- |
| `Login` | [phemera.authority.v1.LoginRequest](#phemera.authority.v1.LoginRequest) | [phemera.authority.v1.LoginResponse](#phemera.authority.v1.LoginResponse) | Login checks a user's local password and returns a session token. All wrong credentials get the same error. |
| `Logout` | [phemera.authority.v1.LogoutRequest](#phemera.authority.v1.LogoutRequest) | [phemera.authority.v1.LogoutResponse](#phemera.authority.v1.LogoutResponse) | Logout revokes the session token in the call's `Authorization: Bearer` header, so later calls with it fail. A missing, unknown or expired token fails with unauthenticated. |

## phemera.authority.v1.ProjectService { #phemera.authority.v1.ProjectService }

ProjectService creates and lists projects. A project holds assets, roles and
bindings.

| Method | Request | Response | Description |
| ------ | ------- | -------- | ----------- |
| `CreateProject` | [phemera.authority.v1.CreateProjectRequest](#phemera.authority.v1.CreateProjectRequest) | [phemera.authority.v1.CreateProjectResponse](#phemera.authority.v1.CreateProjectResponse) | CreateProject creates a project and returns it. A name that another project has fails with already_exists. |
| `ListProjects` | [phemera.authority.v1.ListProjectsRequest](#phemera.authority.v1.ListProjectsRequest) | [phemera.authority.v1.ListProjectsResponse](#phemera.authority.v1.ListProjectsResponse) | ListProjects returns every project. |

## phemera.authority.v1.RoleService { #phemera.authority.v1.RoleService }

RoleService creates roles. A role is a named list of permissions in one
project.

| Method | Request | Response | Description |
| ------ | ------- | -------- | ----------- |
| `CreateRole` | [phemera.authority.v1.CreateRoleRequest](#phemera.authority.v1.CreateRoleRequest) | [phemera.authority.v1.CreateRoleResponse](#phemera.authority.v1.CreateRoleResponse) | CreateRole creates a role in a project and returns it. A project that doesn't exist fails with not_found. A name that another role of the project has fails with already_exists. |

## phemera.authority.v1.TicketService { #phemera.authority.v1.TicketService }

TicketService records asset access requests.

| Method | Request | Response | Description |
| ------ | ------- | -------- | ----------- |
| `CreateTicket` | [phemera.authority.v1.CreateTicketRequest](#phemera.authority.v1.CreateTicketRequest) | [phemera.authority.v1.CreateTicketResponse](#phemera.authority.v1.CreateTicketResponse) | CreateTicket issues a ticket for the signed-in user and the requested asset. For now, the authority assumes the user has a project binding whose role permits access to the asset. If the asset does not exist, the call fails with not_found. |

## Types

### phemera.authority.v1.CreateAssetRequest { #phemera.authority.v1.CreateAssetRequest }

CreateAssetRequest is the input of AssetService.CreateAsset.

| Field | Type | Description | Rules |
| ----- | ---- | ----------- | ----- |
| `project_name` | `string` | ProjectName is the name of the project that gets the asset. | Starts with a lowercase letter, goes on with lowercase letters, digits and dashes, and ends with a letter or digit. At most 63 characters. |
| `name` | `string` | Name is the name of the new asset. | Starts with a lowercase letter, goes on with lowercase letters, digits and dashes, and ends with a letter or digit. At most 63 characters. |
| `labels` | `repeated phemera.authority.v1.CreateAssetRequest.LabelsEntry` | Labels are the labels of the new asset. | At most 64 labels. A key is an optional DNS name and a slash, followed by a name. The DNS name has at most 253 characters, in lowercase parts of at most 63. A name has at most 63 letters, digits, dashes, underscores and dots, and starts and ends with a letter or digit. A value is empty or follows the rule for a name. |
| `ssh` | `phemera.authority.v1.CreateAssetRequest.SSHAsset` | SSH makes the asset a host that phemera reaches over SSH. |  |

### phemera.authority.v1.CreateAssetRequest.LabelsEntry { #phemera.authority.v1.CreateAssetRequest.LabelsEntry }



| Field | Type | Description | Rules |
| ----- | ---- | ----------- | ----- |
| `key` | `string` |  |  |
| `value` | `string` |  |  |

### phemera.authority.v1.CreateAssetRequest.SSHAsset { #phemera.authority.v1.CreateAssetRequest.SSHAsset }

SSHAsset is an SSH host and what phemera logs in to it with.

| Field | Type | Description | Rules |
| ----- | ---- | ----------- | ----- |
| `target` | `phemera.authority.v1.SSHTarget` | Target is the host's address and its host key. | Required. |
| `login` | `phemera.authority.v1.CreateAssetRequest.SSHLogin` | Login is the user that phemera logs in as and what the user logs in with. | Required. |

### phemera.authority.v1.CreateAssetRequest.SSHLogin { #phemera.authority.v1.CreateAssetRequest.SSHLogin }

SSHLogin is the user that phemera logs in as on an SSH host and what the
user logs in with.

| Field | Type | Description | Rules |
| ----- | ---- | ----------- | ----- |
| `user_name` | `string` | UserName is the user that phemera logs in as on the host, like "deploy", "deploy@example.com" or "EXAMPLE\deploy". | Uppercase and lowercase letters from A to Z, digits, underscores, dots, dashes and at signs. It doesn't start with a dash. A Windows domain and one backslash may come first, and the domain follows the same rule. At most 64 characters. These are the characters that useradd(8) allows, plus the at sign of a name with a domain. |
| `password` | `string` | Password is the user's password on the host. | Not empty, at most 1024 bytes. |
| `private_key` | `string` | PrivateKey is a private key of the user in PEM, without a passphrase. A key that phemera can't read or that has a passphrase fails with invalid_argument. | Not empty, at most 16384 bytes. |

### phemera.authority.v1.CreateAssetResponse { #phemera.authority.v1.CreateAssetResponse }

CreateAssetResponse is the result of AssetService.CreateAsset.

| Field | Type | Description | Rules |
| ----- | ---- | ----------- | ----- |
| `id` | `string` | ID identifies the new asset. It is a UUID. |  |
| `project_id` | `string` | ProjectID identifies the project that the new asset belongs to. It is a UUID. |  |
| `name` | `string` | Name is the new asset's name. |  |
| `labels` | `repeated phemera.authority.v1.CreateAssetResponse.LabelsEntry` | Labels are the keys and values that a role selects the new asset by. |  |
| `ssh` | `phemera.authority.v1.CreateAssetResponse.SSHAsset` | SSH is set for a host that phemera reaches over SSH. |  |

### phemera.authority.v1.CreateAssetResponse.LabelsEntry { #phemera.authority.v1.CreateAssetResponse.LabelsEntry }



| Field | Type | Description | Rules |
| ----- | ---- | ----------- | ----- |
| `key` | `string` |  |  |
| `value` | `string` |  |  |

### phemera.authority.v1.CreateAssetResponse.SSHAsset { #phemera.authority.v1.CreateAssetResponse.SSHAsset }

SSHAsset is an SSH host and the user that phemera logs in as.

| Field | Type | Description | Rules |
| ----- | ---- | ----------- | ----- |
| `target` | `phemera.authority.v1.SSHTarget` | Target is the host's address and its host key. |  |
| `user_name` | `string` | UserName is the user that phemera logs in as on the host. |  |

### phemera.authority.v1.ListAssetsRequest { #phemera.authority.v1.ListAssetsRequest }

ListAssetsRequest is the input of AssetService.ListAssets.

| Field | Type | Description | Rules |
| ----- | ---- | ----------- | ----- |
| `project_name` | `string` | ProjectName is the name of the project whose assets are listed. | Starts with a lowercase letter, goes on with lowercase letters, digits and dashes, and ends with a letter or digit. At most 63 characters. |

### phemera.authority.v1.ListAssetsResponse { #phemera.authority.v1.ListAssetsResponse }

ListAssetsResponse is the result of AssetService.ListAssets.

| Field | Type | Description | Rules |
| ----- | ---- | ----------- | ----- |
| `assets` | `repeated phemera.authority.v1.ListAssetsResponse.Asset` | Assets are all assets of the project. |  |

### phemera.authority.v1.ListAssetsResponse.Asset { #phemera.authority.v1.ListAssetsResponse.Asset }

Asset is one asset in the list.

| Field | Type | Description | Rules |
| ----- | ---- | ----------- | ----- |
| `id` | `string` | ID identifies the asset. It is a UUID. |  |
| `project_id` | `string` | ProjectID identifies the project that the asset belongs to. It is a UUID. |  |
| `name` | `string` | Name is the asset's name. |  |
| `labels` | `repeated phemera.authority.v1.ListAssetsResponse.Asset.LabelsEntry` | Labels are the keys and values that a role selects the asset by. |  |
| `ssh` | `phemera.authority.v1.ListAssetsResponse.SSHAsset` | SSH is set for a host that phemera reaches over SSH. |  |

### phemera.authority.v1.ListAssetsResponse.Asset.LabelsEntry { #phemera.authority.v1.ListAssetsResponse.Asset.LabelsEntry }



| Field | Type | Description | Rules |
| ----- | ---- | ----------- | ----- |
| `key` | `string` |  |  |
| `value` | `string` |  |  |

### phemera.authority.v1.ListAssetsResponse.SSHAsset { #phemera.authority.v1.ListAssetsResponse.SSHAsset }

SSHAsset is an SSH host and the user that phemera logs in as.

| Field | Type | Description | Rules |
| ----- | ---- | ----------- | ----- |
| `target` | `phemera.authority.v1.SSHTarget` | Target is the host's address and its host key. |  |
| `user_name` | `string` | UserName is the user that phemera logs in as on the host. |  |

### phemera.authority.v1.SSHTarget { #phemera.authority.v1.SSHTarget }

SSHTarget is the address of an SSH host and the key that the host proves
its identity with.

| Field | Type | Description | Rules |
| ----- | ---- | ----------- | ----- |
| `host` | `string` | Host is the host's DNS name or IP address. | A DNS name of at most 253 characters, or an IPv4 or IPv6 address without brackets and without a zone. |
| `port` | `uint32` | Port is the TCP port that the host's SSH server listens on. | 1 to 65535. |
| `host_key` | `string` | HostKey is the host's public key as the key type and the key in base64, like "ssh-ed25519 AAAAC3...". A key that phemera can't read fails with invalid_argument. | The key type, one space and the key in base64 with its padding. The key type is ssh-ed25519, ecdsa-sha2-nistp256, ecdsa-sha2-nistp384, ecdsa-sha2-nistp521 or ssh-rsa. At most 8192 characters. |

### phemera.authority.v1.CreateBindingRequest { #phemera.authority.v1.CreateBindingRequest }

CreateBindingRequest is the input of BindingService.CreateBinding.

| Field | Type | Description | Rules |
| ----- | ---- | ----------- | ----- |
| `project_name` | `string` | ProjectName is the name of the project that gets the binding. | Starts with a lowercase letter, goes on with lowercase letters, digits and dashes, and ends with a letter or digit. At most 63 characters. |
| `role_name` | `string` | RoleName is the name of the role that the user gets. The role must belong to the project. | Starts with a lowercase letter, goes on with lowercase letters, digits and dashes, and ends with a letter or digit. At most 63 characters. |
| `user_name` | `string` | UserName is the name of the user who gets the role. Uppercase and lowercase letters count as the same. | Not empty. |

### phemera.authority.v1.CreateBindingResponse { #phemera.authority.v1.CreateBindingResponse }

CreateBindingResponse is the result of BindingService.CreateBinding.

| Field | Type | Description | Rules |
| ----- | ---- | ----------- | ----- |
| `id` | `string` | ID identifies the new binding. It is a UUID. |  |
| `project_id` | `string` | ProjectID identifies the project that the new binding belongs to. It is a UUID. |  |
| `role_name` | `string` | RoleName is the name of the role that the new binding gives. |  |
| `user_name` | `string` | UserName is the name of the user who gets the role, as the authority stores it. |  |

### phemera.authority.v1.ListBindingsRequest { #phemera.authority.v1.ListBindingsRequest }

ListBindingsRequest is the input of BindingService.ListBindings.

| Field | Type | Description | Rules |
| ----- | ---- | ----------- | ----- |
| `project_name` | `string` | ProjectName is the name of the project whose bindings are listed. | Starts with a lowercase letter, goes on with lowercase letters, digits and dashes, and ends with a letter or digit. At most 63 characters. |

### phemera.authority.v1.ListBindingsResponse { #phemera.authority.v1.ListBindingsResponse }

ListBindingsResponse is the result of BindingService.ListBindings.

| Field | Type | Description | Rules |
| ----- | ---- | ----------- | ----- |
| `bindings` | `repeated phemera.authority.v1.ListBindingsResponse.Binding` | Bindings are all bindings of the project, ordered by the role name and then the user name. |  |

### phemera.authority.v1.ListBindingsResponse.Binding { #phemera.authority.v1.ListBindingsResponse.Binding }

Binding is one binding in the list.

| Field | Type | Description | Rules |
| ----- | ---- | ----------- | ----- |
| `id` | `string` | ID identifies the binding. It is a UUID. |  |
| `project_id` | `string` | ProjectID identifies the project that the binding belongs to. It is a UUID. |  |
| `role_name` | `string` | RoleName is the name of the role that the binding gives. |  |
| `user_name` | `string` | UserName is the name of the user who has the role. |  |

### phemera.authority.v1.LoginRequest { #phemera.authority.v1.LoginRequest }

LoginRequest is the input of LoginService.Login.

| Field | Type | Description | Rules |
| ----- | ---- | ----------- | ----- |
| `user_name` | `string` | UserName is the user's name, in any letter case. | Not empty. |
| `password` | `string` | Password is the user's local password. | Not empty, at most 1024 bytes. |

### phemera.authority.v1.LoginResponse { #phemera.authority.v1.LoginResponse }

LoginResponse is the result of LoginService.Login.

| Field | Type | Description | Rules |
| ----- | ---- | ----------- | ----- |
| `token` | `string` | Token is valid for the authority's token lifetime, 12 hours by default. |  |

### phemera.authority.v1.LogoutRequest { #phemera.authority.v1.LogoutRequest }

LogoutRequest is the input of LoginService.Logout.

| Field | Type | Description | Rules |
| ----- | ---- | ----------- | ----- |
| `all_sessions` | `bool` | AllSessions revokes every session token of the caller's user, not only the one the call carries. |  |

### phemera.authority.v1.LogoutResponse { #phemera.authority.v1.LogoutResponse }

LogoutResponse is the result of LoginService.Logout.

### phemera.authority.v1.CreateProjectRequest { #phemera.authority.v1.CreateProjectRequest }

CreateProjectRequest is the input of ProjectService.CreateProject.

| Field | Type | Description | Rules |
| ----- | ---- | ----------- | ----- |
| `name` | `string` | Name is the name of the new project. | Starts with a lowercase letter, goes on with lowercase letters, digits and dashes, and ends with a letter or digit. At most 63 characters. |

### phemera.authority.v1.CreateProjectResponse { #phemera.authority.v1.CreateProjectResponse }

CreateProjectResponse is the result of ProjectService.CreateProject.

| Field | Type | Description | Rules |
| ----- | ---- | ----------- | ----- |
| `id` | `string` | ID identifies the new project. It is a UUID. |  |
| `name` | `string` | Name is the new project's name. |  |

### phemera.authority.v1.ListProjectsRequest { #phemera.authority.v1.ListProjectsRequest }

ListProjectsRequest is the input of ProjectService.ListProjects.

### phemera.authority.v1.ListProjectsResponse { #phemera.authority.v1.ListProjectsResponse }

ListProjectsResponse is the result of ProjectService.ListProjects.

| Field | Type | Description | Rules |
| ----- | ---- | ----------- | ----- |
| `projects` | `repeated phemera.authority.v1.ListProjectsResponse.Project` | Projects are all projects. |  |

### phemera.authority.v1.ListProjectsResponse.Project { #phemera.authority.v1.ListProjectsResponse.Project }

Project is one project in the list.

| Field | Type | Description | Rules |
| ----- | ---- | ----------- | ----- |
| `id` | `string` | ID identifies the project. It is a UUID. |  |
| `name` | `string` | Name is the project's name. |  |

### phemera.authority.v1.CreateRoleRequest { #phemera.authority.v1.CreateRoleRequest }

CreateRoleRequest is the input of RoleService.CreateRole.

| Field | Type | Description | Rules |
| ----- | ---- | ----------- | ----- |
| `project_name` | `string` | ProjectName is the name of the project that gets the role. | Starts with a lowercase letter, goes on with lowercase letters, digits and dashes, and ends with a letter or digit. At most 63 characters. |
| `name` | `string` | Name is the name of the new role. | Starts with a lowercase letter, goes on with lowercase letters, digits and dashes, and ends with a letter or digit. At most 63 characters. |
| `permissions` | `phemera.authority.v1.ProjectPermissions` | Permissions are the permissions of the new role. A role without them gives its holders nothing. |  |

### phemera.authority.v1.CreateRoleResponse { #phemera.authority.v1.CreateRoleResponse }

CreateRoleResponse is the result of RoleService.CreateRole.

| Field | Type | Description | Rules |
| ----- | ---- | ----------- | ----- |
| `id` | `string` | ID identifies the new role. It is a UUID. |  |
| `project_id` | `string` | ProjectID identifies the project that the new role belongs to. It is a UUID. |  |
| `name` | `string` | Name is the new role's name. |  |
| `permissions` | `phemera.authority.v1.ProjectPermissions` | Permissions say what the users who have the new role may do. |  |

### phemera.authority.v1.ProjectPermissions { #phemera.authority.v1.ProjectPermissions }

ProjectPermissions says what the holders of a role may do with the roles
and bindings of its project.

| Field | Type | Description | Rules |
| ----- | ---- | ----------- | ----- |
| `create_roles` | `bool` | CreateRoles means the holders may create roles in the project. |  |
| `escalate` | `bool` | Escalate means the holders may create roles with permissions that they don't have themselves. |  |
| `create_bindings` | `bool` | CreateBindings means the holders may create bindings in the project. |  |
| `bind` | `bool` | Bind means the holders may bind roles with permissions that they don't have themselves. |  |

### phemera.authority.v1.CreateTicketRequest { #phemera.authority.v1.CreateTicketRequest }

CreateTicketRequest is the input of TicketService.CreateTicket.

| Field | Type | Description | Rules |
| ----- | ---- | ----------- | ----- |
| `asset_id` | `string` | AssetID identifies the asset to connect to. It is a UUID. |  |

### phemera.authority.v1.CreateTicketResponse { #phemera.authority.v1.CreateTicketResponse }

CreateTicketResponse is the result of TicketService.CreateTicket.

| Field | Type | Description | Rules |
| ----- | ---- | ----------- | ----- |
| `id` | `string` | ID identifies the ticket. It is a UUID. |  |
| `asset_id` | `string` | AssetID identifies the requested asset. It is a UUID. |  |
| `user_id` | `string` | UserID identifies the signed-in user. It is a UUID. |  |

