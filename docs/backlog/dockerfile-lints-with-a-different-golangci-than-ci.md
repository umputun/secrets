---
worth: later
where: Dockerfile:17
added: 2026-08-21
---
# the Dockerfile lints with a different golangci-lint than the workflow does

`Dockerfile:17` runs `golangci-lint run` using whatever version `umputun/baseimage:buildgo-latest`
ships, while `.github/workflows/ci.yml:32` pins `v2.12.2`. Those are two independent versions of the
same linter checking the same code, and they can demand opposite things: gosec at 2.12.2 emits G124,
G705 and G118 at sites where 2.9.0 does not, so a `//nolint` required by one version reads as unused
to the other and `nolintlint` fails the build. No set of suppressions satisfies both.

This surfaced on 2026-08-21 when `buildgo-latest` was rebuilt for the first time since February and
delivered a 2.7.2 to 2.9.0 jump at once, turning master red. Fixed by bumping the base image pin to
2.12.2 (umputun/baseimage#53) so both sides agree, but nothing prevents them diverging again: the tag
is floating and the two pins are in different repositories with no link between them.

Options if it recurs: drop the lint step from the Dockerfile since CI already covers it, or pin the
base image to a version tag the way remark42 does with `buildgo-v1.17.0`.
