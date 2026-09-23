# Fork image publishing

This fork's regular image is `git.iio.fi/hsaksela/homebox`, built for
`linux/amd64`. The source revision is recorded in the tag and passed to the
backend build. From a clean, committed checkout:

```sh
scripts/docker-image.sh build
# Test the local image before publishing it.
scripts/docker-image.sh push
```

The `build` command creates `git.iio.fi/hsaksela/homebox:<12-character commit>`.
The `push` command publishes that tag and updates `latest`. Set `HOMEBOX_IMAGE`
to publish to another registry. Docker must already be logged in to the target
registry. For a pinned deployment, use the commit tag or the published digest.

The upstream `.github/workflows/docker-publish.yaml` builds separate amd64 and
arm64 images on main, release tags, and nightly schedules. It pushes to GHCR,
and publishes releases and nightly builds to Docker Hub. That GitHub Actions
workflow does not publish images to this Gitea registry. The script above is
the repeatable build and publish procedure for this fork.

This procedure reproduces the build inputs and tags for a Git revision; the
Dockerfile still uses floating base-image and Alpine package versions, so
rebuilding later is not guaranteed to produce identical bytes.
