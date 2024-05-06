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

## Secrets
> [!CAUTION] 
>Secrets should never be stored in `config.yaml` nor in `.env` files.

The only exception to this rule is when you are developing locally, and you need to provide the secrets.
In this case they should be stored in a `.env` file and never be committed.

The framework provides secrets management by setting the value to `<secret>`. 
This will trigger a search for the secret in the configured SecretManager.

The key in the SecretManager will be constructed using the `env.name` and the key in the `config.yaml` file.
- Example 1: `env.name: dev1` and `database.pass: <secret>`: the key will be `dev1/database.pass`
- Example 2: `env.name: QA2` and `Triggers.on_save.dataStorePass: <secret>`: the key will be `QA2/Triggers.on_save.dataStorePass`

You can override this behavior by using the format `<secret:{custom_key}>`. In this case no change will be
made to the provided `custom_key`.
- Example 1: `database.pass: <secret:Dev1@database_pass>`: the key will be `Dev1@database_pass`
- Example 2: `Triggers.on_save.dataStorePass: <secret:Password-For-DataStore>`: the key will be `Password-For-DataStore`


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