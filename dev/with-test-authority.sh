#!/usr/bin/env bash
# Usage: with-test-authority.sh <coverage file> <command> [<argument>...]
#
# Starts an authority on its own database in TIDEGATE_TEST_DATABASE_URL,
# creates the initial admin and runs the command. The command finds the
# authority in TIDEGATE_TEST_AUTHORITY, the admin in TIDEGATE_TEST_USER and
# TIDEGATE_TEST_PASSWORD, and the CA in TIDEGATE_TEST_CA_FILE. It finds the
# authority binary on its PATH, and the URL of the test Postgres server
# without a database in TIDEGATE_TEST_DATABASE_SERVER and
# TIDEGATE_TEST_DATABASE_QUERY. Afterwards the authority's coverage is written
# to the coverage file.
set -euo pipefail

if [ $# -lt 2 ]; then
    echo "Usage: $0 <coverage file> <command> [<argument>...]" >&2
    exit 2
fi

coverage_file=$1
shift

root=$(cd "$(dirname "$0")/.." && pwd)
"$root/dev/tls-setup.sh"
mkdir -p "$(dirname "$coverage_file")"

server=${TIDEGATE_TEST_DATABASE_URL%%\?*}
query=${TIDEGATE_TEST_DATABASE_URL#"$server"}
# libpq lets these replace the database in the path.
case $query in *dbname=* | *database=*)
    echo "TIDEGATE_TEST_DATABASE_URL must not name a database in its query." >&2
    exit 1
    ;;
esac
case ${server#*://} in */*) server=${server%/*} ;; esac
database=test_$(od -An -N8 -tx1 /dev/urandom | tr -d ' \n')
psql=(psql --quiet --no-psqlrc --set ON_ERROR_STOP=1 "$TIDEGATE_TEST_DATABASE_URL")
"${psql[@]}" --command "CREATE DATABASE $database"

work=$(mktemp -d)
authority=
cleanup() {
    if [ -n "$authority" ]; then
        kill "$authority" 2>/dev/null || true
        wait "$authority" 2>/dev/null || true
    fi
    rm -rf "$work"
    "${psql[@]}" --command "DROP DATABASE $database WITH (FORCE)"
}
trap cleanup EXIT
fail() {
    cat "$work/log" >&2
    exit 1
}

mkdir "$work/bin" "$work/coverage"
(cd "$root" && go build -cover -coverpkg=github.com/tidegate-dev/tidegate/... -o "$work/bin/authority" ./authority/cmd/authority)
export PATH="$work/bin:$PATH"

# Other ports than just dev, so both can run at the same time.
export GOCOVERDIR="$work/coverage"
export TIDEGATE_AUTHORITY__SERVER__LISTEN_ADDRESS=127.0.0.1:18080
export TIDEGATE_AUTHORITY__METRICS__LISTEN_ADDRESS=127.0.0.1:19464
export TIDEGATE_AUTHORITY__SERVER__TLS__CERT_FILE="$root/dev/tls/authority.pem"
export TIDEGATE_AUTHORITY__SERVER__TLS__KEY_FILE="$root/dev/tls/authority-key.pem"
export TIDEGATE_AUTHORITY__DATABASE__URL="$server/$database$query"
export TIDEGATE_AUTHORITY__DATABASE__MIGRATE_ON_START=true
authority >>"$work/log" 2>&1 &
authority=$!

export TIDEGATE_TEST_AUTHORITY=https://127.0.0.1:18080
export TIDEGATE_TEST_USER=alice
export TIDEGATE_TEST_PASSWORD='correct horse battery staple'
export TIDEGATE_TEST_CA_FILE="$root/dev/tls/ca.pem"
export TIDEGATE_TEST_DATABASE_SERVER=$server
export TIDEGATE_TEST_DATABASE_QUERY=$query

# The authority logs this line only after it migrated the database and opened its port.
listening() {
    jq -e -s 'any(.[]; .msg == "authority listening" and .server == "api")' "$work/log" >/dev/null 2>&1
}
for _ in $(seq 100); do
    listening && break
    kill -0 "$authority" 2>/dev/null || fail
    sleep 0.1
done
listening || fail

printf '%s\n' "$TIDEGATE_TEST_PASSWORD" |
    authority create-initial-admin "$TIDEGATE_TEST_USER" >>"$work/log" 2>&1 || fail

"$@" || fail

# The authority writes its coverage only when it stops cleanly.
kill "$authority"
stopped=$authority
authority=
wait "$stopped" || fail
go tool covdata textfmt -i="$work/coverage" -o "$coverage_file"
