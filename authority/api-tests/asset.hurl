POST {{authority}}/phemera.authority.v1.AssetService/CreateAsset
Content-Type: application/json
{"projectName": "api-tests-assets", "name": "bastion", "ssh": {"target": {"host": "bastion.example", "port": 22, "hostKey": "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIGVtcHR5"}, "login": {"userName": "deploy", "password": "the password of deploy"}}}
HTTP 401
[Asserts]
jsonpath "$.code" == "unauthenticated"
jsonpath "$.message" == "not signed in"

POST {{authority}}/phemera.authority.v1.AssetService/ListAssets
Content-Type: application/json
{"projectName": "api-tests-assets"}
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

POST {{authority}}/phemera.authority.v1.AssetService/CreateAsset
Content-Type: application/json
Authorization: Bearer {{token}}
{"projectName": "api-tests-assets", "name": "bastion", "ssh": {"target": {"host": "bastion.example", "port": 22, "hostKey": "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIGVtcHR5"}, "login": {"userName": "deploy", "password": "the password of deploy"}}}
HTTP 404
[Asserts]
jsonpath "$.code" == "not_found"
jsonpath "$.message" == "project not found"

POST {{authority}}/phemera.authority.v1.AssetService/ListAssets
Content-Type: application/json
Authorization: Bearer {{token}}
{"projectName": "api-tests-assets"}
HTTP 404
[Asserts]
jsonpath "$.code" == "not_found"
jsonpath "$.message" == "project not found"

POST {{authority}}/phemera.authority.v1.ProjectService/CreateProject
Content-Type: application/json
Authorization: Bearer {{token}}
{"name": "api-tests-assets"}
HTTP 200
[Captures]
project_id: jsonpath "$.id"

POST {{authority}}/phemera.authority.v1.AssetService/CreateAsset
Content-Type: application/json
Authorization: Bearer {{token}}
{"projectName": "api-tests-assets", "name": "bastion"}
HTTP 400
[Asserts]
jsonpath "$.code" == "invalid_argument"

POST {{authority}}/phemera.authority.v1.AssetService/CreateAsset
Content-Type: application/json
Authorization: Bearer {{token}}
{"projectName": "api-tests-assets", "name": "bastion", "ssh": {"target": {"host": "bastion.example", "port": 22, "hostKey": "not a key"}, "login": {"userName": "deploy", "password": "the password of deploy"}}}
HTTP 400
[Asserts]
jsonpath "$.code" == "invalid_argument"
jsonpath "$.message" contains "must be the key type"

POST {{authority}}/phemera.authority.v1.AssetService/CreateAsset
Content-Type: application/json
Authorization: Bearer {{token}}
{"projectName": "api-tests-assets", "name": "bastion", "labels": {"env": "prod"}, "ssh": {"target": {"host": "bastion.example", "port": 2222, "hostKey": "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIGVtcHR5"}, "login": {"userName": "deploy", "password": "the password of deploy"}}}
HTTP 200
[Captures]
asset_id: jsonpath "$.id"
[Asserts]
jsonpath "$.id" matches "^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$"
jsonpath "$.projectId" == "{{project_id}}"
jsonpath "$.name" == "bastion"
jsonpath "$.labels.env" == "prod"
jsonpath "$.ssh.target.host" == "bastion.example"
jsonpath "$.ssh.target.port" == 2222
jsonpath "$.ssh.target.hostKey" == "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIGVtcHR5"

POST {{authority}}/phemera.authority.v1.AssetService/CreateAsset
Content-Type: application/json
Authorization: Bearer {{token}}
{"projectName": "api-tests-assets", "name": "bastion", "ssh": {"target": {"host": "other.example", "port": 22, "hostKey": "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIGVtcHR5"}, "login": {"userName": "deploy", "password": "the password of deploy"}}}
HTTP 409
[Asserts]
jsonpath "$.code" == "already_exists"
jsonpath "$.message" == "the asset name is taken"

POST {{authority}}/phemera.authority.v1.AssetService/ListAssets
Content-Type: application/json
Authorization: Bearer {{token}}
{"projectName": "api-tests-assets"}
HTTP 200
[Asserts]
jsonpath "$.assets" count == 1
jsonpath "$.assets[0].id" == "{{asset_id}}"
jsonpath "$.assets[0].name" == "bastion"
jsonpath "$.assets[0].labels.env" == "prod"
jsonpath "$.assets[0].ssh.target.host" == "bastion.example"
jsonpath "$.assets[0].ssh.target.port" == 2222
jsonpath "$.assets[0].ssh.userName" == "deploy"

POST {{authority}}/phemera.authority.v1.AssetService/CreateAsset
Content-Type: application/json
Authorization: Bearer {{token}}
{"projectName": "api-tests-assets", "name": "database", "ssh": {"target": {"host": "database.example", "port": 22, "hostKey": "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIGVtcHR5"}, "login": {"userName": "deploy"}}}
HTTP 400
[Asserts]
jsonpath "$.code" == "invalid_argument"
jsonpath "$.message" contains "ssh.login.secret"

POST {{authority}}/phemera.authority.v1.AssetService/CreateAsset
Content-Type: application/json
Authorization: Bearer {{token}}
{"projectName": "api-tests-assets", "name": "database", "ssh": {"target": {"host": "database.example", "port": 22, "hostKey": "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIGVtcHR5"}, "login": {"password": "the password of deploy"}}}
HTTP 400
[Asserts]
jsonpath "$.code" == "invalid_argument"
jsonpath "$.message" contains "ssh.login.user_name"

POST {{authority}}/phemera.authority.v1.AssetService/CreateAsset
Content-Type: application/json
Authorization: Bearer {{token}}
{"projectName": "api-tests-assets", "name": "database", "ssh": {"target": {"host": "database.example", "port": 22, "hostKey": "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIGVtcHR5"}, "login": {"userName": "deploy", "privateKey": "not a private key"}}}
HTTP 400
[Asserts]
jsonpath "$.code" == "invalid_argument"
jsonpath "$.message" == "the SSH private key isn't a private key that phemera can read"

POST {{authority}}/phemera.authority.v1.AssetService/CreateAsset
Content-Type: application/json
Authorization: Bearer {{token}}
{"projectName": "api-tests-assets", "name": "database", "ssh": {"target": {"host": "database.example", "port": 22, "hostKey": "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIGVtcHR5"}, "login": {"userName": "deploy", "password": "the password of deploy"}}}
HTTP 200
[Asserts]
jsonpath "$.name" == "database"
jsonpath "$.ssh.userName" == "deploy"
body not contains "the password of deploy"

POST {{authority}}/phemera.authority.v1.AssetService/ListAssets
Content-Type: application/json
Authorization: Bearer {{token}}
{"projectName": "api-tests-assets"}
HTTP 200
[Asserts]
jsonpath "$.assets" count == 2
jsonpath "$.assets[1].name" == "database"
jsonpath "$.assets[1].ssh.userName" == "deploy"
