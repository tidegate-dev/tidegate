bats_require_minimum_version 1.5.0

@test "login refuses a plain HTTP server that isn't on this machine" {
    run --separate-stderr tidegate --server http://tidegate.example login < /dev/null
    [ "$status" -eq 1 ]
    [[ $stderr == *"refusing to send a password or a token over plain HTTP"* ]]
}

@test "a call refuses to send the session token over plain HTTP to another machine" {
    TIDEGATE_TOKEN=secret run --separate-stderr tidegate --server http://tidegate.example dummy write hello
    [ "$status" -eq 1 ]
    [[ $stderr == *"refusing to send a password or a token over plain HTTP"* ]]
}

@test "an authority that can't be reached is named in the error" {
    run --separate-stderr tidegate --server https://127.0.0.1:9 ping
    [ "$status" -eq 1 ]
    [[ $stderr == *"can't reach the authority at https://127.0.0.1:9"* ]]
}
