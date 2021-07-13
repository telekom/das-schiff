module gitlab.devops.telekom.de/schiff/engine/schiff-operator.git

go 1.15

require (
	github.com/go-logr/logr v0.3.0
	github.com/infobloxopen/infoblox-go-client v1.1.1-0.20210326040601-b71324be2432
	github.com/onsi/ginkgo v1.14.1
	github.com/onsi/gomega v1.10.2
	github.com/spf13/viper v1.7.1
	k8s.io/apimachinery v0.19.2
	k8s.io/client-go v0.19.2
	sigs.k8s.io/cluster-api v0.3.11
	sigs.k8s.io/cluster-api-provider-vsphere v0.7.4
	sigs.k8s.io/controller-runtime v0.7.0
)
