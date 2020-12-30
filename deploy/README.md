# Netris-Operator Deployment

## Installing with regular manifests 

All resources are included in a single YAML manifest file:

1) Create credentials secret for netris-operator:

```
kubectl create secret generic netris-creds --from-literal=host="http://example.com" --from-literal=login="login" --from-literal=password="pass"
```

2) Install the CustomResourceDefinitions and netris-operator itself:

```
kubectl apply -f https://github.com/netrisai/netris-operator/releases/download/v0.0.1/netris-operator.yaml
```

## Installing with Helm

- Helm v3 only

### Option 1: Creds from secret

1) Create credentials secret for netris-operator:

```
kubectl create secret generic netris-creds --from-literal=host="http://example.com" --from-literal=login="login" --from-literal=password="pass"
```

2) Install helm chart

```
helm upgrade -i netris-operator deploy/charts/netris-operator
```

### Option 2: Creds from helm values

 1) Install helm chart with netris controller creds

```
helm upgrade -i netris-operator deploy/charts/netris-operator --set controller.host="http://example.com" --set controller.login="login" --set controller.password="pass"
```
