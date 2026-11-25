# PingService was removed. Its procedure is unknown, like any other.
POST {{authority}}/tidegate.authority.v1.PingService/Ping
Content-Type: application/json
{}
HTTP 404

# DummyService was removed too.
POST {{authority}}/tidegate.authority.v1.DummyService/Write
Content-Type: application/json
{"text": "hello"}
HTTP 404

POST {{authority}}/tidegate.authority.v1.DummyService/Read
Content-Type: application/json
{"id": 1}
HTTP 404
