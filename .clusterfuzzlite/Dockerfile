# ClusterFuzzLite only reads a file with this name.
FROM gcr.io/oss-fuzz-base/base-builder-go:v1@sha256:f43249e6bc6e9036cd18f7b698302ac0ffc66c4cc10c01b6d09825f37f2c66cf
COPY . $SRC/phemera
WORKDIR $SRC/phemera
COPY .clusterfuzzlite/build.sh $SRC/
