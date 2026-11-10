POST {{authority}}/tidegate.authority.v1.PingService/Ping
Content-Type: application/json
{}
HTTP 200
[Asserts]
jsonpath "$.message" == "pong"
