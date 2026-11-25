psql=(psql --quiet --no-psqlrc --set ON_ERROR_STOP=1)

# create_database gives a test an empty database of its own and points the
# authority at it.
create_database() {
    database=test_$(od -An -N8 -tx1 /dev/urandom | tr -d ' \n')
    "${psql[@]}" "$PHEMERA_TEST_DATABASE_URL" --command "CREATE DATABASE $database"
    export PHEMERA_AUTHORITY__DATABASE__URL="$PHEMERA_TEST_DATABASE_SERVER/$database$PHEMERA_TEST_DATABASE_QUERY"
}

drop_database() {
    "${psql[@]}" "$PHEMERA_TEST_DATABASE_URL" --command "DROP DATABASE $database WITH (FORCE)"
}
