# ClusterFuzzLite only reads a file with this name.
FROM gcr.io/oss-fuzz-base/base-builder-go:v1@sha256:a83b0af8917a7df6d19c7437265f003a7418237ace38044e7209679a0b09f488
COPY . $SRC/phemera
WORKDIR $SRC/phemera
COPY .clusterfuzzlite/build.sh $SRC/
