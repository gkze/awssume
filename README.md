# awssume - Golang package (and CLI) to manage assuming IAM Roles

[![Actions Test Workflow Widget]][Actions Test Workflow Status]
[![GoReport Widget]][GoReport Status]
[![GoDocWidget]][GoDocReference]

[Actions Test Workflow Status]: https://github.com/gkze/awssume/actions?query=workflow%3ACI
[Actions Test Workflow Widget]: https://github.com/gkze/awssume/workflows/CI/badge.svg

[GoReport Status]: https://goreportcard.com/report/github.com/gkze/awssume
[GoReport Widget]: https://goreportcard.com/badge/github.com/gkze/awssume

[GoDocWidget]: https://godoc.org/github.com/gkze/awssume?status.svg
[GoDocReference]:https://godoc.org/github.com/gkze/awssume

Package `awssume` implements operations around assuming [AWS IAM Roles](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_roles.html). See documentation on [Using IAM Roles](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_roles_use.html) and the [STS AssumeRole API](https://docs.aws.amazon.com/STS/latest/APIReference/API_AssumeRole.html) for more information on how assuming IAM Roles works.

The package uses [AWS SDK for Go v2](https://docs.aws.amazon.com/sdk-for-go/v2/api/), so it uses the [standard configuration patterns common to all official AWS SDKs](https://docs.aws.amazon.com/sdk-for-go/v1/developer-guide/configuring-sdk.html). It (`awssume`) does, however, introduce its own configuration, because the configuration shape it works with does not fit within an existing scheme easily.

`awssume` can be useful in scenarios when working with credentials in one AWS Account, but needing to quickly switch IAM Roles to perform certain tasks. There are other packages out there that help with [assuming Roles from identity providers through federataion](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_roles_providers.html) (see [`sts:AssumeRoleWithSAML`](https://docs.aws.amazon.com/STS/latest/APIReference/API_AssumeRoleWithSAML.html) and [`sts:AssumeRoleWithWebIdentity`](https://docs.aws.amazon.com/STS/latest/APIReference/API_AssumeRoleWithWebIdentity.html)) (like [`saml2aws`](https://github.com/Versent/saml2aws)), but they do not offer a solution for performing [`sts:AssumeRole`](https://docs.aws.amazon.com/STS/latest/APIReference/API_AssumeRole.html) without any federation and exposing the [security credentials](https://docs.aws.amazon.com/general/latest/gr/aws-sec-cred-types.html#access-keys-and-secret-access-keys) as environment variables. This package (and CLI) was written out of that need.

## Install

`awssume` the package can be installed via Go get:

```bash
$ go get -u -v github.com/gkze/awssume
```

`awssume` the CLI can be installed either via Go get:

```bash
$ go get -u -v github.com/gkze/awssume/cmd/awssume
```

or Homebrew:

```bash
brew install gkze/gkze/awssume
```

## Configuration and Usage

`awssume` does not require initial configuration, and is able to generate its
own configuration, given the correct calls / commands.

This is a simple example of how it can be done from Go:

```golang
import (
    "github.com/aws/aws-sdk-go-v2/aws/arn"
    "github.com/gkze/awssume/pkg/awssume"
    "github.com/spf13/afero"
)

func main() {
    cfg, err := awssume.NewConfig(&awssume.NewConfigOpts{
        Fs: afero.NewOsFs(), Path: awassume.DefaultConfigFilePath,
    })
    if err != nil {
        panic(err)
    }

    parsedARN, err := arn.Parse("arn:aws:iam::0000000000000:role/SomeRole")
    if err != nil {
        panic(err)
    }

    if err := cfg.AddRole(&awssume.Role{
        Alias: "someAlias",
        ARN: parsedARN,
        SessionName: "someSession",
    }); err != nil {
        panic(err)
    }

    if err := cfg.Save(); err != nil {
        panic(err)
    }
}
```

From the CLI, it can be done like so:

```bash
# awssume add [role] [alias] [session_name]
$ awssume add arn:aws:iam::0000000000000:role/SomeRole someAlias someSession
```

By default, `awssume` serializes the configuration to YAML, at the path `~/.config/awssume.yaml`. However, `awssume` is also capable of storing its configuration in either JSON or TOML, and can convert between any two of the formats.

Converting formats is easy:

```golang
import (
    "github.com/gkze/awssume/pkg/awssume"
    "github.com/spf13/afero"
)

func main() {
    cfg, err := awssume.NewConfig(&awssume.NewConfigOpts{
        Fs: afero.NewOsFs(), Path: awassume.DefaultConfigFilePath,
    })
    if err != nil {
        panic(err)
    }

    cfg.SetFormat(awssume.JSON)

    if err := cfg.Save(); err != nil {
        panic(err)
    }
}
```

or

```bash
$ awssume convert json
```

This would make `~/.config/awssume.yaml` disappear and `~/.config/awssume.json` appear instead in its place.

### Executing an Authenticated Subprocess

The main feature of `awssume` is to execute processes that have [STS Temporary Security Credentials](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_credentials_temp.html) exposed as environment variables.

From Go:

```golang
import (
    "github.com/gkze/awssume/pkg/awssume"
    "github.com/spf13/afero"
)

func main() {
    cfg, err := awssume.NewConfig(&awssume.NewConfigOpts{
        Fs: afero.NewOsFs(), Path: awassume.DefaultConfigFilePath,
    })
    if err != nil {
        panic(err)
    }

    cfg.ExecRole(
        "roleAlias", // Configured IAM Role alias
        60 * 60,     // 1 hour
        "aws",       // command
        []string{    // arguments
            "sts",
            "get-caller-identity",
        },
    )
}
```

From the CLI:

```bash
$ awssume roleAlias exec -- aws sts get-caller-identity
```

# License

[MIT](LICENSE)
