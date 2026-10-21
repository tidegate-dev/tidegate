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

# Run every check that CI runs.
ci: proto::ci control-plane::ci cli::ci internal::ci docs::build
