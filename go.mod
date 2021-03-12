module github.com/netrisai/netris-operator

go 1.14

require (
	github.com/go-logr/logr v0.1.0
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/mitchellh/mapstructure v1.4.1 // indirect
	github.com/netrisai/netrisapi v0.0.0-20210312165744-bf305fd67c53
	github.com/onsi/ginkgo v1.12.1
	github.com/onsi/gomega v1.10.1
	github.com/r3labs/diff/v2 v2.9.1
	github.com/sirupsen/logrus v1.8.1 // indirect
	go.uber.org/zap v1.10.0
	golang.org/x/sys v0.0.0-20210309074719-68d13333faf2 // indirect
	gopkg.in/yaml.v2 v2.3.0
	k8s.io/apimachinery v0.18.6
	k8s.io/client-go v0.18.6
	sigs.k8s.io/controller-runtime v0.6.4
)
