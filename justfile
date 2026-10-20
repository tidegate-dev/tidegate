mod control-plane
mod docs
mod proto

# List the recipes.
default:
    @{{just_executable()}} --list --list-submodules

# Build every component into bin/.
build: control-plane::build

# Run the unit tests of every component.
test: control-plane::test

# Generate code from the .proto files.
generate: proto::generate

# Lint every component.
lint: control-plane::lint proto::lint

# Check the dependencies of every component for known vulnerabilities.
vuln: control-plane::vuln

# Format the code of every component in place.
fmt: control-plane::fmt

# Run every check that CI runs.
ci: proto::ci control-plane::ci docs::build
