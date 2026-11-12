bats_require_minimum_version 1.5.0

load database

setup() {
    create_database
}

teardown() {
    drop_database
}

@test "authority says to run authority migrate when the database has no schema" {
    TIDEGATE_AUTHORITY__DATABASE__MIGRATE_ON_START=false run --separate-stderr authority
    [ "$status" -eq 1 ]
    [[ $stderr == *"Run authority migrate first"* ]]
}
