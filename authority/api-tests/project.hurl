POST {{authority}}/phemera.authority.v1.ProjectService/CreateProject
Content-Type: application/json
{"name": "api-tests"}
HTTP 401
[Asserts]
jsonpath "$.code" == "unauthenticated"
jsonpath "$.message" == "not signed in"

POST {{authority}}/phemera.authority.v1.ProjectService/ListProjects
Content-Type: application/json
{}
HTTP 401
[Asserts]
jsonpath "$.code" == "unauthenticated"
jsonpath "$.message" == "not signed in"

POST {{authority}}/phemera.authority.v1.LoginService/Login
Content-Type: application/json
{"userName": "{{user}}", "password": "{{password}}"}
HTTP 200
[Captures]
token: jsonpath "$.token"

POST {{authority}}/phemera.authority.v1.ProjectService/CreateProject
Content-Type: application/json
Authorization: Bearer {{token}}
{"name": ""}
HTTP 400
[Asserts]
jsonpath "$.code" == "invalid_argument"

POST {{authority}}/phemera.authority.v1.ProjectService/CreateProject
Content-Type: application/json
Authorization: Bearer {{token}}
{"name": "api-tests"}
HTTP 200
[Captures]
project_id: jsonpath "$.id"
[Asserts]
jsonpath "$.name" == "api-tests"

POST {{authority}}/phemera.authority.v1.ProjectService/CreateProject
Content-Type: application/json
Authorization: Bearer {{token}}
{"name": "api-tests"}
HTTP 409
[Asserts]
jsonpath "$.code" == "already_exists"
jsonpath "$.message" == "the project name is taken"

POST {{authority}}/phemera.authority.v1.ProjectService/CreateProject
Content-Type: application/json
Authorization: Bearer {{token}}
{"name": "API-tests"}
HTTP 400
[Asserts]
jsonpath "$.code" == "invalid_argument"
jsonpath "$.message" contains "must start with a lowercase letter"

# 64 characters, one more than a name may have.
POST {{authority}}/phemera.authority.v1.ProjectService/CreateProject
Content-Type: application/json
Authorization: Bearer {{token}}
{"name": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}
HTTP 400
[Asserts]
jsonpath "$.code" == "invalid_argument"

POST {{authority}}/phemera.authority.v1.ProjectService/ListProjects
Content-Type: application/json
Authorization: Bearer {{token}}
{}
HTTP 200
[Asserts]
jsonpath "$.projects[?(@.id == '{{project_id}}')].name" == "api-tests"
