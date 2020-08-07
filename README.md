[![Sensu Bonsai Asset](https://img.shields.io/badge/Bonsai-Download%20Me-brightgreen.svg?colorB=89C967&logo=sensu)](https://bonsai.sensu.io/assets/sensu/sensu-kubernetes-checks)
![Go Test](https://github.com/sensu/sensu-kubernetes-checks/workflows/Go%20Test/badge.svg)
![goreleaser](https://github.com/sensu/sensu-kubernetes-checks/workflows/goreleaser/badge.svg)

# Sensu Kubernetes checks

## Table of Contents
- [Overview](#overview)
- [Files](#files)
- [Usage examples](#usage-examples)
- [Configuration](#configuration)
  - [Asset registration](#asset-registration)
  - [Check definition](#check-definition)
- [Installation from source](#installation-from-source)
- [Additional notes](#additional-notes)
- [Contributing](#contributing)

## Overview

The Sensu Kubernetes checks is a [Sensu Check][2] that ...

## Files

## Usage examples

## Configuration

### Asset registration

[Sensu Assets][3] are the best way to make use of this plugin. If you're not using an asset, please
consider doing so! If you're using sensuctl 5.13 with Sensu Backend 5.13 or later, you can use the
following command to add the asset:

```
sensuctl asset add sensu/sensu-kubernetes-checks
```

If you're using an earlier version of sensuctl, you can find the asset on the [Bonsai Asset Index][4].

### Check definition

```yml
---
type: CheckConfig
api_version: core/v2
metadata:
  name: sensu-kubernetes-checks
  namespace: default
spec:
  command: sensu-kubernetes-checks --example example_arg
  subscriptions:
  - system
  runtime_assets:
  - sensu/sensu-kubernetes-checks
```

## Installation from source

The preferred way of installing and deploying this plugin is to use it as an Asset. If you would
like to compile and install the plugin from source or contribute to it, download the latest version
or create an executable script from this source.

From the local path of the sensu-kubernetes-checks repository:

```
go build
```

## Additional notes

## Contributing

For more information about contributing to this plugin, see [Contributing][1].

[1]: https://github.com/sensu/sensu-go/blob/master/CONTRIBUTING.md
[2]: https://docs.sensu.io/sensu-go/latest/reference/checks/
[3]: https://docs.sensu.io/sensu-go/latest/reference/assets/
[4]: https://bonsai.sensu.io/assets/sensu/sensu-kubernetes-checks
