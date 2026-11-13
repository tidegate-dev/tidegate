setup() {
    export SSL_CERT_FILE=$TIDEGATE_TEST_CA_FILE
}

@test "text output is the bare result" {
    run tidegate --server "$TIDEGATE_TEST_AUTHORITY" ping
    [ "$status" -eq 0 ]
    [ "$output" = pong ]
}

@test "json output is an object with the result's fields" {
    run tidegate --server "$TIDEGATE_TEST_AUTHORITY" --output json ping
    [ "$status" -eq 0 ]
    [ "$(jq -r .message <<<"$output")" = pong ]
}

@test "yaml output is a mapping with the result's fields" {
    run tidegate --server "$TIDEGATE_TEST_AUTHORITY" -o yaml ping
    [ "$status" -eq 0 ]
    [ "$output" = "message: pong" ]
}

@test "an unknown output format fails" {
    run tidegate --server "$TIDEGATE_TEST_AUTHORITY" -o xml ping
    [ "$status" -eq 1 ]
    [[ $output == *'unknown output format "xml"'* ]]
}
