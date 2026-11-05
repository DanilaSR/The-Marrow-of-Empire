# Test Infrastructure

**Engine**: Kaiju Engine 26.07.29.1-nightly (Go/Vulkan)
**Test Framework**: Go native `testing` package + Kaiju integration test harness
**CI**: `.github/workflows/test.yml`
**Setup date**: 2026-08-09

## Directory Layout

```
tests/
  unit/           # Isolated unit tests (formulas, state machines, logic)
  integration/    # Cross-system and save/load round-trip tests
  smoke/          # Critical path test list for /smoke-check gate
  evidence/       # Screenshot logs and manual test sign-off records
```

## Running Tests

```bash
# Unit tests (fast, no engine required)
cd src
go test ./tests/unit/...

# Integration tests (requires Kaiju Engine build)
CGO_ENABLED=1 go build -tags="debug,editor,filedrop,rawsrc" -o ../bin/ ./
../bin/kaijuengine.com -integrationtest=<testname>

# All tests
cd src
go test ./tests/...
```

## Test Naming

- **Files**: `[system]_test.go`
- **Functions**: `Test[Feature]_[Scenario]_[Expected]`
- **Example**: `combat_damage_test.go` → `TestDamage_BaseAttack_ReturnsExpectedValue(t *testing.T)`

## Story Type → Test Evidence

| Story Type | Required Evidence | Location |
|---|---|---|
| Logic | Automated unit test — must pass | `tests/unit/[system]/` |
| Integration | Integration test OR documented playtest | `tests/integration/[system]/` |
| Visual/Feel | Screenshot + lead sign-off | `tests/evidence/` |
| UI | Manual walkthrough OR interaction test | `tests/evidence/` |
| Config/Data | Smoke check pass | `production/qa/smoke-*.md` |

## CI

Tests run automatically on every push to `main` and on every pull request.
A failed test suite blocks merging. CGO is enabled in CI for Kaiju Engine compatibility.
