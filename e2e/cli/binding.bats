bats_require_minimum_version 1.7.0

setup() {
    load common
    isolate
    # The tests share one database, so each one creates a project of its own.
    project=bindings-$BATS_TEST_NUMBER
}

@test "binding create gives the admin a role, and binding list shows the binding" {
    sign_in login "$TIDEGATE_TEST_AUTHORITY"
    tidegate project create "$project"
    tidegate role create --project-name "$project" operator

    run tidegate binding create --project-name "$project" --role-name operator --user-name "$TIDEGATE_TEST_USER"
    [ "$status" -eq 0 ]
    [[ $output == operator$'\t'$TIDEGATE_TEST_USER$'\t'?* ]]
    id=${output##*$'\t'}

    run tidegate binding list --project-name "$project"
    [ "$status" -eq 0 ]
    [[ ${lines[0]} == ROLE*USER*ID ]]
    grep -qE "^operator +$TIDEGATE_TEST_USER +$id\$" <<<"$output"
}

@test "binding create and binding list print json" {
    sign_in login "$TIDEGATE_TEST_AUTHORITY"
    tidegate project create "$project"
    tidegate role create --project-name "$project" operator
    project_id=$(tidegate -o json project list | jq -r --arg name "$project" '.projects[] | select(.name == $name) | .id')

    run tidegate -o json binding create --project-name "$project" --role-name operator --user-name "$TIDEGATE_TEST_USER"
    [ "$status" -eq 0 ]
    [ "$(jq -r .projectId <<<"$output")" = "$project_id" ]
    [ "$(jq -r .role <<<"$output")" = operator ]
    [ "$(jq -r .user <<<"$output")" = "$TIDEGATE_TEST_USER" ]
    id=$(jq -r .id <<<"$output")

    run tidegate -o json binding list --project-name "$project"
    [ "$status" -eq 0 ]
    [ "$(jq -r '.bindings | length' <<<"$output")" = 1 ]
    [ "$(jq -r '.bindings[0].id' <<<"$output")" = "$id" ]
    [ "$(jq -r '.bindings[0].role' <<<"$output")" = operator ]
    [ "$(jq -r '.bindings[0].user' <<<"$output")" = "$TIDEGATE_TEST_USER" ]
}

@test "binding list of a project without bindings says how to add one" {
    sign_in login "$TIDEGATE_TEST_AUTHORITY"
    tidegate project create "$project"

    run tidegate binding list --project-name "$project"
    [ "$status" -eq 0 ]
    [[ $output == *"No bindings in the project"* ]]
}

@test "binding list shows only the bindings of the project" {
    sign_in login "$TIDEGATE_TEST_AUTHORITY"
    tidegate project create "$project"
    tidegate project create "$project-other"
    tidegate role create --project-name "$project-other" operator
    tidegate binding create --project-name "$project-other" --role-name operator --user-name "$TIDEGATE_TEST_USER"

    run tidegate -o json binding list --project-name "$project"
    [ "$status" -eq 0 ]
    [ "$(jq -r '.bindings | length' <<<"$output")" = 0 ]
}

@test "binding create fails for a role that the user already has" {
    sign_in login "$TIDEGATE_TEST_AUTHORITY"
    tidegate project create "$project"
    tidegate role create --project-name "$project" operator
    tidegate binding create --project-name "$project" --role-name operator --user-name "$TIDEGATE_TEST_USER"

    run --separate-stderr tidegate binding create --project-name "$project" --role-name operator --user-name "$TIDEGATE_TEST_USER"
    [ "$status" -eq 1 ]
    [ -z "$output" ]
    [[ $stderr == *"the user already has the role"* ]]
}

@test "binding create fails for a project, a role or a user that doesn't exist" {
    sign_in login "$TIDEGATE_TEST_AUTHORITY"

    run --separate-stderr tidegate binding create --project-name "$project" --role-name operator --user-name "$TIDEGATE_TEST_USER"
    [ "$status" -eq 1 ]
    [ -z "$output" ]
    [[ $stderr == *"project not found"* ]]

    tidegate project create "$project"

    run --separate-stderr tidegate binding create --project-name "$project" --role-name operator --user-name "$TIDEGATE_TEST_USER"
    [ "$status" -eq 1 ]
    [ -z "$output" ]
    [[ $stderr == *"role not found"* ]]

    tidegate role create --project-name "$project" operator

    run --separate-stderr tidegate binding create --project-name "$project" --role-name operator --user-name nobody
    [ "$status" -eq 1 ]
    [ -z "$output" ]
    [[ $stderr == *"user not found"* ]]
}

@test "binding list fails for a project that doesn't exist" {
    sign_in login "$TIDEGATE_TEST_AUTHORITY"

    run --separate-stderr tidegate binding list --project-name "$project"
    [ "$status" -eq 1 ]
    [ -z "$output" ]
    [[ $stderr == *"project not found"* ]]
}

@test "binding create wants a project, a role, a user and no arguments" {
    run --separate-stderr tidegate binding create --role-name operator --user-name alice
    [ "$status" -eq 1 ]
    [[ $stderr == *"want a project, name it with --project-name"* ]]

    run --separate-stderr tidegate binding create --project-name "$project" --user-name alice
    [ "$status" -eq 1 ]
    [[ $stderr == *"want a role, name it with --role-name"* ]]

    run --separate-stderr tidegate binding create --project-name "$project" --role-name operator
    [ "$status" -eq 1 ]
    [[ $stderr == *"want a user, name them with --user-name"* ]]

    run --separate-stderr tidegate binding create --project-name "$project" --role-name operator --user-name alice extra
    [ "$status" -eq 1 ]
    [[ $stderr == *"want no arguments"* ]]
}

@test "binding list wants a project and no arguments" {
    run --separate-stderr tidegate binding list
    [ "$status" -eq 1 ]
    [[ $stderr == *"want a project, name it with --project-name"* ]]

    run --separate-stderr tidegate binding list --project-name "$project" extra
    [ "$status" -eq 1 ]
    [[ $stderr == *"want no arguments"* ]]
}
