bats_require_minimum_version 1.7.0

setup() {
    load common
    isolate
}

@test "connect requests a ticket with the saved session and prints JSON and text" {
    sign_in login "$TIDEGATE_TEST_AUTHORITY"
    project=connect-$BATS_TEST_NUMBER
    tidegate project create "$project"
    host_key=$(cat "$BATS_TEST_DIRNAME/../../dev/ssh-host/ssh_host_ed25519_key.pub")
    run tidegate -o json asset create ssh --project-name "$project" --address bastion.example --host-key "$host_key" bastion
    [ "$status" -eq 0 ]
    asset=$(jq -er .id <<<"$output")

    run tidegate -o json connect "$asset"
    [ "$status" -eq 0 ]
    [ "$(jq -er .assetId <<<"$output")" = "$asset" ]
    ticket=$(jq -er .id <<<"$output")
    user=$(jq -er .userId <<<"$output")
    uuid='^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'
    [[ $ticket =~ $uuid && $user =~ $uuid ]]

    run tidegate connect "$asset"
    [ "$status" -eq 0 ]
    next=${output%%$'\t'*}
    [[ $next =~ $uuid && $next != "$ticket" ]]
    [ "$output" = "$next"$'\t'"$asset"$'\t'"$user" ]
}

@test "connect requires exactly one asset ID in UUID format" {
    run --separate-stderr tidegate connect
    [ "$status" -eq 1 ]
    [[ $stderr == *"want exactly one asset ID"* ]]

    run --separate-stderr tidegate connect invalid
    [ "$status" -eq 1 ]
    [[ $stderr == *"want an asset ID in UUID format"* ]]
}

@test "connect reports a missing asset" {
    sign_in login "$TIDEGATE_TEST_AUTHORITY"
    run --separate-stderr tidegate connect 00000000-0000-4000-8000-000000000000
    [ "$status" -eq 1 ]
    [ -z "$output" ]
    [[ $stderr == *"asset not found"* ]]
}
