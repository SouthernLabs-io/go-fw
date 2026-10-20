# Configuration

This framework can be configured using a `config.yaml` file. It must be placed in the same 
folder as the main binary.
> [!CAUTION]
> The `config.yaml` file should never contain secrets.

You can also override the configuration using environment variables.
You can store these in an optional `.env` file that should never be committed, so it is recommended to add it
to the `.gitignore` file.

> [!CAUTION]
> The `.env` file should never be committed to the repository.

Env Vars will be matched to the configuration keys in the `config.yaml` following these rules:
- The key in the `config.yaml` are converted to lowercase and a `_` is added for sub-keys.
  - Example: `httpServer.port` becomes `httpserver_port`
- Environment variables are converted to lowercase.
  - Example: `HTTPSERVER_port` becomes `httpserver_port`

Example using the `config.yaml` and override with `.env` file:
```yaml
name: simple-server
log:
  level: info

env:
  type: local
  name: local

httpServer:
    port: 8080

database:
  user: postgres
```

```shell
HTTPSERVER_PORT=9090
DATABASE_USER=my_custom_user
DATABASE_PASS=no_password
LOG_LEVEL=debug
```

## Request logging

Request logging is secure by default. The logged URL contains only the request path;
the query string is not logged. Referrer metadata is also omitted by default. These
defaults apply to both the standard `net/http` middleware and the Gin middleware.

If an application needs a small, known set of non-sensitive query parameters for
diagnostics, name them explicitly in the allowlist:

```yaml
httpServer:
  requestLogger:
    queryParameterAllowlist: ["page", "sort"]
    referrerMode: origin_path
```

`queryParameterAllowlist` logs only the named parameters. All other query parameters
remain excluded. `referrerMode: origin_path` enables sanitized referrer metadata using
the referrer's origin and path; the default empty value omits it.

The corresponding environment variables are:

```shell
HTTPSERVER_REQUESTLOGGER_QUERYPARAMETERALLOWLIST_0=page
HTTPSERVER_REQUESTLOGGER_QUERYPARAMETERALLOWLIST_1=sort
HTTPSERVER_REQUESTLOGGER_REFERRERMODE=origin_path
```

There is no compatibility switch for logging a raw URL or the full query string.
Keep the allowlist limited to values that are safe to include in logs.

## Secrets
> [!CAUTION] 
>Secrets should never be stored in `config.yaml` nor in `.env` files.

The only exception to this rule is when you are developing locally, and you need to provide the secrets.
In this case they should be stored in a `.env` file and never be committed.

A config value set to a `<secret…>` placeholder is replaced at load time with a value fetched from the
configured secrets manager. Two managers ship with the framework:
- `secrets.FxExportAWSSM` — AWS Secrets Manager.
- `secrets.FxExportEnvSM` — environment variables (the usual choice for local development).

### `<secret>` — id derived from the config path

The secret id is built from the config key's path using `secrets.prefixFmt` and `secrets.keyFmt`,
which default to `{Name}/{Env.Type}/{Env.Name}/` and `{Key}`. Supported placeholders are `{Name}`
(service name), `{Env.Type}`, `{Env.Name}` and, in `keyFmt` only, `{Key}`.

`{Key}` is the dotted path of the **Go struct fields** that hold the value (or their `json` tag names
when present), not the spelling used in `config.yaml`. The config is normalised through the struct
before secrets are resolved.

```yaml
name: my-service
env:
  type: dev
  name: dev1

database:
  pass: <secret>     # id: my-service/dev/dev1/Database.Pass
```

```yaml
secrets:
  prefixFmt: "{Env.Name}/"   # id becomes: dev1/Database.Pass
```

### `<secret:ID>` — id used verbatim

No transformation is applied to `ID`.

```yaml
database:
  pass: <secret:Dev1@database_pass>          # id: Dev1@database_pass
smtp:
  password: <secret:Password-For-DataStore>  # id: Password-For-DataStore
```

### Grouped secrets: one JSON object, many values

