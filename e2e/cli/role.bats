bats_require_minimum_version 1.7.0

setup() {
    load common
    isolate
    # The tests share one database, so each one creates a project of its own.
    project=roles-$BATS_TEST_NUMBER
}

@test "role create prints the name and the ID of the new role" {
    sign_in login "$TIDEGATE_TEST_AUTHORITY"
    tidegate project create "$project"

    run tidegate role create --project "$project" operator
    [ "$status" -eq 0 ]
    [[ $output == operator$'\t'?* ]]
}

@test "role create gives the role the permission of each flag and no other" {
    sign_in login "$TIDEGATE_TEST_AUTHORITY"
    tidegate project create "$project"

    while read -r flag field; do
        run tidegate -o json role create --project "$project" "$flag" "role$flag"
        [ "$status" -eq 0 ]
        [ "$(jq -c '.permissions | map_values(select(.)) | keys' <<<"$output")" = "[\"$field\"]" ]
    done <<'FLAGS'
--create-roles createRoles
--escalate escalate
--create-bindings createBindings
--bind bind
FLAGS
}

@test "role create without a permission flag creates a role without permissions" {
    sign_in login "$TIDEGATE_TEST_AUTHORITY"
    tidegate project create "$project"
    project_id=$(tidegate -o json project list | jq -r --arg name "$project" '.projects[] | select(.name == $name) | .id')

    run tidegate -o json role create --project "$project" operator
    [ "$status" -eq 0 ]
    [ "$(jq -r .name <<<"$output")" = operator ]
    [ "$(jq -r .projectId <<<"$output")" = "$project_id" ]
    [ "$(jq -c .permissions <<<"$output")" = '{"createRoles":false,"escalate":false,"createBindings":false,"bind":false}' ]
}

@test "role create fails for a name that another role of the project has" {
    sign_in login "$TIDEGATE_TEST_AUTHORITY"
    tidegate project create "$project"
    tidegate role create --project "$project" operator

    run --separate-stderr tidegate role create --project "$project" operator
    [ "$status" -eq 1 ]
    [ -z "$output" ]
    [[ $stderr == *"the role name is taken"* ]]
}

@test "role create fails for a project that doesn't exist" {
    sign_in login "$TIDEGATE_TEST_AUTHORITY"

    run --separate-stderr tidegate role create --project "$project" operator
    [ "$status" -eq 1 ]
    [ -z "$output" ]
    [[ $stderr == *"project not found"* ]]
}

@test "role create wants exactly one name" {
    run --separate-stderr tidegate role create --project "$project"
    [ "$status" -eq 1 ]
    [[ $stderr == *"want exactly one name"* ]]

    run --separate-stderr tidegate role create --project "$project" first second
    [ "$status" -eq 1 ]
    [[ $stderr == *"want exactly one name"* ]]
}

@test "role create wants a project" {
    run --separate-stderr tidegate role create operator
    [ "$status" -eq 1 ]
    [ -z "$output" ]
    [[ $stderr == *"want a project, name it with --project"* ]]
}
