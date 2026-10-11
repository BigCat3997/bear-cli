# Bear CLI

Bear CLI is a DevOps automation toolkit that brings cloud sandbox, CI/CD and
infrastructure tasks into a single command-line interface. It reduces manual
steps, keeps secrets out of terminal output, and integrates with Terraform,
shell scripts and pipelines.

## Features

| Area | Capabilities |
| --- | --- |
| **Pluralsight sandbox** (`ps`) | Extract AWS and Azure sandbox credentials, output as `env`, `json` or `table`, and publish them to Azure DevOps or Jenkins |
| **Jenkins** (`jenkins`) | Create Secret Text and SSH private key credentials; register permanent SSH agents |
| **Azure DevOps** (`ado`) | List projects and variable groups; create or update variable groups, including secret variables |
| **AWS** (`aws`, `dynamodb`, `secret`) | Create CodeCommit repositories and push projects; create Terraform lock tables; manage Secrets Manager secrets |
| **Terraform** (`terraform`) | Initialize an Azure remote-state backend from sandbox credentials, bootstrap AWS state resources, and clean local working directories |
| **GoDaddy** (`godaddy`) | List domains, name servers and DNS records; change name servers |
| **Menu bar app** (macOS) | Start and stop Jenkins, manage sandbox credentials and run common actions from the menu bar |

## Table of contents

