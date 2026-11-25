bats_require_minimum_version 1.7.0

setup() {
    load common
    isolate
}

@test "tidegate refuses ping, which was removed" {
    run --separate-stderr tidegate ping
    [ "$status" -eq 1 ]
    [[ $stderr == *'unknown command "ping"'* ]]
}

@test "tidegate refuses dummy, which was removed" {
    run --separate-stderr tidegate dummy write hello
    [ "$status" -eq 1 ]
    [[ $stderr == *'unknown command "dummy"'* ]]
}
