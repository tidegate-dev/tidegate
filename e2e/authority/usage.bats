bats_require_minimum_version 1.5.0

@test "authority --help works without a config" {
    TIDEGATE_AUTHORITY__DATABASE__URL='' run authority --help
    [ "$status" -eq 0 ]
    [[ $output == *USAGE* ]]
}

@test "authority migrate --help works without a config" {
    TIDEGATE_AUTHORITY__DATABASE__URL='' run authority migrate --help
    [ "$status" -eq 0 ]
    [[ $output == *USAGE* ]]
}

@test "authority create-initial-admin --help works without a config" {
    TIDEGATE_AUTHORITY__DATABASE__URL='' run authority create-initial-admin --help
    [ "$status" -eq 0 ]
    [[ $output == *USAGE* ]]
}

@test "authority refuses an unknown command" {
    run --separate-stderr authority bogus
    [ "$status" -eq 1 ]
    [[ $stderr == *'unknown command'*bogus* ]]
}

@test "authority migrate refuses an argument" {
    run --separate-stderr authority migrate bogus
    [ "$status" -eq 1 ]
    [[ $stderr == *'unknown command'*bogus* ]]
}

@test "authority create-initial-admin without a name fails" {
    run --separate-stderr authority create-initial-admin < /dev/null
    [ "$status" -eq 1 ]
    [[ $stderr == *"create-initial-admin takes exactly one argument, the user's name"* ]]
}

@test "authority create-initial-admin with two names fails" {
    run --separate-stderr authority create-initial-admin alice bob < /dev/null
    [ "$status" -eq 1 ]
    [[ $stderr == *"create-initial-admin takes exactly one argument, the user's name"* ]]
}