- [Getting started](#getting-started)
- [Configuration](#configuration)
- [Jenkins](#jenkins)
- [Pluralsight sandbox](#pluralsight-sandbox)
- [AWS](#aws)
- [Azure DevOps](#azure-devops)
- [GoDaddy](#godaddy)
- [Terraform](#terraform)
- [Menu bar app](#menu-bar-app)
- [Security notes](#security-notes)

## Getting started

### Prerequisites

- Go (version declared in [go.mod](go.mod))
- `make`
- macOS for the menu bar app

### Build and run

```sh
make build            # creates ./bear
./bear --help
```

During development, run without producing a binary:

```sh
go run ./cmd/bear --help
```

Other useful targets:

| Target | Description |
| --- | --- |
| `make install` | Build and copy `bear` to `/usr/local/bin` |
| `make build-menu-bar` | Build `dist/Bear Menu Bar.app` (macOS) |
| `make install-menu-bar` | Install the app into `~/Applications` |
| `make clean` | Remove the built binary |

## Configuration

Bear reads `config.toml` from the current working directory by default. Select a
different file with `BEAR_CONFIG` or the `--config` flag:

```sh
export BEAR_CONFIG="$HOME/.config/bear/work.toml"
```

Values are resolved in this order: **command flag**, then **environment
variable**, then **`config.toml`**.

```toml
default_profile = "work"

[output]
format = "table"          # json | table | env
color  = "auto"           # auto | always | never

[profiles.work]
aws_region       = "us-east-1"
aws_profile      = "work"
ado_organization = "<ADO_ORGANIZATION>"
ado_project      = "<ADO_PROJECT>"

[jenkins]
jenkins_url       = "http://localhost:8080"
jenkins_user_id   = "admin"
jenkins_api_token = "<API_TOKEN>"
jenkins_directory = "~/path/to/jenkins"   # folder containing jenkins.war (menu bar app, default ~/jenkins)

[terraform]
path = "<TERRAFORM_PATH>"

[godaddy]
base_url = "https://api.godaddy.com"

[[aws.secret_sets]]
title   = "Publish SSH keys"
tooltip = "Push key files to AWS Secrets Manager"

  [[aws.secret_sets.secrets]]
  name     = "<SECRET_NAME>"
  path_env = "KEY_PATH"          # or: path = "~/keys/id_rsa"
```

## Jenkins

Environment variables override `[jenkins]` properties:

```sh
export BEAR_JENKINS_URL="http://localhost:8080"
export BEAR_JENKINS_USER_ID="admin"
export BEAR_JENKINS_API_TOKEN="<API_TOKEN>"
```

### Create a Secret Text credential

```sh
bear jenkins credential create-secret-text \
  --id terraform-client-secret \
  --secret-env ARM_CLIENT_SECRET \
  --scope GLOBAL
```

Use `--secret-file <path>` to read from a file. `--secret` is available but can
expose the value in shell history and process listings. Add `--override` to
update an existing credential with the same ID.

### Create an SSH private key credential

The global `--user` is the Jenkins API user; `--username` is the remote SSH user.

```sh
bear jenkins credential create-ssh-private-key \
  --user admin \
  --id linux-agent-key \
  --username <SSH_USER> \
  --private-key-file <PRIVATE_KEY_PATH> \
  --scope SYSTEM \
  --override
```

Use `--private-key-env` when the key is in an environment variable. Encrypted
keys can supply a passphrase through `--passphrase-file` or `--passphrase-env`.

### Create a permanent SSH agent

```sh
bear jenkins agent create-ssh \
  --name linux-terraform-01 \
  --host <AGENT_HOST> \
  --ssh-port 22 \
  --credentials-id linux-agent-key \
  --remote-fs /home/jenkins \
  --executors 1 \
  --labels "linux terraform"
```

Requirements:

- The controller has the **SSH Build Agents** plugin.
- The host is reachable from the controller, has Java, and accepts the
  credential referenced by `--credentials-id`.
- With the default `known-hosts` strategy, the controller's Jenkins user has the
  host key in `~/.ssh/known_hosts`. `manually-trusted` is also available;
  `non-verifying` weakens SSH security.

Add `--override` to update an existing agent with the same name.

## Pluralsight sandbox

Copy the sandbox page's HTML to the clipboard, then extract credentials. Use
`--html-path <file>` to read a saved HTML file instead of the clipboard.

```sh
bear ps create --cloud-provider aws   --output env  --scope terraform
bear ps create --cloud-provider azure --output json --scope full
```

| Flag | Values | Default |
| --- | --- | --- |
| `--cloud-provider` | `aws`, `azure` | `azure` |
| `--output`, `-o` | `env`, `json`, `table`, `none` | `env` |
| `--scope`, `-s` | `full`, `terraform` | `full` |

Sign in to the sandbox portal automatically with `--login`:

```sh
bear ps create --cloud-provider aws --login
```

Other `ps` subcommands:

| Command | Purpose |
| --- | --- |
| `bear ps get` | Get credentials (supports `--html-path`, `--clipboard`, `--login`) |
| `bear ps init` | Initialize credentials in every consumed environment (`--path`, `--sandbox-path`) |
| `bear ps login` | Log in to the Azure portal (uses `ARM_SANDBOX_URL`, `ARM_USERNAME`, `ARM_PASSWORD` when set) |
| `bear ps terraform` | Set up an Azure Terraform backend (`--account-name`, `--location` required) |

### Publish to Azure DevOps

```sh
bear ps create \
  --cloud-provider aws \
  --publish-to-ado <ADO_PROJECT> \
  --login
```

Publishing uses the Terraform scope, suppresses credential output, and targets
the `vg-terraform-cred` variable group. Use `--variable-group <name>` to pick a
different group.

### Publish to Jenkins

```sh
export BEAR_JENKINS_API_TOKEN="<API_TOKEN>"

bear ps create \
  --cloud-provider aws \
  --publish-to-jenkins http://localhost:8080 \
  --jenkins-user admin \
  --jenkins-credential-scope GLOBAL \
  --override
```

Each environment variable becomes a Jenkins Secret Text credential. The default
AWS credential IDs are:

```text
aws-access-key-id
aws-secret-access-key
aws-region
```

Notes:

- Without `--override`, publishing stops instead of replacing existing values.
- Add an ID prefix with `--jenkins-credential-prefix` (for example `ps` gives
  `ps-aws-access-key-id`).
- Prefer `GLOBAL` scope for credentials used by pipelines; `SYSTEM` is for
  controller-level functions.
- The Jenkins account needs `Credentials/Create` (and `Credentials/Update` when
  overriding); the controller needs the Credentials Binding and Plain
  Credentials plugins.

Example pipeline binding:

```groovy
withCredentials([
  string(credentialsId: 'aws-access-key-id',     variable: 'AWS_ACCESS_KEY_ID'),
  string(credentialsId: 'aws-secret-access-key', variable: 'AWS_SECRET_ACCESS_KEY'),
  string(credentialsId: 'aws-region',            variable: 'AWS_REGION')
]) {
  sh 'terraform init'
  sh 'terraform plan'
}
```

## AWS

AWS commands use the standard credential chain (environment variables, shared
profile, or machine configuration). Override with `--region` and `--profile`.

### CodeCommit

```sh
bear aws create-repo \
  --repo <REPO_NAME> \
  --description "<DESCRIPTION>" \
  --region us-east-1
```

Create the repository if needed and upload a local folder or a public GitHub
repository:

```sh
bear aws push-project \
  --path <PROJECT_PATH> \
  --repo <REPO_NAME> \
  --branch main \
  --description "Initial push from <REPO_NAME>" \
  --region us-east-1
```

`--path` also accepts `<GIT_REPO_URL>`.

### Terraform lock table

```sh
bear dynamodb create-terraform-locks-table --table <TABLE_NAME>
```

### Secrets Manager

```sh
bear secret create --name <SECRET_NAME> --value "<VALUE>" --description "<DESCRIPTION>"
bear secret get    --name <SECRET_NAME>
bear secret delete --name <SECRET_NAME>            # 7-day recovery window
bear secret delete --name <SECRET_NAME> --force    # immediate deletion
```

Add `--override` to `create` to update an existing secret, `--tags key=value`
to tag it, and `--region` / `--profile` to target another account.

## Azure DevOps

Create a variable group or add variables to an existing one:

```sh
bear ado add-variable-group \
  --project <ADO_PROJECT> \
  --group shared-settings \
  --description "Shared pipeline settings" \
  --variable ENV=prod \
  --variable REGION=ap-southeast-1
```

Add a secret variable:

```sh
bear ado add-variable-group \
  --project <ADO_PROJECT> \
  --group shared-settings \
  --secret-variable DB_PASSWORD="$DB_PASSWORD"
```

Replace an existing value with `--override`:

```sh
bear ado add-variable-group \
  --project <ADO_PROJECT> \
  --group shared-settings \
  --variable ENV=staging \
  --override
```

Save Azure DevOps credentials once, then list resources:

```sh
bear ado bootstrap --org <ADO_ORGANIZATION> --pat <PERSONAL_ACCESS_TOKEN>
bear ado list-projects
bear ado list-variable-groups --project <ADO_PROJECT>
```

## GoDaddy

Every command needs `--api-key` and `--api-secret`:

```sh
bear godaddy list-domains    --api-key "$GODADDY_API_KEY" --api-secret "$GODADDY_API_SECRET" --limit 50
bear godaddy list-dnsservers --api-key "$GODADDY_API_KEY" --api-secret "$GODADDY_API_SECRET" --domain example.com
bear godaddy list-records    --api-key "$GODADDY_API_KEY" --api-secret "$GODADDY_API_SECRET" --domain example.com

bear godaddy change-dnsserver \
  --api-key "$GODADDY_API_KEY" \
  --api-secret "$GODADDY_API_SECRET" \
  --domain example.com \
  --nameserver ns1.example.net \
  --nameserver ns2.example.net
```

Optional filters: `list-domains` accepts `--limit` and `--offset`;
`list-records` accepts `--type` (for example `A`, `CNAME`, `TXT`) and `--name`
(requires `--type`). Use `--base-url` to target a different API endpoint.

## Terraform

```sh
# Azure remote-state backend from the stored sandbox credentials
bear terraform init --account-name <STORAGE_ACCOUNT> --location southeastasia

# Create AWS Terraform state resources
bear terraform bootstrap

# Remove .terraform directories and state files
bear terraform clean --path <TERRAFORM_PATH> --recursive
```

`init` also accepts `--container-name` (default `tfstate`) and
`--output env|json|table`. `clean` falls back to `[terraform].path` in
`config.toml` when `--path` is omitted.

## Menu bar app

The macOS menu bar app wraps common actions. Build and run it with:

```sh
make run-menu-bar
```

- **Preferences** shows the selected config file and how it was chosen. When
  launched from Finder, the app checks its own environment, then the Zsh startup
  environment (including `~/.zshrc`), before falling back to the default path.
- **Jenkins** starts and stops Jenkins from `jenkins_directory` in `config.toml`.
  Restart the app after changing it.

## Security notes

- Prefer environment variables or files over passing secrets as flag values,
  which can leak through shell history and process listings.
- If you store a token in `config.toml`, restrict it with `chmod 600 config.toml`.
- Publishing commands suppress credential output by design.
