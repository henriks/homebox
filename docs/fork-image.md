# Fork image publishing

This fork's rootless image is `ghcr.io/henriks/homebox`, built for
`linux/amd64`. It runs as UID/GID 65532. The source revision is recorded in
the tag and passed to the backend build. From a clean, committed checkout:

```sh
scripts/docker-image.sh build
# Test the local image before publishing it.
scripts/docker-image.sh push
```

The `build` command creates `ghcr.io/henriks/homebox:<12-character commit>-rootless`.
The `push` command publishes that tag and updates `latest-rootless`. Set `HOMEBOX_IMAGE`
to publish to another registry. Docker must already be logged in to the target
registry. For a pinned deployment, use the commit tag or the published digest.

The fork's `.github/workflows/fork-rootless-image.yml` runs from the fork's
default branch. It rebases `feature/scale-uploaded-photos` onto upstream `main`
hourly and publishes a rootless image to GHCR when that source revision is not
already published. The workflow can also be run manually. A rebase conflict
fails the workflow without rewriting the branch. The local script is an
equivalent manual build and publish procedure.

The upstream `.github/workflows/docker-publish.yaml` builds separate amd64 and
arm64 images on main, release tags, and nightly schedules. It pushes to GHCR,
and publishes releases and nightly builds to Docker Hub. The fork image uses
only amd64, matching the current deployment host.

This procedure reproduces the build inputs and tags for a Git revision; the
Dockerfile.rootless still uses floating base-image and Alpine package versions, so
rebuilding later is not guaranteed to produce identical bytes.
