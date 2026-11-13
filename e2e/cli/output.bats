setup() {
    load common
    isolate
    export TIDEGATE_SERVER=$TIDEGATE_TEST_AUTHORITY
}

@test "text output is the bare result" {
    run tidegate ping
    [ "$status" -eq 0 ]
    [ "$output" = pong ]
}

@test "json output is an object with the result's fields" {
    run tidegate --output json ping
    [ "$status" -eq 0 ]
    [ "$(jq -r .message <<<"$output")" = pong ]
}

@test "yaml output is a mapping with the result's fields" {
    run tidegate -o yaml ping
    [ "$status" -eq 0 ]
    [ "$output" = "message: pong" ]
}

@test "an unknown output format fails" {
    run tidegate -o xml ping
    [ "$status" -eq 1 ]
    [[ $output == *'unknown output format "xml"'* ]]
}
