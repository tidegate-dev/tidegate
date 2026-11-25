bats_require_minimum_version 1.7.0

setup() {
    load common
    isolate
}

@test "phemera refuses ping, which was removed" {
    run --separate-stderr phemera ping
    [ "$status" -eq 1 ]
    [[ $stderr == *'unknown command "ping"'* ]]
}

@test "phemera refuses dummy, which was removed" {
    run --separate-stderr phemera dummy write hello
    [ "$status" -eq 1 ]
    [[ $stderr == *'unknown command "dummy"'* ]]
}
