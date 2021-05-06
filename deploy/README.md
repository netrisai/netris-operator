# Netris-Operator Deployment

Netris-operator runs within your Kubernetes cluster as a deployment resource. It utilizes CustomResourceDefinitions to configure netris cloud resources.

It is deployed using regular YAML manifests, like any other application on Kubernetes.

# Installing with regular manifests 

All resources are included in a single YAML manifest file:

1) Install the CustomResourceDefinitions and netris-operator itself:

```
kubectl apply -f https://github.com/netrisai/netris-operator/releases/download/v0.3.9/netris-operator.yaml
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

- Kubernetes 1.16+
- Helm 3.1+

Documentation for netris-operator chart can be found 
[here](https://github.com/netrisai/netris-operator/tree/master/deploy/charts/netris-operator).
