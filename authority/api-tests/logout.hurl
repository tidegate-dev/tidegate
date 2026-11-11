# Clients can't tell a missing token from a wrong one.
POST {{authority}}/tidegate.authority.v1.LoginService/Logout
Content-Type: application/json
{}
HTTP 401
[Asserts]
jsonpath "$.code" == "unauthenticated"
jsonpath "$.message" == "not signed in"

POST {{authority}}/tidegate.authority.v1.LoginService/Logout
Content-Type: application/json
Authorization: Basic YWxpY2U6c2VjcmV0
{}
HTTP 401
[Asserts]
jsonpath "$.code" == "unauthenticated"
jsonpath "$.message" == "not signed in"

POST {{authority}}/tidegate.authority.v1.LoginService/Logout
Content-Type: application/json
Authorization: Bearer unknown
{}
HTTP 401
[Asserts]
jsonpath "$.code" == "unauthenticated"
jsonpath "$.message" == "not signed in"
