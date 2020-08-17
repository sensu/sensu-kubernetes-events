[![Sensu Bonsai Asset](https://img.shields.io/badge/Bonsai-Download%20Me-brightgreen.svg?colorB=89C967&logo=sensu)](https://bonsai.sensu.io/assets/sensu/sensu-kubernetes-events)
![Go Test](https://github.com/sensu/sensu-kubernetes-events/workflows/Go%20Test/badge.svg)
![goreleaser](https://github.com/sensu/sensu-kubernetes-events/workflows/goreleaser/badge.svg)

# Sensu Kubernetes Events Check

## Table of Contents
- [Overview](#overview)
- [Usage examples](#usage-examples)
  - [API Authentication](#api-authentication)
  - [Object kind](#object-kind)
  - [Event types](#event-types)
  - [Label selectors](#label-selectors)
  - [Status map](#status-map)
- [Configuration](#configuration)
  - [Asset registration](#asset-registration)
  - [Check definition](#check-definition)
- [Installation from source](#installation-from-source)
- [Additional notes](#additional-notes)
- [Contributing](#contributing)

## Overview

The Sensu Kubernetes events check is a [Sensu Check][2] that uses the
[Kubernetes Event API][5] to identify events that should generate corresponding
Sensu events.

This check should be thought of as a meta-check.  The check itself, unless it
encounters issues (e.g. trouble authenticating with Kubernetes for API access),
will always return an OK status (exit code 0).  However, for each matching event
type it does find, it will create separate events using the [agent API][6].

Given the above, when a matching event occurs, the check will need to be able
to connect to the agent API on `http://127.0.0.1:3031/events`.

## Usage examples
```
Sensu Kubernetes events check

Usage:
  sensu-kubernetes-events [flags]
  sensu-kubernetes-events [command]

Available Commands:
  help        Help about any command
  version     Print the version number of this plugin

Flags:
  -a, --agent-api-url string     The URL for the Agent API used to send events (default "http://127.0.0.1:3031/events")
  -t, --event-type string        Query for fieldSelector type (supports = and !=) (default "!=Normal")
  -e, --external                 Connect to cluster externally (using kubeconfig)
  -h, --help                     help for sensu-kubernetes-events
  -c, --kubeconfig string        Path to the kubeconfig file (default $HOME/.kube/config)
  -l, --label-selectors string   Query for labelSelectors (e.g. release=stable,environment=qa)
  -n, --namespace string         Namespace to which to limit this check
  -k, --object-kind string       Object kind to limit query to (Pod, Cluster, etc.)
  -s, --status-map string        Map Kubernetes event type to Sensu event status (default "{\"normal\": 0, \"warning\": 1, \"default\": 3}")

Use "sensu-kubernetes-events [command] --help" for more information about a command.

```
#### API authentication
In order to query the API, the check must authenticate.  The normal use case
would be for the check to be running from within a Kubernetes cluster and would
make use of the `rest.InClusterConfig()` function to handle API host discovery
and authentication automatically.  That is described [here][8].  This is the
default behavior.  To use "external" access requires the use of
[kubeconfig files][9] similar to the kubectl command.  This method is enabled
via the `--external` flag.  Additionally, the `--kubeconfig` option can be used
to point to an alternative kubeconfig file.

#### Object kind
If an object kind is not specified via the `--object-kind` argument, events for
all object kinds (cluster, pod, etc.) will be returned.

#### Event types
The expected use case for this check is to find anomalous events in your
Kubernetes environment(s).  For that reason, the default event type is
`!=Normal`.

#### Label selectors
[Label selectors][10] can be used to limit the scope of the Kubernetes events
returned and checked against the requested event type.  You can specify multiple
selectors by separating them with commas as the value for the
`--label-selectors` argument.

#### Status map
The status map allows you to map the event type (e.g. Normal, Warning) to a
[Sensu event check result][7].  It is a simple JSON map represented as a string.
The event types are case-insensitive.  The default, below, shows that Normal
maps to OK (0), Warning maps to Warning (1), and Default (anything else) maps to
 Unknown (3):
```JSON
{
  "Normal": 0,
  "Warning": 1,
  "Default": 3
}
```
## Configuration

### Asset registration

[Sensu Assets][3] are the best way to make use of this plugin. If you're not
using an asset, please consider doing so! If you're using sensuctl 5.13 with
Sensu Backend 5.13 or later, you can use the following command to add the asset:

```
sensuctl asset add sensu/sensu-kubernetes-events
```

If you're using an earlier version of sensuctl, you can find the asset on the
[Bonsai Asset Index][4].

### Check definition

```yml
---
type: CheckConfig
api_version: core/v2
metadata:
  name: sensu-kubernetes-events
  namespace: default
spec:
  command: sensu-kubernetes-events --example example_arg
  subscriptions:
  - system
  runtime_assets:
  - sensu/sensu-kubernetes-events
  stdin: true
  handlers:
  - slack
```
**Notes:**
* The check definition requires `stdin` be set to `true`.
* Any Events created by this check will include the handlers defined for it.

## Installation from source

The preferred way of installing and deploying this plugin is to use it as an
Asset. If you would like to compile and install the plugin from source or
contribute to it, download the latest version or create an executable binary
from this source.

From the local path of the sensu-kubernetes-events repository:

```
go build
```

## Additional notes

## Contributing

For more information about contributing to this plugin, see [Contributing][1].

[1]: https://github.com/sensu/sensu-go/blob/master/CONTRIBUTING.md
[2]: https://docs.sensu.io/sensu-go/latest/reference/checks/
[3]: https://docs.sensu.io/sensu-go/latest/reference/assets/
[4]: https://bonsai.sensu.io/assets/sensu/sensu-kubernetes-events
[5]: https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#event-v1-core
[6]: https://docs.sensu.io/sensu-go/latest/reference/agent/#create-monitoring-events-using-the-agent-api
[7]: https://docs.sensu.io/sensu-go/latest/reference/checks/#check-result-specification
[8]: https://kubernetes.io/docs/tasks/administer-cluster/access-cluster-api/#accessing-the-api-from-within-a-pod
[9]: https://kubernetes.io/docs/concepts/configuration/organize-cluster-access-kubeconfig/
[10]: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/
