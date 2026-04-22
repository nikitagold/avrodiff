# avrodiff

Semantic diff for Avro schemas. Tells you whether a change is **BREAKING**, **SAFE**, or **COSMETIC** — not just what changed, but what it means for your consumers.

```
$ avrodiff --base user.avsc --head user.new.avsc

user.avsc

  BREAKING  fields.email  F-01  field "email" removed
                                consumers reading old messages will fail to deserialize

  SAFE      fields.phone  F-04  field "phone" added (default: <nil>)
                                backward and forward compatible

Result: MAJOR (breaking changes present)
```

## Why

`git diff` shows text changes. A developer sees "renamed a field" and thinks it's fine. In Avro binary format, that's a **MAJOR** breaking change — all consumers fail to deserialize. `avrodiff` understands Avro semantics and tells you the actual compatibility impact.

## Install

```bash
go install github.com/nikitagold/avrodiff@latest
```

Or build from source:

```bash
git clone https://github.com/nikitagold/avrodiff
cd avrodiff
go build -o avrodiff .
```

## Usage

```bash
avrodiff --base <base.avsc> --head <head.avsc> [--format text|json] [--mode BACKWARD|FORWARD|FULL]
```

| Flag | Default | Description |
|---|---|---|
| `--base` | — | Path to the original (old) schema |
| `--head` | — | Path to the modified (new) schema |
| `--format` | `text` | Output format: `text` or `json` |
| `--mode` | `FULL` | Compatibility mode (see below) |

**Exit codes:**

| Code | Meaning |
|---|---|
| `0` | No changes (`NONE`) |
| `1` | Tool error (bad flags, file not found, etc.) |
| `2` | Breaking changes (`MAJOR`) |
| `3` | Safe additions only (`MINOR`) |
| `4` | Cosmetic changes only (`PATCH`) |

## Output levels

| Level | Meaning | Examples |
|---|---|---|
| `MAJOR` | Breaking — consumers will fail | Field removed, type changed, enum reordered |
| `MINOR` | Safe addition — backward compatible | Field added with default, type promotion |
| `PATCH` | Cosmetic — no compatibility impact | Doc changed, default value changed |
| `NONE` | No changes | Identical schemas |

## Compatibility modes

Avro compatibility is directional. Use `--mode` to check specific directions:

| Mode | Guarantee | Typical use |
|---|---|---|
| `BACKWARD` | New schema reads data written by old schema | Consumer upgrade |
| `FORWARD` | Old schema reads data written by new schema | Producer upgrade |
| `FULL` | Both directions (default) | Schema registry |
| `NONE` | Report all changes without breaking/safe classification | Audit |

In `FULL` mode all breaking changes are flagged. In `BACKWARD`, only changes that break reading old data are flagged. In `FORWARD`, only changes that break old readers of new data.

## Rule reference

Every detected change is assigned a rule ID. The ID appears in both text and JSON output.

### Record rules (R)

| Rule | Change | Level | Why |
|---|---|---|---|
| R-01 | Record deleted | MAJOR | All consumers break — detected as F-09 when a field's type changes from a record to another type, or as S-01 when the root schema type changes |
| R-02 | Record renamed without alias | MAJOR | Fully-qualified name is the schema identifier; all consumers break |
| R-03 | Record renamed, old name kept as alias | MINOR | Alias provides backward-compatible alternative name |
| R-04 | Namespace changed | MAJOR | Namespace is part of the fully-qualified name |
| R-05 | Record alias removed | MAJOR | Consumers referencing this record by alias will break |
| R-06 | Record alias added | MINOR | Adds an alternative name; nothing breaks |
| R-07 | Record doc changed | PATCH | Documentation only |

### Field rules (F)

