#!/bin/bash -eu

# The fuzz build replaces part of Go's testing package, which Go refuses for a
# toolchain in the module cache. That's where Go keeps the one go.mod asks for
# when the image's Go is older, so use a copy outside the cache.
toolchain=$(go env GOROOT)
if [[ $toolchain == "$(go env GOMODCACHE)"/* ]]; then
	cp -r "$toolchain" /tmp/go
	export GOROOT=/tmp/go PATH=/tmp/go/bin:$PATH GOTOOLCHAIN=local
fi

fuzz_test='^func Fuzz\w*\((\w+ )?\*testing\.F\)'

# Find every native Go fuzz test, so a new one can't be left out.
mapfile -t files < <(grep -rl --include='*_test.go' -E "$fuzz_test" .)

for file in "${files[@]}"; do
	# The fuzz build uses no build tags, so it can't see a tagged test.
	if grep -q '^//go:build' "$file"; then
		echo "$file: move the fuzz tests to a test file without a //go:build line" >&2
		exit 1
	fi

	# The build deletes the files of external test packages, see below.
	if grep -qE '^package \w+_test$' "$file"; then
		echo "$file: move the fuzz tests to a test file of the package itself, not of a _test package" >&2
		exit 1
	fi
done

for file in "${files[@]}"; do
	dir=$(dirname "$file")
	package=$(cd "$dir" && go list .)

	# The fuzz build turns every test file of the directory into part of the
	# package, so the files of an external test package would clash with it.
	# This is a throwaway copy of the repository.
	grep -lE '^package \w+_test$' "$dir"/*_test.go | xargs -r rm

	for function in $(grep -oP '^func \KFuzz\w*(?=\((\w+ )?\*testing\.F\))' "$file"); do
		# Unique across packages, and the corpus is stored under this name.
		name=${package#github.com/tidegate-dev/tidegate/}
		name=${name//\//_}_$function

		# The build writes a wrapper into the current directory, and Go only lets
		# it import an internal package from inside the directory above internal/.
		(cd "$dir" && compile_native_go_fuzzer_v2 "$package" "$function" "$name")
	done
done