Related secrets can share a single secrets-manager entry whose value is a JSON object. Append `#` and
an [RFC 6901 JSON Pointer](https://datatracker.ietf.org/doc/html/rfc6901) to select one member. The
group is fetched **once** per config load no matter how many values reference it.

| Placeholder          | Secret id                          | JSON Pointer                     |
| -------------------- | ---------------------------------- | -------------------------------- |
| `<secret#>`          | transformed **parent** config path | `/<config key>` (implicit)       |
| `<secret#/ptr>`      | transformed **parent** config path | `/ptr`                           |
| `<secret:ID#>`       | `ID` verbatim                      | `/<config key>` (implicit)       |
| `<secret:ID#/ptr>`   | `ID` verbatim                      | `/ptr`                           |

The implicit form uses the config key the value sits on as the member name, so a whole block can be
grouped without repeating names:

```yaml
database:
  user: <secret#>    # id: my-service/dev/dev1/Database   member: /User
  pass: <secret#>    # same id, fetched once              member: /Pass
  port: <secret#>    #                                    member: /Port
```

```jsonc
// value of secret "my-service/dev/dev1/Database"
{ "User": "app", "Pass": "s3cr3t", "Port": 5432 }
```

Member names are matched exactly (case-sensitive). With the implicit form they follow the same rule
as `{Key}` above: Go field names, or `json` tag names when the struct declares them. Use an explicit
Pointer when the JSON uses different names or is nested:

```yaml
llm:
  openai:
    apiKey: <secret:llm-keys#/openai/apiKey>       # id: llm-keys
  anthropic:
    apiKey: <secret:llm-keys#/anthropic/apiKey>    # same id, fetched once
rotation:
  current: <secret:rotation#/passwords/0>          # array index
misc:
  value: <secret:misc#/api~1keys/openai>           # member "api/keys": '/' is written ~1, '~' is written ~0
```

Rules:
- The Pointer must start with `/`. `<secret#…>` is not allowed on a top-level key (there is no parent
  path); use `<secret:ID#…>` there.
- The secret must be a JSON object. The selected member must be a string, number or bool; numbers and
  bools are inserted as their literal text (`5432`, `true`) and converted by the normal config
  decoding, so they can target `int`/`bool` fields. `null`, objects and arrays are rejected.
- `#` is split at its first occurrence; it is not a legal character in AWS secret names or env var
  names, so ids never need escaping.
- Plain `<secret>` / `<secret:ID>` never parse the value: a secret that happens to be JSON (for
  example a GCP service-account key) is inserted whole.
- Errors name the config key, the secret id and the Pointer; they never include secret content.

For local development the same syntax works with `secrets.FxExportEnvSM` and a JSON blob in `.env`:

```shell
LLM_KEYS={"openai":{"apiKey":"sk-..."},"anthropic":{"apiKey":"sk-ant-..."}}
```


## Tests
When running test Go will set the working directory to the folder where the test file is located.
Configuration files will be searched following the algorithm below:

```plaintext
while not found:
    if in a sub-folder named "test": found
    if in the current folder: found
    if current folder contains a go.mod file: stop
    if parent folder is volume root: stop
    move to parent folder
```

- Example 1: config.yaml file in a sub-folder named "test"
```
   /project-root
   ├── config.yaml # this is the regular config file
   ├── go.mod
   ├── go.sum
   ├── main.go
   └── resource
       ├── test
           ├── config.yaml # It will be loaded for the tests in resource folder
           ├── user_test_data.sql
           └── docker-compose.yaml
       ├── user.go
       └── user_test.go
```

- Example 2: config.yaml file in the current folder
```
   /project-root
   ├── config.yaml # this is the regular config file
   ├── go.mod
   ├── go.sum
   ├── main.go
   └── resource
       ├── test
       │   ├── config.yaml # It will be loaded for the tests in resource folder
       │   ├── user_test_data.sql
       │   ├── docker-compose.yaml
       │   └── user_test.go
       └── user.go
```

- Example 3: config.yaml file in the parent folder, but under test.
```
   /project-root
   ├── config.yaml # this is the regular config file
   ├── go.mod
   ├── go.sum
   ├── main.go
   └── resource
       ├── test
       │   ├── config.yaml # It will be loaded for the tests in sub_resource folder
       │   ├── user_test_data.sql
       │   └── docker-compose.yaml
       ├── sub_resource
       │   ├── sub_resource.go
       │   └── sub_resource_test.go 
       └── user.go
```

This is also true for the `.env` file.
