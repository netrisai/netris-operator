# Netris-Operator Deployment

Netris-operator runs within your Kubernetes cluster as a deployment resource. It utilizes CustomResourceDefinitions to configure netris cloud resources.

It is deployed using regular YAML manifests, like any other application on Kubernetes.

# Installing with regular manifests 

All resources are included in a single YAML manifest file:

1) Install the CustomResourceDefinitions and netris-operator itself:

```
kubectl apply -f https://github.com/netrisai/netris-operator/releases/download/v0.0.1/netris-operator.yaml
```


2) Create credentials secret for netris-operator:

```
kubectl -n netris-operator create secret generic netris-creds \
  --from-literal=host="http://example.com" \
  --from-literal=login="login" --from-literal=password="pass"
```

# Installing with Helm

As an alternative to the YAML manifests referenced above, we also provide an official Helm chart for installing netris-operator.
## Prerequisites

- Helm v3 only

## Steps

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
