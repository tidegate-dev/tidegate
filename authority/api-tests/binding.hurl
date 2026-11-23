POST {{authority}}/tidegate.authority.v1.BindingService/CreateBinding
Content-Type: application/json
{"project": "api-tests-bindings", "role": "operator", "user": "{{user}}"}
HTTP 401
[Asserts]
jsonpath "$.code" == "unauthenticated"
jsonpath "$.message" == "not signed in"

POST {{authority}}/tidegate.authority.v1.BindingService/ListBindings
Content-Type: application/json
{"project": "api-tests-bindings"}
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

POST {{authority}}/tidegate.authority.v1.BindingService/CreateBinding
Content-Type: application/json
Authorization: Bearer {{token}}
{"project": "api-tests-bindings", "role": "operator", "user": "{{user}}"}
HTTP 404
[Asserts]
jsonpath "$.code" == "not_found"
jsonpath "$.message" == "project not found"

POST {{authority}}/tidegate.authority.v1.BindingService/ListBindings
Content-Type: application/json
Authorization: Bearer {{token}}
{"project": "api-tests-bindings"}
HTTP 404
[Asserts]
jsonpath "$.code" == "not_found"
jsonpath "$.message" == "project not found"

POST {{authority}}/tidegate.authority.v1.ProjectService/CreateProject
Content-Type: application/json
Authorization: Bearer {{token}}
{"name": "api-tests-bindings"}
HTTP 200
[Captures]
project_id: jsonpath "$.id"

POST {{authority}}/tidegate.authority.v1.BindingService/CreateBinding
Content-Type: application/json
Authorization: Bearer {{token}}
{"project": "api-tests-bindings", "role": "operator", "user": "{{user}}"}
HTTP 404
[Asserts]
jsonpath "$.code" == "not_found"
jsonpath "$.message" == "role not found"

POST {{authority}}/tidegate.authority.v1.RoleService/CreateRole
Content-Type: application/json
Authorization: Bearer {{token}}
{"project": "api-tests-bindings", "name": "operator"}
HTTP 200

POST {{authority}}/tidegate.authority.v1.BindingService/CreateBinding
Content-Type: application/json
Authorization: Bearer {{token}}
{"project": "api-tests-bindings", "role": "operator", "user": "nobody"}
HTTP 404
[Asserts]
jsonpath "$.code" == "not_found"
jsonpath "$.message" == "user not found"

POST {{authority}}/tidegate.authority.v1.BindingService/CreateBinding
Content-Type: application/json
Authorization: Bearer {{token}}
{"project": "api-tests-bindings", "role": "Operator", "user": "{{user}}"}
HTTP 400
[Asserts]
jsonpath "$.code" == "invalid_argument"
jsonpath "$.message" contains "must start with a lowercase letter"

POST {{authority}}/tidegate.authority.v1.BindingService/CreateBinding
Content-Type: application/json
Authorization: Bearer {{token}}
{"project": "api-tests-bindings", "role": "operator", "user": "{{user}}"}
HTTP 200
[Captures]
binding_id: jsonpath "$.id"
[Asserts]
jsonpath "$.id" matches "^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$"
jsonpath "$.projectId" == "{{project_id}}"
jsonpath "$.role" == "operator"
jsonpath "$.user" == "{{user}}"

POST {{authority}}/tidegate.authority.v1.BindingService/CreateBinding
Content-Type: application/json
Authorization: Bearer {{token}}
{"project": "api-tests-bindings", "role": "operator", "user": "{{user}}"}
HTTP 409
[Asserts]
jsonpath "$.code" == "already_exists"
jsonpath "$.message" == "the user already has the role"

POST {{authority}}/tidegate.authority.v1.BindingService/ListBindings
Content-Type: application/json
Authorization: Bearer {{token}}
{"project": "api-tests-bindings"}
HTTP 200
[Asserts]
jsonpath "$.bindings" count == 1
jsonpath "$.bindings[0].id" == "{{binding_id}}"
jsonpath "$.bindings[0].projectId" == "{{project_id}}"
jsonpath "$.bindings[0].role" == "operator"
jsonpath "$.bindings[0].user" == "{{user}}"
