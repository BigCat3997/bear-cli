# Bear CLI

Bear CLI is a DevOps automation toolkit designed to streamline cloud infrastructure and DevOps workflows.
Bear CLI helps DevOps engineers save time, reduce manual steps, and improve security when working with cloud sandboxes and DevOps environments.
It provides a unified command-line interface for:

- Extracting and managing sandbox credentials from Pluralsight labs (AWS, Azure)
- Outputting credentials in multiple formats for easy integration with Terraform, shell scripts, and CI/CD pipelines

---

## Pluralsight Sandbox (`ps`) Command Usage

**Extract AWS credentials:**

```sh
bear ps create-cred --cloud-provider=aws --output=env --scope=terraform
```

**Extract Azure credentials:**

```sh
bear ps create-cred --cloud-provider=azure --output=json --scope=full
```

**Login to sandbox portal automatically:**

```sh
bear ps create-cred --cloud-provider=aws --login
```

**Get stored credentials:**

```sh
bear ps get-cred --output=env
```

---
