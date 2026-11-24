bats_require_minimum_version 1.7.0

setup() {
    load common
    isolate
    # The tests share one database, so each one creates a project of its own.
    project=assets-$BATS_TEST_NUMBER
    host_key_file=$BATS_TEST_DIRNAME/../../dev/ssh-host/ssh_host_ed25519_key.pub
    host_key=$(cut -d' ' -f1,2 "$host_key_file")
}

@test "asset create ssh adds the test SSH host with its host key and a label, and asset list shows it" {
    sign_in login "$TIDEGATE_TEST_AUTHORITY"
    tidegate project create "$project"

    run tidegate asset create ssh --project-name "$project" --address 127.0.0.1:2222 --host-key "$(cat "$host_key_file")" --label env=dev ssh-host
    [ "$status" -eq 0 ]
    [[ $output == ?*$'\t'ssh-host ]]
    id=${output%%$'\t'*}

    run tidegate asset list --project-name "$project"
    [ "$status" -eq 0 ]
    [[ ${lines[0]} == ID*NAME*ADDRESS*LABELS ]]
    grep -qE "^$id +ssh-host +127\.0\.0\.1:2222 +env=dev\$" <<<"$output"
}

@test "asset create ssh and asset list print json" {
    sign_in login "$TIDEGATE_TEST_AUTHORITY"
    tidegate project create "$project"

    run tidegate -o json asset create ssh --project-name "$project" --address bastion.example --host-key "$host_key" --label env=prod --label team=payments bastion
    [ "$status" -eq 0 ]
    [ "$(jq -r .name <<<"$output")" = bastion ]
    [ "$(jq -c .labels <<<"$output")" = '{"env":"prod","team":"payments"}' ]
    [ "$(jq -c .ssh <<<"$output")" = "{\"host\":\"bastion.example\",\"port\":22,\"hostKey\":\"$host_key\"}" ]
    id=$(jq -r .id <<<"$output")

    run tidegate -o json asset list --project-name "$project"
    [ "$status" -eq 0 ]
    [ "$(jq -r '.assets | length' <<<"$output")" = 1 ]
    [ "$(jq -r '.assets[0].id' <<<"$output")" = "$id" ]
    [ "$(jq -r '.assets[0].ssh.host' <<<"$output")" = bastion.example ]
}

@test "asset list of a project without assets says how to add one" {
    sign_in login "$TIDEGATE_TEST_AUTHORITY"
    tidegate project create "$project"

    run tidegate asset list --project-name "$project"
    [ "$status" -eq 0 ]
    [[ $output == *"No assets in the project"* ]]
}

@test "asset list shows only the assets of the project" {
    sign_in login "$TIDEGATE_TEST_AUTHORITY"
    tidegate project create "$project"
    tidegate project create "$project-other"
    tidegate asset create ssh --project-name "$project-other" --address other.example --host-key "$host_key" other

    run tidegate -o json asset list --project-name "$project"
    [ "$status" -eq 0 ]
    [ "$(jq -r '.assets | length' <<<"$output")" = 0 ]
}

@test "asset create ssh fails for a name that another asset of the project has" {
    sign_in login "$TIDEGATE_TEST_AUTHORITY"
    tidegate project create "$project"
    tidegate asset create ssh --project-name "$project" --address bastion.example --host-key "$host_key" bastion

    run --separate-stderr tidegate asset create ssh --project-name "$project" --address other.example --host-key "$host_key" bastion
    [ "$status" -eq 1 ]
    [ -z "$output" ]
    [[ $stderr == *"the asset name is taken"* ]]
}

@test "asset create ssh and asset list fail for a project that doesn't exist" {
    sign_in login "$TIDEGATE_TEST_AUTHORITY"

    run --separate-stderr tidegate asset create ssh --project-name "$project" --address bastion.example --host-key "$host_key" bastion
    [ "$status" -eq 1 ]
    [ -z "$output" ]
    [[ $stderr == *"project not found"* ]]

    run --separate-stderr tidegate asset list --project-name "$project"
    [ "$status" -eq 1 ]
    [ -z "$output" ]
    [[ $stderr == *"project not found"* ]]
}

@test "asset create ssh fails for a host key that isn't a public key" {
    sign_in login "$TIDEGATE_TEST_AUTHORITY"

    run --separate-stderr tidegate asset create ssh --project-name "$project" --address bastion.example --host-key "not-a-type not/a*key" bastion
    [ "$status" -eq 1 ]
    [ -z "$output" ]
    [[ $stderr == *"must be the key type"* ]]
}

@test "asset create ssh wants exactly one name, a project, an address and a host key" {
    run --separate-stderr tidegate asset create ssh --project-name "$project" --address bastion.example --host-key "$host_key"
    [ "$status" -eq 1 ]
    [[ $stderr == *"want exactly one name"* ]]

    run --separate-stderr tidegate asset create ssh --address bastion.example --host-key "$host_key" bastion
    [ "$status" -eq 1 ]
    [[ $stderr == *'Required flag "project-name" not set'* ]]

    run --separate-stderr tidegate asset create ssh --project-name "$project" --host-key "$host_key" bastion
    [ "$status" -eq 1 ]
    [[ $stderr == *'Required flag "address" not set'* ]]

    run --separate-stderr tidegate asset create ssh --project-name "$project" --address bastion.example bastion
    [ "$status" -eq 1 ]
    [[ $stderr == *'Required flag "host-key" not set'* ]]
}

@test "asset create ssh refuses a label that isn't key=value" {
    run --separate-stderr tidegate asset create ssh --project-name "$project" --address bastion.example --host-key "$host_key" --label env bastion
    [ "$status" -eq 1 ]
    [ -z "$output" ]
    [[ $stderr == *'--label: label "env" must look like key=value'* ]]
}

@test "asset create ssh takes a comma as a part of the label's value, which the authority refuses" {
    sign_in login "$TIDEGATE_TEST_AUTHORITY"

    run --separate-stderr tidegate asset create ssh --project-name "$project" --address bastion.example --host-key "$host_key" --label env=prod,team=ops bastion
    [ "$status" -eq 1 ]
    [ -z "$output" ]
    [[ $stderr == *labels* ]]
}

@test "asset list wants a project and no arguments" {
    run --separate-stderr tidegate asset list
    [ "$status" -eq 1 ]
    [[ $stderr == *'Required flag "project-name" not set'* ]]

    run --separate-stderr tidegate asset list --project-name "$project" bastion
    [ "$status" -eq 1 ]
    [[ $stderr == *"want no arguments"* ]]
}

@test "asset create without a kind prints the kinds" {
    run tidegate asset create
    [ "$status" -eq 0 ]
    [[ $output == *ssh* ]]
}

@test "asset create with an unknown kind fails and names a similar one" {
    run --separate-stderr tidegate asset create shh
    [ "$status" -eq 1 ]
    [[ $stderr == *'unknown command "shh". Did you mean "ssh"?'* ]]
}
