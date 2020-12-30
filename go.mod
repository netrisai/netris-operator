module github.com/netrisai/netris-operator

go 1.13

require (
	github.com/go-logr/logr v0.1.0
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/netrisai/netrisapi v0.0.0-20201229123610-e57db0e1168e
	github.com/onsi/ginkgo v1.12.1
	github.com/onsi/gomega v1.10.1
	golang.org/x/sys v0.0.0-20201223074533-0d417f636930 // indirect
	gopkg.in/yaml.v2 v2.3.0
	k8s.io/apimachinery v0.18.6
	k8s.io/client-go v0.18.6
	sigs.k8s.io/controller-runtime v0.6.4
)
