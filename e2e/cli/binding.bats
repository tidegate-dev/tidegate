bats_require_minimum_version 1.7.0

setup() {
    load common
    isolate
    # The tests share one database, so each one creates a project of its own.
    project=bindings-$BATS_TEST_NUMBER
}

@test "binding create gives the admin a role, and binding list shows the binding" {
    sign_in login "$PHEMERA_TEST_AUTHORITY"
    phemera project create "$project"
    phemera role create --project-name "$project" operator

    run phemera binding create --project-name "$project" --role-name operator --user-name "$PHEMERA_TEST_USER"
    [ "$status" -eq 0 ]
    [[ $output == ?*$'\t'operator$'\t'$PHEMERA_TEST_USER ]]
    id=${output%%$'\t'*}

    run phemera binding list --project-name "$project"
    [ "$status" -eq 0 ]
    [[ ${lines[0]} == ID*ROLE*USER ]]
    grep -qE "^$id +operator +$PHEMERA_TEST_USER\$" <<<"$output"
}

@test "binding create and binding list print json" {
    sign_in login "$PHEMERA_TEST_AUTHORITY"
    phemera project create "$project"
    phemera role create --project-name "$project" operator
    project_id=$(phemera -o json project list | jq -r --arg name "$project" '.projects[] | select(.name == $name) | .id')

    run phemera -o json binding create --project-name "$project" --role-name operator --user-name "$PHEMERA_TEST_USER"
    [ "$status" -eq 0 ]
    [ "$(jq -r .projectId <<<"$output")" = "$project_id" ]
    [ "$(jq -r .roleName <<<"$output")" = operator ]
    [ "$(jq -r .userName <<<"$output")" = "$PHEMERA_TEST_USER" ]
    id=$(jq -r .id <<<"$output")

    run phemera -o json binding list --project-name "$project"
    [ "$status" -eq 0 ]
    [ "$(jq -r '.bindings | length' <<<"$output")" = 1 ]
    [ "$(jq -r '.bindings[0].id' <<<"$output")" = "$id" ]
    [ "$(jq -r '.bindings[0].roleName' <<<"$output")" = operator ]
    [ "$(jq -r '.bindings[0].userName' <<<"$output")" = "$PHEMERA_TEST_USER" ]
}

@test "binding list of a project without bindings says how to add one" {
    sign_in login "$PHEMERA_TEST_AUTHORITY"
    phemera project create "$project"

    run phemera binding list --project-name "$project"
    [ "$status" -eq 0 ]
    [[ $output == *"No bindings in the project"* ]]
}

@test "binding list shows only the bindings of the project" {
    sign_in login "$PHEMERA_TEST_AUTHORITY"
    phemera project create "$project"
    phemera project create "$project-other"
    phemera role create --project-name "$project-other" operator
    phemera binding create --project-name "$project-other" --role-name operator --user-name "$PHEMERA_TEST_USER"

    run phemera -o json binding list --project-name "$project"
    [ "$status" -eq 0 ]
    [ "$(jq -r '.bindings | length' <<<"$output")" = 0 ]
}

@test "binding create fails for a role that the user already has" {
    sign_in login "$PHEMERA_TEST_AUTHORITY"
    phemera project create "$project"
    phemera role create --project-name "$project" operator
    phemera binding create --project-name "$project" --role-name operator --user-name "$PHEMERA_TEST_USER"

    run --separate-stderr phemera binding create --project-name "$project" --role-name operator --user-name "$PHEMERA_TEST_USER"
    [ "$status" -eq 1 ]
    [ -z "$output" ]
    [[ $stderr == *"the user already has the role"* ]]
}

@test "binding create fails for a project, a role or a user that doesn't exist" {
    sign_in login "$PHEMERA_TEST_AUTHORITY"

    run --separate-stderr phemera binding create --project-name "$project" --role-name operator --user-name "$PHEMERA_TEST_USER"
    [ "$status" -eq 1 ]
    [ -z "$output" ]
    [[ $stderr == *"project not found"* ]]

    phemera project create "$project"

    run --separate-stderr phemera binding create --project-name "$project" --role-name operator --user-name "$PHEMERA_TEST_USER"
    [ "$status" -eq 1 ]
    [ -z "$output" ]
    [[ $stderr == *"role not found"* ]]

    phemera role create --project-name "$project" operator

    run --separate-stderr phemera binding create --project-name "$project" --role-name operator --user-name nobody
    [ "$status" -eq 1 ]
    [ -z "$output" ]
    [[ $stderr == *"user not found"* ]]
}

@test "binding list fails for a project that doesn't exist" {
    sign_in login "$PHEMERA_TEST_AUTHORITY"

    run --separate-stderr phemera binding list --project-name "$project"
    [ "$status" -eq 1 ]
    [ -z "$output" ]
    [[ $stderr == *"project not found"* ]]
}

@test "binding create wants a project, a role, a user and no arguments" {
    run --separate-stderr phemera binding create --role-name operator --user-name alice
    [ "$status" -eq 1 ]
    [[ $stderr == *'Required flag "project-name" not set'* ]]

    run --separate-stderr phemera binding create --project-name "$project" --user-name alice
    [ "$status" -eq 1 ]
    [[ $stderr == *'Required flag "role-name" not set'* ]]

    run --separate-stderr phemera binding create --project-name "$project" --role-name operator
    [ "$status" -eq 1 ]
    [[ $stderr == *'Required flag "user-name" not set'* ]]

    run --separate-stderr phemera binding create --project-name "$project" --role-name operator --user-name alice extra
    [ "$status" -eq 1 ]
    [[ $stderr == *"want no arguments"* ]]
}

@test "binding list wants a project and no arguments" {
    run --separate-stderr phemera binding list
    [ "$status" -eq 1 ]
    [[ $stderr == *'Required flag "project-name" not set'* ]]

    run --separate-stderr phemera binding list --project-name "$project" extra
    [ "$status" -eq 1 ]
    [[ $stderr == *"want no arguments"* ]]
}