| Rule | Change | Level | Why |
|---|---|---|---|
| F-01 | Field removed (no default) | MAJOR | Readers expect the field and have no fallback |
| F-02 | Field removed (had default) | MINOR | Readers fall back to their local default |
| F-03 | Field added without default | MAJOR | Old readers can't find this field in existing data |
| F-04 | Field added with default | MINOR | Old readers use the default for missing fields |
| F-05 | Field renamed without alias | MAJOR | Binary format uses position; JSON encoding uses name |
| F-06 | Field renamed, old name kept as alias | MINOR | Alias preserves backward compatibility |
| F-07 | Field alias removed | MAJOR | Consumers referencing the field by alias will break |
| F-08 | Field alias added | MINOR | Adds an alternative name; nothing breaks |
| F-09 | Field type changed (incompatible) | MAJOR | Binary deserialization fails |
| F-10 | Field type promoted (`int`→`long`, etc.) | MINOR | Avro natively supports these promotions |
| F-11 | Field order changed | MAJOR | Avro binary encodes fields by position; consumers reading schemaless data break |
| F-12 | Default value changed | PATCH | Affects only future writers, not existing data |
| F-13 | Default added to a field | MINOR | Improves forward compatibility; nothing breaks |
| F-14 | Default removed from a field | MAJOR | Readers can no longer fall back to a value |
| F-15 | Field doc changed | PATCH | Documentation only |

### Enum rules (E)

| Rule | Change | Level | Why |
|---|---|---|---|
| E-01 | Symbol removed | MAJOR | Existing data contains this value; deserialization fails |
| E-02 | Symbol added (no enum default) | MAJOR | Old readers don't know how to handle the new symbol |
| E-03 | Symbol added (enum has default) | MINOR | Old readers fall back to the enum default for unknown symbols |
| E-04 | Symbols reordered | MAJOR | Avro binary encodes enum as an index, not a name |
| E-05 | Enum renamed without alias | MAJOR | Fully-qualified name is the schema identifier |
| E-06 | Enum renamed, old name kept as alias | MINOR | Alias preserves backward compatibility |
| E-07 | Enum alias removed | MAJOR | Consumers referencing the enum by alias will break |
| E-08 | Enum alias added | MINOR | Adds an alternative name; nothing breaks |
| E-09 | Enum default changed | PATCH | Affects only how unknown symbols are handled by new readers |
| E-10 | Enum namespace changed | MAJOR | Namespace is part of the fully-qualified name |
| E-11 | Enum doc changed | PATCH | Documentation only |

> **Note:** E-05..E-11 are detected when enum definitions are compared inline. The enum schema must include an `aliases` field for E-06..E-08 to be detected.

### Union rules (U)

| Rule | Change | Level | Why |
|---|---|---|---|
| U-01 | Type removed from union | MAJOR | Existing data may contain this type; deserialization fails |
| U-02 | Type added to union | MAJOR | Old readers don't know how to handle the new type |
| U-03 | Union member order changed | MAJOR | Avro binary encodes union as an index; reordering changes meaning |
| U-04 | `null` moved from first position | MAJOR | `["null","T"]` and `["T","null"]` differ in JSON encoding semantics |
| U-05 | Incompatible change inside union member | MAJOR | Emitted alongside the inner rule (e.g. F-01) when a breaking change is found inside a union member |

The safe widening `"T"` → `["null","T"]` and the breaking narrowing `["null","T"]` → `"T"` are also detected (via U-01/U-02 logic).

### Array rules (A)

| Rule | Change | Level | Why |
|---|---|---|---|
| A-01 | Array item type changed (incompatible) | MAJOR | Consumers can't deserialize existing elements |
| A-02 | Array item type promoted (`int`→`long`, etc.) | MINOR | Avro natively supports these promotions |
| A-03 | Array replaced by a different type | MAJOR | Complete binary incompatibility |

Changes inside record or enum item types are detected recursively.

### Map rules (M)

