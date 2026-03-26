# avrodiff

Semantic diff for Avro schemas. Tells you whether a change is **BREAKING**, **SAFE**, or **COSMETIC** — not just what changed, but what it means for your consumers.

```
$ avrodiff --base user-created.avsc --head user-created.new.avsc

user-created.avsc

  BREAKING  field "email" removed
            consumers reading old messages will fail to deserialize

  SAFE      field "phone" added (default: <nil>)
            backward and forward compatible

Result: MAJOR (breaking changes present)
```

## Why

`git diff` shows text changes. A developer sees "renamed a field" and thinks it's minor. In Avro, that's a **MAJOR** breaking change — all consumers will fail to deserialize.

`avrodiff` understands Avro semantics and tells you the actual impact.

## Install

```bash
go install avrodiff@latest
```

Or build from source:

```bash
git clone ...
cd avrodiff
go build -o avrodiff .
```

## Usage

```bash
avrodiff --base <base.avsc> --head <head.avsc> [--format text|json]
```

| Flag | Description |
|---|---|
| `--base` | Path to the original schema |
| `--head` | Path to the modified schema |
| `--format` | Output format: `text` (default) or `json` |

**Exit codes:**
- `0` — no changes
- `1` — breaking changes (MAJOR)
- `2` — safe additions (MINOR)
- `3` — cosmetic changes (PATCH)

## Output levels

| Level | Means | When |
|---|---|---|
| `MAJOR` | Breaking change — consumers will break | Field removed, type changed, enum reordered |
| `MINOR` | Safe addition — backward compatible | Field added with default, enum symbol added |
| `PATCH` | Cosmetic change — no compatibility impact | Doc changed, default value changed |
| `NONE` | No changes | Identical schemas |

## Classification rules

### Fields

| Change | Level | Why |
|---|---|---|
| Field removed | MAJOR | Consumers reading old messages will fail to deserialize |
| Field added without default | MAJOR | Old producers don't write this field |
| Field added with default | MINOR | Backward and forward compatible |
| Field type changed | MAJOR | Binary incompatibility |
| Field renamed (no aliases) | MAJOR | Treated as remove + add |
| Field renamed (with alias) | MINOR | Old name available as alias |
| Doc changed | PATCH | Documentation only |
| Default value changed | PATCH | Does not affect already written data |

### Enum

| Change | Level | Why |
|---|---|---|
| Symbol removed | MAJOR | Old data contains this value, deserialization will fail |
| Symbol order changed | MAJOR | Avro encodes enum as index, not name |
| Symbol added | MINOR | Consumers should handle unknown enum values |

### Union

| Change | Level | Why |
|---|---|---|
| Type removed | MAJOR | Old data may contain this type |
| Type added | MAJOR | Old consumers don't know the new type |
| Order changed | MAJOR | Avro binary encodes union as index |
| `"T"` → `["null", "T"]` | MINOR | Safe widening |
| `["null", "T"]` → `"T"` | MAJOR | Narrowing — old null values won't deserialize |

### Nested records

Rules apply recursively. A breaking change inside a nested record bubbles up as MAJOR.

```
fields.shipping.fields.country  BREAKING  field "country" removed
```

## Examples

**MAJOR — field removed:**
```bash
$ avrodiff --base base.avsc --head head.avsc

user.avsc

  BREAKING  field "email" removed
            consumers reading old messages will fail to deserialize

Result: MAJOR (breaking changes present)
# exit code: 1
```

**MINOR — optional field added:**
```bash
$ avrodiff --base base.avsc --head head.avsc

user.avsc

  SAFE      field "phone" added (default: <nil>)
            backward and forward compatible

Result: MINOR (backward compatible additions)
# exit code: 2
```

**MINOR — safe rename with alias:**
```json
// head.avsc — field renamed but old name preserved as alias
{"name": "userId", "type": "string", "aliases": ["user_id"]}
```
```
  SAFE      field "user_id" renamed to "userId" (alias preserved)
            old name available as alias, backward compatible
```

**PATCH — doc updated:**
```
  COSMETIC  field "id" doc changed
            documentation only, no compatibility impact

Result: PATCH (cosmetic changes only: doc, defaults)
# exit code: 3
```

**JSON output (for CI):**
```bash
$ avrodiff --base base.avsc --head head.avsc --format json
{
  "schema": "base.avsc",
  "level": "MAJOR",
  "changes": [
    {
      "path": "fields.email",
      "description": "field \"email\" removed",
      "reason": "consumers reading old messages will fail to deserialize",
      "severity": "BREAKING"
    }
  ]
}
```

## CI integration

```yaml
# .github/workflows/schema-check.yml
- name: Check Avro schema compatibility
  run: |
    avrodiff --base origin/main:avro/user-created.avsc \
                  --head avro/user-created.avsc \
                  --format json > schema-report.json
    cat schema-report.json
```

The command exits with `1` if breaking changes are detected, blocking the pipeline.

## Supported Avro features

- [x] Primitive types (`string`, `int`, `long`, `float`, `double`, `boolean`, `bytes`, `null`)
- [x] Record types (including nested records, recursively)
- [x] Enum types (symbol add/remove/reorder)
- [x] Union types (`["null", "string"]`, multi-type unions)
- [x] Field aliases (safe rename detection)
- [x] Field defaults (including `"default": null`)
- [x] Named type references (e.g. `"type": "MyRecord"` as a string)
- [x] Array and map types
- [x] `BACKWARD` / `FORWARD` / `FULL` mode selection
- [ ] Multiple schemas in one diff run