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

if ! openssl verify -CAfile "$directory/ca.pem" "$directory/authority.pem" >/dev/null 2>&1; then
    openssl req -newkey ec -pkeyopt ec_paramgen_curve:P-256 -nodes -subj "/CN=localhost" \
        -keyout "$directory/authority-key.pem" -out "$directory/authority.csr" 2>/dev/null
    printf 'basicConstraints=critical,CA:FALSE\nsubjectAltName=DNS:localhost,IP:127.0.0.1,IP:::1\nextendedKeyUsage=serverAuth\n' \
        > "$directory/authority.ext"
    openssl x509 -req -days 3650 -in "$directory/authority.csr" \
        -CA "$directory/ca.pem" -CAkey "$directory/ca-key.pem" -CAcreateserial \
        -extfile "$directory/authority.ext" -out "$directory/authority.pem" 2>/dev/null
    rm -f "$directory/authority.csr" "$directory/authority.ext" "$directory/ca.srl"
fi