| Rule | Change | Level | Why |
|---|---|---|---|
| M-01 | Map value type changed (incompatible) | MAJOR | Consumers can't deserialize existing values |
| M-02 | Map value type promoted (`int`→`long`, etc.) | MINOR | Avro natively supports these promotions |
| M-03 | Map replaced by a different type | MAJOR | Complete binary incompatibility |

### Logical type rules (L)

| Rule | Change | Level | Why |
|---|---|---|---|
| L-01 | `logicalType` annotation added | MAJOR | Changes deserialization semantics for readers that understand it |
| L-02 | `logicalType` annotation removed | MAJOR | Readers lose semantic meaning of the data |
| L-03 | `logicalType` changed (e.g. `decimal`→`date`) | MAJOR | Different logicalTypes are semantically incompatible |
| L-04 | `decimal` precision changed | MAJOR | Alters the valid value range |
| L-05 | `decimal` scale decreased | MAJOR | Loss of fractional precision in existing data |
| L-06 | `decimal` scale increased | MAJOR | Old readers interpret the exponent incorrectly |
| L-07 | Underlying type changed, same `logicalType` | MAJOR | Underlying type determines binary encoding |

### Schema-level rules (S)

These are **lint** rules that only activate when the base schema contains a `version` field.

| Rule | Change | Level | Why |
|---|---|---|---|
| S-01 | Root schema type changed (`record`→`enum`, etc.) | MAJOR | Fundamental contract change; all consumers break |
| S-02 | `version` field missing in head schema | MAJOR | Version field is required for semver tracking |
| S-03 | Version bump is too small for the change level | MAJOR | e.g. only patch bumped when a minor-level change was made |
| S-04 | Version decreased | MAJOR | Semver versions must never go backwards |

Version must follow `MAJOR.MINOR.PATCH` format. S-02..S-04 are skipped if the base schema has no `version` field.

## Type promotions

Avro natively supports promoting a field's type to a compatible wider type (rules F-10, A-02, M-02). These are classified as `MINOR`.

| From | To |
|---|---|
| `int` | `long`, `float`, `double` |
| `long` | `float`, `double` |
| `float` | `double` |
| `string` | `bytes` |
| `bytes` | `string` |

All other type changes are `MAJOR`.

## Nested records

Rules apply recursively. A breaking change inside a nested record, array, or union member is reported with its full path and bubbles up as MAJOR.

```
fields.shipping.fields.country  BREAKING  F-01  field "country" removed
```

## JSON output

Use `--format json` for machine-readable output (CI, scripts):

```bash
$ avrodiff --base base.avsc --head head.avsc --format json
```

```json
{
  "level": "MAJOR",
  "changes": [
    {
      "rule": "F-01",
      "path": "fields.email",
      "description": "field \"email\" removed",
      "reason": "old schema readers expect this field; without a default they cannot read new data",
      "severity": "BREAKING",
      "affected_modes": ["FORWARD", "FULL"]
    }
  ]
}
```

The `rule` field identifies which of the 53 rules was triggered. `affected_modes` lists the compatibility modes in which this change is breaking.

## Semver versioning

Add a `version` field to your schema to enable semver lint checks (S-02..S-04):

```json
{
  "type": "record",
  "name": "User",
  "version": "1.2.0",
  "fields": [...]
}
```

`avrodiff` will verify that the version bump matches the change level:
- `MAJOR` change → major must increment (`1.x.x` → `2.0.0`)
- `MINOR` change → minor must increment (`1.2.x` → `1.3.0`)
- `PATCH` change → patch must increment (`1.2.3` → `1.2.4`)

Over-bumping is allowed (e.g. bumping major for a minor change is conservative but valid).

## CI integration

```yaml
# .github/workflows/schema-check.yml
- name: Check Avro schema compatibility
  run: |
    git show origin/main:avro/user.avsc > /tmp/user-base.avsc
    avrodiff \
      --base /tmp/user-base.avsc \
      --head avro/user.avsc \
      --format json \
      --mode FULL
```
