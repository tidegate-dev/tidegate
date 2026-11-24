# API reference

This page is generated from the comments in the `.proto` files in `proto/`. Clients call the API with [Connect](https://connectrpc.com/), gRPC or gRPC-Web.

## tidegate.authority.v1.AssetService { #tidegate.authority.v1.AssetService }

AssetService creates and lists assets. An asset is a target in one project
that tidegate gives access to, like an SSH host.

| Method | Request | Response | Description |
| ------ | ------- | -------- | ----------- |
| `CreateAsset` | [tidegate.authority.v1.CreateAssetRequest](#tidegate.authority.v1.CreateAssetRequest) | [tidegate.authority.v1.CreateAssetResponse](#tidegate.authority.v1.CreateAssetResponse) | CreateAsset creates an asset in a project and returns it. A project that doesn't exist fails with not_found. A name that another asset of the project has fails with already_exists. |
| `ListAssets` | [tidegate.authority.v1.ListAssetsRequest](#tidegate.authority.v1.ListAssetsRequest) | [tidegate.authority.v1.ListAssetsResponse](#tidegate.authority.v1.ListAssetsResponse) | ListAssets returns every asset of a project. A project that doesn't exist fails with not_found. |

## tidegate.authority.v1.BindingService { #tidegate.authority.v1.BindingService }

BindingService creates and lists bindings. A binding gives a role of one
project to a user.

| Method | Request | Response | Description |
| ------ | ------- | -------- | ----------- |
| `CreateBinding` | [tidegate.authority.v1.CreateBindingRequest](#tidegate.authority.v1.CreateBindingRequest) | [tidegate.authority.v1.CreateBindingResponse](#tidegate.authority.v1.CreateBindingResponse) | CreateBinding gives a role of a project to a user. It fails with not_found if the project, the role or the user doesn't exist, and with already_exists if the user already has the role. |
| `ListBindings` | [tidegate.authority.v1.ListBindingsRequest](#tidegate.authority.v1.ListBindingsRequest) | [tidegate.authority.v1.ListBindingsResponse](#tidegate.authority.v1.ListBindingsResponse) | ListBindings returns every binding of a project. It fails with not_found if the project doesn't exist. |

## tidegate.authority.v1.LoginService { #tidegate.authority.v1.LoginService }

LoginService signs users in to the authority.

| Method | Request | Response | Description |
| ------ | ------- | -------- | ----------- |
| `Login` | [tidegate.authority.v1.LoginRequest](#tidegate.authority.v1.LoginRequest) | [tidegate.authority.v1.LoginResponse](#tidegate.authority.v1.LoginResponse) | Login checks a user's local password and returns a session token. All wrong credentials get the same error. |
| `Logout` | [tidegate.authority.v1.LogoutRequest](#tidegate.authority.v1.LogoutRequest) | [tidegate.authority.v1.LogoutResponse](#tidegate.authority.v1.LogoutResponse) | Logout revokes the session token in the call's `Authorization: Bearer` header, so later calls with it fail. A missing, unknown or expired token fails with unauthenticated. |

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

### tidegate.authority.v1.CreateAssetRequest { #tidegate.authority.v1.CreateAssetRequest }

CreateAssetRequest is the input of AssetService.CreateAsset.

| Field | Type | Description | Rules |
| ----- | ---- | ----------- | ----- |
| `project_name` | `string` | ProjectName is the name of the project that gets the asset. | Starts with a lowercase letter, goes on with lowercase letters, digits and dashes, and ends with a letter or digit. At most 63 characters. |
| `name` | `string` | Name is the name of the new asset. | Starts with a lowercase letter, goes on with lowercase letters, digits and dashes, and ends with a letter or digit. At most 63 characters. |
| `labels` | `repeated tidegate.authority.v1.CreateAssetRequest.LabelsEntry` | Labels are the labels of the new asset. | At most 64 labels. A key is an optional DNS name and a slash, followed by a name. The DNS name has at most 253 characters, in lowercase parts of at most 63. A name has at most 63 letters, digits, dashes, underscores and dots, and starts and ends with a letter or digit. A value is empty or follows the rule for a name. |
| `ssh` | `tidegate.authority.v1.SSHTarget` | SSH makes the asset a host that tidegate reaches over SSH. |  |
| `ssh_password` | `string` | SSHPassword is the user's password on the host. | Not empty, at most 1024 bytes. |
| `ssh_private_key` | `string` | SSHPrivateKey is a private key of the user in PEM, without a passphrase. A key that tidegate can't read or that has a passphrase fails with invalid_argument. | Not empty, at most 16384 bytes. |

### tidegate.authority.v1.CreateAssetRequest.LabelsEntry { #tidegate.authority.v1.CreateAssetRequest.LabelsEntry }



| Field | Type | Description | Rules |
| ----- | ---- | ----------- | ----- |
| `key` | `string` |  |  |
| `value` | `string` |  |  |

### tidegate.authority.v1.CreateAssetResponse { #tidegate.authority.v1.CreateAssetResponse }

CreateAssetResponse is the result of AssetService.CreateAsset.

| Field | Type | Description | Rules |
| ----- | ---- | ----------- | ----- |
| `id` | `string` | ID identifies the new asset. It is a UUID. |  |
| `project_id` | `string` | ProjectID identifies the project that the new asset belongs to. It is a UUID. |  |
| `name` | `string` | Name is the new asset's name. |  |
| `labels` | `repeated tidegate.authority.v1.CreateAssetResponse.LabelsEntry` | Labels are the keys and values that a role selects the new asset by. |  |
| `ssh` | `tidegate.authority.v1.SSHTarget` | SSH is set for a host that tidegate reaches over SSH. |  |

### tidegate.authority.v1.CreateAssetResponse.LabelsEntry { #tidegate.authority.v1.CreateAssetResponse.LabelsEntry }



| Field | Type | Description | Rules |
| ----- | ---- | ----------- | ----- |
| `key` | `string` |  |  |
| `value` | `string` |  |  |

### tidegate.authority.v1.ListAssetsRequest { #tidegate.authority.v1.ListAssetsRequest }

ListAssetsRequest is the input of AssetService.ListAssets.

| Field | Type | Description | Rules |
| ----- | ---- | ----------- | ----- |
| `project_name` | `string` | ProjectName is the name of the project whose assets are listed. | Starts with a lowercase letter, goes on with lowercase letters, digits and dashes, and ends with a letter or digit. At most 63 characters. |

### tidegate.authority.v1.ListAssetsResponse { #tidegate.authority.v1.ListAssetsResponse }

ListAssetsResponse is the result of AssetService.ListAssets.

| Field | Type | Description | Rules |
| ----- | ---- | ----------- | ----- |
| `assets` | `repeated tidegate.authority.v1.ListAssetsResponse.Asset` | Assets are all assets of the project. |  |

### tidegate.authority.v1.ListAssetsResponse.Asset { #tidegate.authority.v1.ListAssetsResponse.Asset }

Asset is one asset in the list.

| Field | Type | Description | Rules |
| ----- | ---- | ----------- | ----- |
| `id` | `string` | ID identifies the asset. It is a UUID. |  |
| `project_id` | `string` | ProjectID identifies the project that the asset belongs to. It is a UUID. |  |
| `name` | `string` | Name is the asset's name. |  |
| `labels` | `repeated tidegate.authority.v1.ListAssetsResponse.Asset.LabelsEntry` | Labels are the keys and values that a role selects the asset by. |  |
| `ssh` | `tidegate.authority.v1.SSHTarget` | SSH is set for a host that tidegate reaches over SSH. |  |

### tidegate.authority.v1.ListAssetsResponse.Asset.LabelsEntry { #tidegate.authority.v1.ListAssetsResponse.Asset.LabelsEntry }



| Field | Type | Description | Rules |
| ----- | ---- | ----------- | ----- |
| `key` | `string` |  |  |
| `value` | `string` |  |  |

### tidegate.authority.v1.SSHTarget { #tidegate.authority.v1.SSHTarget }

SSHTarget is the address of an SSH host and the key that the host proves
its identity with.

| Field | Type | Description | Rules |
| ----- | ---- | ----------- | ----- |
| `host` | `string` | Host is the host's DNS name or IP address. | A DNS name of at most 253 characters, or an IPv4 or IPv6 address without brackets and without a zone. |
| `port` | `uint32` | Port is the TCP port that the host's SSH server listens on. | 1 to 65535. |
| `host_key` | `string` | HostKey is the host's public key as the key type and the key in base64, like "ssh-ed25519 AAAAC3...". | The key type, one space and the key in base64 with its padding. The key type is ssh-ed25519, ecdsa-sha2-nistp256, ecdsa-sha2-nistp384, ecdsa-sha2-nistp521 or ssh-rsa. At most 8192 characters. |
| `user_name` | `string` | UserName is the user that tidegate logs in as on the host. It is empty for a host without a login. | At most 64 characters, without spaces and control characters. It doesn't start with a dash. |

### tidegate.authority.v1.CreateBindingRequest { #tidegate.authority.v1.CreateBindingRequest }

CreateBindingRequest is the input of BindingService.CreateBinding.

| Field | Type | Description | Rules |
| ----- | ---- | ----------- | ----- |
| `project_name` | `string` | ProjectName is the name of the project that gets the binding. | Starts with a lowercase letter, goes on with lowercase letters, digits and dashes, and ends with a letter or digit. At most 63 characters. |
| `role_name` | `string` | RoleName is the name of the role that the user gets. The role must belong to the project. | Starts with a lowercase letter, goes on with lowercase letters, digits and dashes, and ends with a letter or digit. At most 63 characters. |
| `user_name` | `string` | UserName is the name of the user who gets the role. Uppercase and lowercase letters count as the same. | Not empty. |

### tidegate.authority.v1.CreateBindingResponse { #tidegate.authority.v1.CreateBindingResponse }

CreateBindingResponse is the result of BindingService.CreateBinding.

| Field | Type | Description | Rules |
| ----- | ---- | ----------- | ----- |
| `id` | `string` | ID identifies the new binding. It is a UUID. |  |
| `project_id` | `string` | ProjectID identifies the project that the new binding belongs to. It is a UUID. |  |
| `role_name` | `string` | RoleName is the name of the role that the new binding gives. |  |
| `user_name` | `string` | UserName is the name of the user who gets the role, as the authority stores it. |  |

### tidegate.authority.v1.ListBindingsRequest { #tidegate.authority.v1.ListBindingsRequest }

ListBindingsRequest is the input of BindingService.ListBindings.

| Field | Type | Description | Rules |
| ----- | ---- | ----------- | ----- |
| `project_name` | `string` | ProjectName is the name of the project whose bindings are listed. | Starts with a lowercase letter, goes on with lowercase letters, digits and dashes, and ends with a letter or digit. At most 63 characters. |

### tidegate.authority.v1.ListBindingsResponse { #tidegate.authority.v1.ListBindingsResponse }

ListBindingsResponse is the result of BindingService.ListBindings.

| Field | Type | Description | Rules |
| ----- | ---- | ----------- | ----- |
| `bindings` | `repeated tidegate.authority.v1.ListBindingsResponse.Binding` | Bindings are all bindings of the project, ordered by the role name and then the user name. |  |

### tidegate.authority.v1.ListBindingsResponse.Binding { #tidegate.authority.v1.ListBindingsResponse.Binding }

Binding is one binding in the list.

| Field | Type | Description | Rules |
| ----- | ---- | ----------- | ----- |
| `id` | `string` | ID identifies the binding. It is a UUID. |  |
| `project_id` | `string` | ProjectID identifies the project that the binding belongs to. It is a UUID. |  |
| `role_name` | `string` | RoleName is the name of the role that the binding gives. |  |
| `user_name` | `string` | UserName is the name of the user who has the role. |  |

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

### tidegate.authority.v1.CreateProjectRequest { #tidegate.authority.v1.CreateProjectRequest }

CreateProjectRequest is the input of ProjectService.CreateProject.

| Field | Type | Description | Rules |
| ----- | ---- | ----------- | ----- |
| `name` | `string` | Name is the name of the new project. | Starts with a lowercase letter, goes on with lowercase letters, digits and dashes, and ends with a letter or digit. At most 63 characters. |

### tidegate.authority.v1.CreateProjectResponse { #tidegate.authority.v1.CreateProjectResponse }

CreateProjectResponse is the result of ProjectService.CreateProject.

| Field | Type | Description | Rules |
| ----- | ---- | ----------- | ----- |
| `id` | `string` | ID identifies the new project. It is a UUID. |  |
| `name` | `string` | Name is the new project's name. |  |

### tidegate.authority.v1.ListProjectsRequest { #tidegate.authority.v1.ListProjectsRequest }

ListProjectsRequest is the input of ProjectService.ListProjects.

### tidegate.authority.v1.ListProjectsResponse { #tidegate.authority.v1.ListProjectsResponse }

ListProjectsResponse is the result of ProjectService.ListProjects.

| Field | Type | Description | Rules |
| ----- | ---- | ----------- | ----- |
| `projects` | `repeated tidegate.authority.v1.ListProjectsResponse.Project` | Projects are all projects. |  |

### tidegate.authority.v1.ListProjectsResponse.Project { #tidegate.authority.v1.ListProjectsResponse.Project }

Project is one project in the list.

| Field | Type | Description | Rules |
| ----- | ---- | ----------- | ----- |
| `id` | `string` | ID identifies the project. It is a UUID. |  |
| `name` | `string` | Name is the project's name. |  |

### tidegate.authority.v1.CreateRoleRequest { #tidegate.authority.v1.CreateRoleRequest }

CreateRoleRequest is the input of RoleService.CreateRole.

| Field | Type | Description | Rules |
| ----- | ---- | ----------- | ----- |
| `project_name` | `string` | ProjectName is the name of the project that gets the role. | Starts with a lowercase letter, goes on with lowercase letters, digits and dashes, and ends with a letter or digit. At most 63 characters. |
| `name` | `string` | Name is the name of the new role. | Starts with a lowercase letter, goes on with lowercase letters, digits and dashes, and ends with a letter or digit. At most 63 characters. |
| `permissions` | `tidegate.authority.v1.ProjectPermissions` | Permissions are the permissions of the new role. A role without them gives its holders nothing. |  |

### tidegate.authority.v1.CreateRoleResponse { #tidegate.authority.v1.CreateRoleResponse }

CreateRoleResponse is the result of RoleService.CreateRole.

| Field | Type | Description | Rules |
| ----- | ---- | ----------- | ----- |
| `id` | `string` | ID identifies the new role. It is a UUID. |  |
| `project_id` | `string` | ProjectID identifies the project that the new role belongs to. It is a UUID. |  |
| `name` | `string` | Name is the new role's name. |  |
| `permissions` | `tidegate.authority.v1.ProjectPermissions` | Permissions say what the users who have the new role may do. |  |

### tidegate.authority.v1.ProjectPermissions { #tidegate.authority.v1.ProjectPermissions }

ProjectPermissions says what the holders of a role may do with the roles
and bindings of its project.

| Field | Type | Description | Rules |
| ----- | ---- | ----------- | ----- |
| `create_roles` | `bool` | CreateRoles means the holders may create roles in the project. |  |
| `escalate` | `bool` | Escalate means the holders may create roles with permissions that they don't have themselves. |  |
| `create_bindings` | `bool` | CreateBindings means the holders may create bindings in the project. |  |
| `bind` | `bool` | Bind means the holders may bind roles with permissions that they don't have themselves. |  |

