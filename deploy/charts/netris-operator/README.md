# Netris-Operator

Netris-operator runs within your Kubernetes cluster as a deployment resource. It utilizes CustomResourceDefinitions to configure netris cloud resources.

## Prerequisites

- Kubernetes 1.16+
- Helm 3.1+

## Installing the Chart

In order to install the Helm chart, you must follow these steps:

Create the namespace for netris-operator:

```
kubectl create namespace netris-operator
```

Add the Netris Helm repository:

```
helm repo add netrisai https://netrisai.github.io/charts
```

Update your local Helm chart repository cache:

```
helm repo update
```

### Option 1: Creds from secret

1) Create credentials secret for netris-operator:

```
kubectl -n netris-operator create secret generic netris-creds \
  --from-literal=host="http://example.com" \
  --from-literal=login="login" --from-literal=password="pass"
```

2) Install helm chart

```
helm install netris-operator netrisai/netris-operator \
--namespace netris-operator
```

### Option 2: Creds from helm values

 1) Install helm chart with netris controller creds

```
helm install netris-operator netrisai/netris-operator \
--namespace netris-operator \
--set controller.host="http://example.com" \
--set controller.login="login" \
--set controller.password="pass"
```

## Uninstalling the Chart

To uninstall/delete the `netris-operator` helm release:

```
helm uninstall netris-operator
```

## Configuration

The following table lists the configurable parameters of the netris-operator chart and their default values.

### Common parameters

| Parameter                             | Description                                                                                               | Default                    |
| ------------------------------------- | --------------------------------------------------------------------------------------------------------- | -------------------------- |
| `nameOverride`                        | String to partially override common.names.fullname template with a string (will prepend the release name) | `nil`                      |
| `fullnameOverride`                    | String to fully override common.names.fullname template with a string                                     | `nil`                      |
| `rbac.create`                         | Specify if an rbac authorization should be created with the necessarry Rolebindings                       | `true`                     |
| `serviceAccount.create`               | Create a serviceAccount for the deployment                                                                | `true`                     |
| `serviceAccount.name`                 | Use the serviceAccount with the specified name                                                            | `""`                       |
| `serviceAccount.annotations`          | Annotations to add to the service account                                                                 | `{}`                       |
| `podAnnotations`                      | Pod annotations                                                                                           | `{}`                       |
| `podSecurityContext`                  | Pod Security Context                                                                                      | `{}`                       |
| `securityContext`                     | Containers security context                                                                               | `{}`                       |
| `service.type`                        | kube-rbac-proxy Service type                                                                              | `ClusterIP`                |
| `service.port`                        | kube-rbac-proxy Service port                                                                              | `8443`                     |
| `resources`                           | CPU/memory resource requests/limits                                                                       | `{}`                       |
| `nodeSelector`                        | Node labels for pod assignment                                                                            | `{}`                       |
| `tolerations`                         | Node tolerations for pod assignment                                                                       | `[]`                       |
| `affinity`                            | Node affinity for pod assignment                                                                          | `{}`                       |

### Netris-Operator parameters
| Parameter                             | Description                                                                                                   | Default                    |
| ------------------------------------- | ------------------------------------------------------------------------------------------------------------- | -------------------------- |
| `imagePullSecrets`                    | Reference to one or more secrets to be used when pulling images                                               | `[]`                       |
| `image.repository`                    | Image repository                                                                                              | `netrisai/netris-operator` |
| `image.tag`                           | Image tag. Overrides the image tag whose default is the chart appVersion                                      | `""`                       |
| `image.pullPolicy`                    | Image pull policy                                                                                             | `Always`                   |
| `controller.host`                     | Netris controller host url (`http://example.com`)                                                             | `""`                       |
| `controller.login`                    | Netris controller login                                                                                       | `""`                       |
| `controller.password`                 | Netris controller password                                                                                    | `""`                       |
| `controller.insecure`                 | Allow insecure server connections when using SSL                                                              | `false`                    |
| `controllerCreds.host.secretName`     | Name of existing secret to use for Netris controller host. Ignored if `controller.host` is set                | `netris-creds`             |
| `controllerCreds.host.key`            | Netris controller host key in existing secret. Ignored if `controller.host` is set                            | `host`                     |
| `controllerCreds.login.secretName`    | Name of existing secret to use for Netris controller login. Ignored if `controller.login` is set              | `netris-creds`             |
| `controllerCreds.login.key`           | Netris controller login key in existing secret. Ignored if `controller.login` is set                          | `login`                    |
| `controllerCreds.password.secretName` | Name of existing secret to use for Netris controller password. Ignored if `controller.password` is set        | `netris-creds`             |
| `controllerCreds.password.key`        | Netris controller password key in existing secret. Ignored if `controller.password` is set                    | `password`                 |
| `logLevel`                            | Log level of netris-operator. Allowed values: `info` or `debug`                                               | `info`                     |
| `requeueInterval`                     | Requeue interval in seconds for the netris-operator                                                           | `15`                       |
| `calicoASNRange`                      | Set Nodes ASN range. Used when Netris-Operator manages Calico CNI                                             | `4230000000-4239999999`    |
| `l4lbTenant`                          | Set the default Tenant for L4LB resources. If set, a tenant autodetection for L4LB resources will be disabled | `""`                       |
| `vpcid`                               | Set the VPC ID (integer) where to create LB                                                                   | `1`                        |
