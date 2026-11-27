bats_require_minimum_version 1.7.0

setup() {
    load common
    isolate
}

@test "connect requires exactly one asset ID in UUID format" {
    run --separate-stderr phemera connect
    [ "$status" -eq 255 ]
    [[ $stderr == *"want exactly one asset ID"* ]]

    run --separate-stderr phemera connect invalid
    [ "$status" -eq 255 ]
    [[ $stderr == *"want an asset ID in UUID format"* ]]
}

@test "connect reports a missing asset" {
    sign_in login "$PHEMERA_TEST_AUTHORITY"
    run --separate-stderr phemera connect 00000000-0000-4000-8000-000000000000
    [ "$status" -eq 255 ]
    [ -z "$output" ]
    [[ $stderr == *"asset not found"* ]]
}
