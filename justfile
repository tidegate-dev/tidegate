mod cli
mod authority
mod docs
mod internal
mod proto

# List the recipes.
default:
    @{{just_executable()}} --list --list-submodules

# Build every component into bin/.
build: authority::build cli::build

# Run the unit tests of every component.
test: authority::test cli::test internal::test

# Run the integration tests.
test-integration: authority::test-integration

# Generate code from the .proto files.
generate: proto::generate authority::generate

# Lint every component.
lint: authority::lint cli::lint internal::lint proto::lint

# Check the dependencies of every component and of the Go tools for known
# vulnerabilities.
vuln: authority::vuln cli::vuln internal::vuln
    govulncheck github.com/arch-go/arch-go/v2/...

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

# Run every check that CI runs.
ci: secrets arch images proto::ci authority::ci cli::ci internal::ci docs::build
