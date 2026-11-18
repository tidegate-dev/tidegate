#!/usr/bin/env sh
# P-256 instead of Ed25519, because browsers don't speak Ed25519 TLS.
set -eu

directory="$(dirname "$0")/tls"
mkdir -p "$directory"

if [ ! -f "$directory/ca.pem" ]; then
    openssl req -x509 -newkey ec -pkeyopt ec_paramgen_curve:P-256 -nodes -days 3650 \
        -subj "/CN=tidegate development CA" \
        -addext "basicConstraints=critical,CA:TRUE" -addext "keyUsage=critical,keyCertSign" \
        -keyout "$directory/ca-key.pem" -out "$directory/ca.pem" 2>/dev/null
fi

# Usage: issue <name>. Writes <name>.pem and <name>-key.pem.
issue() {
    name=$1

    if openssl verify -CAfile "$directory/ca.pem" "$directory/$name.pem" >/dev/null 2>&1; then
        return
    fi

    openssl req -newkey ec -pkeyopt ec_paramgen_curve:P-256 -nodes -subj "/CN=localhost" \
        -keyout "$directory/$name-key.pem" -out "$directory/$name.csr" 2>/dev/null
    printf 'basicConstraints=critical,CA:FALSE\nsubjectAltName=DNS:localhost,IP:127.0.0.1,IP:::1\nextendedKeyUsage=serverAuth\n' \
        > "$directory/$name.ext"
    openssl x509 -req -days 3650 -in "$directory/$name.csr" \
        -CA "$directory/ca.pem" -CAkey "$directory/ca-key.pem" -CAcreateserial \
        -extfile "$directory/$name.ext" -out "$directory/$name.pem" 2>/dev/null
    rm -f "$directory/$name.csr" "$directory/$name.ext" "$directory/ca.srl"
}

issue authority-external-api
issue authority-internal-api
