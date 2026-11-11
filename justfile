mod cli
mod authority
mod docs
mod internal
mod proto

import 'authority/test-database.just'

# List the recipes.
default:
    @{{just_executable()}} --list --list-submodules

# Build every component into bin/.
build: authority::build cli::build

# Remove build and test output: bin/, coverage/ and site/.
clean:
    rm -rf bin coverage site

# Run the unit tests of every component.
test: authority::test cli::test internal::test

# Run the integration tests.
test-integration: authority::test-integration

# Run the API tests.
test-api: authority::test-api

# Run the end-to-end tests in e2e/ with Bats. They start an authority on its
# own database and use it through the tidegate CLI. Like the API tests, they
# need Postgres.
test-e2e: build
    PATH="$PWD/bin:$PATH" dev/with-test-authority.sh coverage/authority-e2e.out bats e2e/

# Change each line the branch changed since it split off from base, and list
# the changes that no test catches.
mutate $base="origin/main":
    #!/usr/bin/env sh
    set -eu
    # gremlins runs every test before it looks at the diff.
    if git diff --quiet --merge-base "$base" -- '*.go'; then
        echo "The branch changes no Go code since it split off from $base."
        exit 0
    fi
    go tool gremlins unleash --diff "$base"

# Generate code from the .proto files.
generate: proto::generate authority::generate

# Lint every component, and report secrets in exported APIs that aren't a
# secret.String.
lint: authority::lint cli::lint internal::lint proto::lint
    go tool secretcheck ./...

# Check the dependencies of every component and of the Go tools for known
# vulnerabilities.
vuln: authority::vuln cli::vuln internal::vuln
    govulncheck github.com/arch-go/arch-go/v2/...
    govulncheck github.com/go-gremlins/gremlins/...

# gitleaks reads every branch unless --log-opts names one.
# Check every commit for secrets, like private keys.
secrets:
    gitleaks git --redact --no-banner --log-opts=HEAD

# Check which packages may import which, as arch-go.yml says.
arch:
    go tool arch-go

# Check that files pinning the same image pin the same version.
images:
    #!/usr/bin/env sh
    set -eu
    pins=$(grep -rhoE --include='*.yaml' 'image: *[^ ]+' dev .github/workflows | sed -E 's/image: *//' | sort -u)
    names=$(echo "$pins" | sed -E 's/:[^/]*$//' | sort | uniq -d)
    for name in $names; do
        echo "$name has different pins:" >&2
        grep -rnoE --include='*.yaml' "image: *$name[:@][^ ]+" dev .github/workflows >&2
    done
    [ -z "$names" ]

# Check that every Go module in tidegate's binaries uses a license that
# .grant.yaml allows, and that every license file syft can't identify is in
# .licenses-reviewed.
licenses: build
    #!/usr/bin/env sh
    set -eu
    syft scan dir:bin --config .syft.yaml -q -o syft-json=bin/sbom.json
    if ! jq -e 'any(.artifacts[]; .type == "go-module")' bin/sbom.json > /dev/null; then
        echo "syft found no Go modules in bin/" >&2
        exit 1
    fi
    # grant drops license entries that syft gives only a sha256 value.
    unreviewed=$(jq -r '.artifacts[] | .name as $module | .licenses[]? | select(.value | startswith("sha256:")) | "\($module) \(.value) \(.urls[0] // "")"' bin/sbom.json \
        | awk 'NR == FNR { if (!/^#/ && NF) reviewed[$1 " " $2] = 1; next } !reviewed[$1 " " $2]' .licenses-reviewed - | sort -u)
    if [ -n "$unreviewed" ]; then
        echo "syft can't identify these license files. Read them and add them to .licenses-reviewed:" >&2
        echo "$unreviewed" >&2
        exit 1
    fi
    grant check --config .grant.yaml bin/sbom.json

# Format the code of every component in place.
fmt: authority::fmt cli::fmt internal::fmt

# Start tidegate's components and the services they need.
dev: _dev-tls (_process-compose "dev/deps.yaml" "dev/tidegate.yaml")

# Start only the services that tidegate's components need, like Postgres.
dev-deps: (_process-compose "dev/deps.yaml")

_dev-tls:
    dev/tls-setup.sh

[positional-arguments]
_process-compose +configs:
    #!/usr/bin/env sh
    set -eu
    CONTAINER_RUNTIME=$(command -v podman || command -v docker) || {
        echo "Install podman or Docker to run the development environment." >&2
        exit 1
    }
    export CONTAINER_RUNTIME
    for config in "$@"; do
        set -- "$@" --config "$config"
        shift
    done
    # process-compose's API server listens on port 8080 by default, which
    # tidegate's components use, so it uses a Unix socket instead.
    exec process-compose up --use-uds "$@"

# Delete the data that the services keep between runs, like the Postgres database.
dev-reset:
    #!/usr/bin/env sh
    set -eu
    runtime=$(command -v podman || command -v docker) || {
        echo "Install podman or Docker to run the development environment." >&2
        exit 1
    }
    "$runtime" volume rm --force tidegate-postgres

images := "localhost/tidegate-authority"

# Build the container images of every component.
image: authority::image

# Run tidegate's components and the services they need in a local Kubernetes cluster.
kind: image _dev-tls
    #!/usr/bin/env sh
    set -eu
    runtime=$(command -v podman || command -v docker) || {
        echo "Install podman or Docker to run the development environment." >&2
        exit 1
    }
    # kind uses Docker unless told otherwise.
    case "$runtime" in */podman) export KIND_EXPERIMENTAL_PROVIDER=podman ;; esac
    if ! kind get clusters 2>/dev/null | grep -qx tidegate; then
        kind create cluster --name tidegate --config dev/kind/cluster.yaml
    fi
    archive=$(mktemp)
    trap 'rm -f "$archive"' EXIT
    for image in {{images}}; do
        "$runtime" save --output "$archive" "$image"
        kind load image-archive "$archive" --name tidegate
    done
    helm dependency build dev/kind/cert-manager
    helm --kube-context kind-tidegate upgrade --install cert-manager dev/kind/cert-manager \
        --namespace cert-manager --create-namespace --wait
    kubectl --context kind-tidegate apply --filename dev/kind/namespace.yaml
    kubectl --context kind-tidegate --namespace tidegate create secret tls dev-ca \
        --cert dev/tls/ca.pem --key dev/tls/ca-key.pem --dry-run=client --output yaml \
        | kubectl --context kind-tidegate apply --filename -
    kubectl --context kind-tidegate apply --kustomize dev/kind
    # The image tags stay the same, so the components only pick up new images
    # when they restart. The services keep running, so Postgres keeps its data.
    kubectl --context kind-tidegate --namespace tidegate rollout restart deployment --selector app.kubernetes.io/part-of=tidegate
    kubectl --context kind-tidegate --namespace tidegate rollout status deployment --timeout 3m

# Delete the cluster that just kind created.
kind-down:
    #!/usr/bin/env sh
    set -eu
    runtime=$(command -v podman || command -v docker) || {
        echo "Install podman or Docker to run the development environment." >&2
        exit 1
    }
    case "$runtime" in */podman) export KIND_EXPERIMENTAL_PROVIDER=podman ;; esac
    kind delete cluster --name tidegate

# Run every check that CI runs, except fuzzing and mutate.
ci: secrets arch images licenses proto::ci authority::ci cli::ci test-e2e internal::ci docs::build
