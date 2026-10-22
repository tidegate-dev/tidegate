mod cli
mod control-plane
mod docs
mod internal
mod proto

# List the recipes.
default:
    @{{just_executable()}} --list --list-submodules

# Build every component into bin/.
build: control-plane::build cli::build

# Run the unit tests of every component.
test: control-plane::test cli::test internal::test

# Generate code from the .proto files.
generate: proto::generate

# Lint every component.
lint: control-plane::lint cli::lint internal::lint proto::lint

# Check the dependencies of every component for known vulnerabilities.
vuln: control-plane::vuln cli::vuln internal::vuln

# Format the code of every component in place.
fmt: control-plane::fmt cli::fmt internal::fmt

# Start tidegate's components and the services they need.
dev: (_process-compose "dev/deps.yaml" "dev/tidegate.yaml")

# Start only the services that tidegate's components need, like Postgres.
dev-deps: (_process-compose "dev/deps.yaml")

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

images := "localhost/tidegate-control-plane"

# Build the container images of every component.
image: control-plane::image

# Run tidegate's components and the services they need in a local Kubernetes cluster.
kind: image
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
    kubectl --context kind-tidegate apply --kustomize dev/kind
    # The image tags stay the same, so the pods only pick up new images when
    # they restart.
    kubectl --context kind-tidegate --namespace tidegate rollout restart deployment
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
ci: proto::ci control-plane::ci cli::ci internal::ci docs::build
