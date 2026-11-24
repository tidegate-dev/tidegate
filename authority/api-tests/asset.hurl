POST {{authority}}/tidegate.authority.v1.AssetService/CreateAsset
Content-Type: application/json
{"projectName": "api-tests-assets", "name": "bastion", "ssh": {"host": "bastion.example", "port": 22, "hostKey": "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIGVtcHR5"}}
HTTP 401
[Asserts]
jsonpath "$.code" == "unauthenticated"
jsonpath "$.message" == "not signed in"

POST {{authority}}/tidegate.authority.v1.AssetService/ListAssets
Content-Type: application/json
{"projectName": "api-tests-assets"}
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

POST {{authority}}/tidegate.authority.v1.AssetService/CreateAsset
Content-Type: application/json
Authorization: Bearer {{token}}
{"projectName": "api-tests-assets", "name": "bastion", "ssh": {"host": "bastion.example", "port": 22, "hostKey": "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIGVtcHR5"}}
HTTP 404
[Asserts]
jsonpath "$.code" == "not_found"
jsonpath "$.message" == "project not found"

POST {{authority}}/tidegate.authority.v1.AssetService/ListAssets
Content-Type: application/json
Authorization: Bearer {{token}}
{"projectName": "api-tests-assets"}
HTTP 404
[Asserts]
jsonpath "$.code" == "not_found"
jsonpath "$.message" == "project not found"

POST {{authority}}/tidegate.authority.v1.ProjectService/CreateProject
Content-Type: application/json
Authorization: Bearer {{token}}
{"name": "api-tests-assets"}
HTTP 200
[Captures]
project_id: jsonpath "$.id"

POST {{authority}}/tidegate.authority.v1.AssetService/CreateAsset
Content-Type: application/json
Authorization: Bearer {{token}}
{"projectName": "api-tests-assets", "name": "bastion"}
HTTP 400
[Asserts]
jsonpath "$.code" == "invalid_argument"

POST {{authority}}/tidegate.authority.v1.AssetService/CreateAsset
Content-Type: application/json
Authorization: Bearer {{token}}
{"projectName": "api-tests-assets", "name": "bastion", "ssh": {"host": "bastion.example", "port": 22, "hostKey": "not a key"}}
HTTP 400
[Asserts]
jsonpath "$.code" == "invalid_argument"
jsonpath "$.message" contains "must be the key type"

POST {{authority}}/tidegate.authority.v1.AssetService/CreateAsset
Content-Type: application/json
Authorization: Bearer {{token}}
{"projectName": "api-tests-assets", "name": "bastion", "labels": {"env": "prod"}, "ssh": {"host": "bastion.example", "port": 2222, "hostKey": "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIGVtcHR5"}}
HTTP 200
[Captures]
asset_id: jsonpath "$.id"
[Asserts]
jsonpath "$.id" matches "^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$"
jsonpath "$.projectId" == "{{project_id}}"
jsonpath "$.name" == "bastion"
jsonpath "$.labels.env" == "prod"
jsonpath "$.ssh.host" == "bastion.example"
jsonpath "$.ssh.port" == 2222
jsonpath "$.ssh.hostKey" == "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIGVtcHR5"

POST {{authority}}/tidegate.authority.v1.AssetService/CreateAsset
Content-Type: application/json
Authorization: Bearer {{token}}
{"projectName": "api-tests-assets", "name": "bastion", "ssh": {"host": "other.example", "port": 22, "hostKey": "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIGVtcHR5"}}
HTTP 409
[Asserts]
jsonpath "$.code" == "already_exists"
jsonpath "$.message" == "the asset name is taken"

POST {{authority}}/tidegate.authority.v1.AssetService/ListAssets
Content-Type: application/json
Authorization: Bearer {{token}}
{"projectName": "api-tests-assets"}
HTTP 200
[Asserts]
jsonpath "$.assets" count == 1
jsonpath "$.assets[0].id" == "{{asset_id}}"
jsonpath "$.assets[0].name" == "bastion"
jsonpath "$.assets[0].labels.env" == "prod"
jsonpath "$.assets[0].ssh.host" == "bastion.example"
jsonpath "$.assets[0].ssh.port" == 2222
