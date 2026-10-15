mod control-plane

# List the recipes.
default:
    @{{just_executable()}} --list --list-submodules

# Build every component into bin/.
build: control-plane::build

# Run the unit tests of every component.
test: control-plane::test

# Run every check that CI runs.
ci: control-plane::ci
