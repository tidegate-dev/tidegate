POST {{authority}}/tidegate.authority.v1.RoleService/CreateRole
Content-Type: application/json
{"project": "api-tests-roles", "name": "operator"}
HTTP 401
[Asserts]
jsonpath "$.code" == "unauthenticated"
jsonpath "$.message" == "not signed in"

POST {{authority}}/tidegate.authority.v1.LoginService/Login
Content-Type: application/json
{"userName": "{{user}}", "password": "{{password}}"}
HTTP 200
[Captures]
token: jsonpath "$.token"

POST {{authority}}/tidegate.authority.v1.RoleService/CreateRole
Content-Type: application/json
Authorization: Bearer {{token}}
{"project": "api-tests-roles", "name": "operator"}
HTTP 404
[Asserts]
jsonpath "$.code" == "not_found"
jsonpath "$.message" == "project not found"

POST {{authority}}/tidegate.authority.v1.ProjectService/CreateProject
Content-Type: application/json
Authorization: Bearer {{token}}
{"name": "api-tests-roles"}
HTTP 200
[Captures]
project_id: jsonpath "$.project.id"

POST {{authority}}/tidegate.authority.v1.RoleService/CreateRole
Content-Type: application/json
Authorization: Bearer {{token}}
{"project": "api-tests-roles", "name": "Operator"}
HTTP 400
[Asserts]
jsonpath "$.code" == "invalid_argument"
jsonpath "$.message" contains "must start with a lowercase letter"

POST {{authority}}/tidegate.authority.v1.RoleService/CreateRole
Content-Type: application/json
Authorization: Bearer {{token}}
{"project": "api-tests-roles", "name": "operator", "permissions": {"createRoles": true, "bind": true}}
HTTP 200
[Asserts]
jsonpath "$.role.id" matches "^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$"
jsonpath "$.role.projectId" == "{{project_id}}"
jsonpath "$.role.name" == "operator"
jsonpath "$.role.permissions.createRoles" == true
jsonpath "$.role.permissions.bind" == true
jsonpath "$.role.permissions.escalate" not exists
jsonpath "$.role.permissions.createBindings" not exists

POST {{authority}}/tidegate.authority.v1.RoleService/CreateRole
Content-Type: application/json
Authorization: Bearer {{token}}
{"project": "api-tests-roles", "name": "operator"}
HTTP 409
[Asserts]
jsonpath "$.code" == "already_exists"
jsonpath "$.message" == "the role name is taken"
