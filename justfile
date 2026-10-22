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

# Run every check that CI runs.
ci: proto::ci control-plane::ci cli::ci internal::ci docs::build
