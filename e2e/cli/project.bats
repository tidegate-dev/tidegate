bats_require_minimum_version 1.7.0

setup() {
    load common
    isolate
    # The tests share one database, so each one creates a project of its own.
    name=payments-$BATS_TEST_NUMBER
}

@test "project create prints the new project, and project list shows it" {
    sign_in login "$TIDEGATE_TEST_AUTHORITY"

    run tidegate project create "$name"
    [ "$status" -eq 0 ]
    [[ $output == "$name"$'\t'?* ]]
    id=${output#*$'\t'}

    run tidegate project list
    [ "$status" -eq 0 ]
    [[ ${lines[0]} == NAME*ID ]]
    grep -qE "^$name +$id\$" <<<"$output"
}

@test "project create and project list print json" {
    sign_in login "$TIDEGATE_TEST_AUTHORITY"

    run tidegate -o json project create "$name"
    [ "$status" -eq 0 ]
    [ "$(jq -r .name <<<"$output")" = "$name" ]
    id=$(jq -r .id <<<"$output")

    run tidegate -o json project list
    [ "$status" -eq 0 ]
    [ "$(jq -r --arg id "$id" '.projects[] | select(.id == $id) | .name' <<<"$output")" = "$name" ]
}

@test "project create fails for a name that another project has" {
    sign_in login "$TIDEGATE_TEST_AUTHORITY"
    tidegate project create "$name"

    run --separate-stderr tidegate project create "$name"
    [ "$status" -eq 1 ]
    [ -z "$output" ]
    [[ $stderr == *"the project name is taken"* ]]
}

@test "project create wants exactly one name" {
    run --separate-stderr tidegate project create
    [ "$status" -eq 1 ]
    [[ $stderr == *"want exactly one name"* ]]

    run --separate-stderr tidegate project create first second
    [ "$status" -eq 1 ]
    [[ $stderr == *"want exactly one name"* ]]
}

@test "project list wants no arguments" {
    run --separate-stderr tidegate project list "$name"
    [ "$status" -eq 1 ]
    [ -z "$output" ]
    [[ $stderr == *"want no arguments"* ]]
}

@test "an unknown project subcommand fails and names a similar one" {
    run --separate-stderr tidegate project lsit
    [ "$status" -eq 1 ]
    [[ $stderr == *'unknown command "lsit". Did you mean "list"?'* ]]
}

@test "project without a subcommand prints its subcommands" {
    run tidegate project
    [ "$status" -eq 0 ]
    [[ $output == *create*list* ]]
}
