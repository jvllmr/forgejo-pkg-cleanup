# Test environment

Runs Go end-to-end tests against a real Forgejo instance in Docker.
No extra container images are built; the action is tested as a plain binary.

## Layout

- `docker-compose.yml` — Forgejo 12 (`codeberg.org/forgejo/forgejo:12`) on
  `127.0.0.1:3000`, data in a throwaway named volume
- `run.sh` — starts Forgejo, creates the `testuser` user via the Forgejo CLI,
  runs `go test -tags e2e ./...`; `./run.sh down` removes everything
- `../main_test.go` — the e2e tests (build tag `e2e`)

The action binary is invoked exactly like forgejo-runner would invoke it: with
`INPUT_*` environment variables (what `githubactions.New()` reads).

## Fixtures (owner `testuser`, type `generic`)

| name     | version     | expected                              |
| -------- | ----------- | ------------------------------------- |
| testpkg  | 1.0.0       | kept (matches keepVersions)           |
| testpkg  | 1.2.3       | kept (matches keepVersions)           |
| testpkg  | nightly-abc | kept only inside retention span       |
| otherpkg | 999-dev     | always kept (name filter excludes it) |

## Scenarios (`go test -tags e2e`)

- retention `0s`: non-matching versions deleted, matching ones and
  name-filtered packages kept
- retention `1h` and `2d`: all fresh packages kept (`d` day suffix supported
  by `ParseRetention`)
- retention `banana`: action must fail, no packages deleted

## Usage

```sh
./run.sh        # start forgejo, run tests
./run.sh down   # stop forgejo, delete data
```

`run.sh` uses an isolated docker config so host credentials are never read.
Set `COMPOSE` to switch implementations, e.g. `COMPOSE=podman-compose ./run.sh`
(used by the CI workflow `.forgejo/workflows/test.yml`).

## Limitations

- Package `created_at` cannot be backdated, so the retention window is only
  exercised with fresh packages.
- Only `generic` packages are seeded; pushing `container` packages would
  require an OCI registry client.
