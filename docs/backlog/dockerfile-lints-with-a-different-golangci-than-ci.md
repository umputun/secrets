---
worth: later
where: Dockerfile:17
added: 2026-08-21
---
# the Dockerfile lints with a different golangci-lint than the workflow does

`Dockerfile:17` runs `golangci-lint run` using whatever version `umputun/baseimage:buildgo-latest`
ships, while `.github/workflows/ci.yml:32` pins `v2.12.2`. Two independent versions of the same
linter check the same code, and they can demand opposite things: gosec at 2.12.2 emits G124, G705 and
G118 at sites where 2.9.0 does not, so a `//nolint` required by one version reads as unused to the
other and `nolintlint` fails the build. No set of suppressions satisfies both.

What makes this easy to misdiagnose is that the container side can go stale without anyone touching
it. `buildgo-latest` is only republished when baseimage cuts a version tag, not on a push to its
master, so it sat on the February v1.20.1 image for months. Master here went red at `bd85932`, which
bumped the CI linter and added the directives that satisfy it: no image changed, and the failure
looked like a base image regression while the real change was local. Fixed by baseimage v1.20.2
publishing a matching 2.12.2 pin.

Nothing prevents them diverging again, since the two pins live in different repositories with no link
between them and one of them only moves on a release. Options if it recurs: drop the lint step from
the Dockerfile since CI already covers it, or pin the base image to a version tag the way remark42
does with `buildgo-v1.17.0`.
