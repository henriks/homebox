#!/bin/sh
set -eu

usage() {
  echo "Usage: $0 build|push" >&2
  exit 2
}

mode=${1:-}
case "$mode" in
  build|push) ;;
  *) usage ;;
esac

repo_dir=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$repo_dir"

if [ -n "$(git status --porcelain)" ]; then
  echo "Commit the working tree before building or publishing an image." >&2
  exit 1
fi

revision=$(git rev-parse HEAD)
tag=$(git rev-parse --short=12 HEAD)
image=${HOMEBOX_IMAGE:-git.iio.fi/hsaksela/homebox}
reference=$image:$tag

if [ "$mode" = build ]; then
  docker buildx build \
    --platform linux/amd64 \
    --file Dockerfile \
    --build-arg "COMMIT=$revision" \
    --build-arg "VERSION=$tag" \
    --tag "$reference" \
    --load .
  echo "Built $reference"
else
  docker image inspect "$reference" >/dev/null
  docker push "$reference"
  docker tag "$reference" "$image:latest"
  docker push "$image:latest"
  echo "Published $reference and $image:latest"
fi
