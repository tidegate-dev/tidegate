setup() {
    load common
    isolate
    export TIDEGATE_SERVER=$TIDEGATE_TEST_AUTHORITY
    TIDEGATE_TOKEN=$(print_token)
    export TIDEGATE_TOKEN
    id=$(tidegate dummy write hello)
}

@test "text output is the bare result" {
    run tidegate dummy read "$id"
    [ "$status" -eq 0 ]
    [ "$output" = hello ]
}

@test "json output is an object with the result's fields" {
    run tidegate --output json dummy read "$id"
    [ "$status" -eq 0 ]
    [ "$(jq -r .text <<<"$output")" = hello ]
}

@test "yaml output is a mapping with the result's fields" {
    run tidegate -o yaml dummy read "$id"
    [ "$status" -eq 0 ]
    [ "$output" = "text: hello" ]
}

@test "an unknown output format fails" {
    run tidegate -o xml dummy read "$id"
    [ "$status" -eq 1 ]
    [[ $output == *'unknown output format "xml"'* ]]
}
