bats_require_minimum_version 1.5.0

load database

setup() {
    create_database
}

teardown() {
    drop_database
}

user_names() {
    "${psql[@]}" --tuples-only --no-align "$TIDEGATE_AUTHORITY__DATABASE__URL" --command "SELECT name FROM users"
}

@test "create-initial-admin creates the user with the password from stdin" {
    authority migrate

    run authority create-initial-admin alice <<<'correct horse battery staple'
    [ "$status" -eq 0 ]
    [ "$(user_names)" = alice ]
}

@test "create-initial-admin doesn't count the line ending as part of the password" {
    authority migrate

    run --separate-stderr authority create-initial-admin alice <<<'fourteen chars'
    [ "$status" -eq 1 ]
    [[ $stderr == *"the password must have at least 15 characters"* ]]
    [ -z "$(user_names)" ]
}

@test "create-initial-admin refuses an invalid name before it reads a password" {
    run --separate-stderr authority create-initial-admin ' alice' < /dev/null
    [ "$status" -eq 1 ]
    [[ $stderr == *"the name must not be empty, start or end with whitespace"* ]]
}

@test "create-initial-admin refuses before it reads a password when a user exists" {
    authority migrate
    authority create-initial-admin alice <<<'correct horse battery staple'

    run --separate-stderr authority create-initial-admin bob < /dev/null
    [ "$status" -eq 1 ]
    [[ $stderr == *"a user already exists"* ]]
}

@test "create-initial-admin says to run authority migrate when the database has no schema" {
    run --separate-stderr authority create-initial-admin alice < /dev/null
    [ "$status" -eq 1 ]
    [[ $stderr == *"Run authority migrate first"* ]]
}
