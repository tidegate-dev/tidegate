setup() {
    load common
    isolate
    export TIDEGATE_SERVER=$TIDEGATE_TEST_AUTHORITY
    TIDEGATE_TOKEN=$(print_token)
    export TIDEGATE_TOKEN
    # The tests share one database, so each one creates a project of its own.
    project=output-$BATS_TEST_NUMBER
}

@test "text output is the result's fields, separated by tabs" {
    run tidegate project create "$project"
    [ "$status" -eq 0 ]
    [[ $output == ?*$'\t'"$project" ]]
}

@test "json output is an object with the result's fields" {
    run tidegate --output json project create "$project"
    [ "$status" -eq 0 ]
    [ "$(jq -r .name <<<"$output")" = "$project" ]
    [ -n "$(jq -r .id <<<"$output")" ]
}

@test "yaml output is a mapping with the result's fields" {
    run tidegate -o yaml project create "$project"
    [ "$status" -eq 0 ]
    [[ ${lines[0]} == "id: "?* ]]
    [ "${lines[1]}" = "name: $project" ]
}

@test "an unknown output format fails" {
    run tidegate -o xml project list
    [ "$status" -eq 1 ]
    [[ $output == *'unknown output format "xml"'* ]]
}
