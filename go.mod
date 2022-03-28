module kubevirt.io/containerdisks

go 1.16

require (
	github.com/containers/image/v5 v5.17.0
	github.com/docker/distribution v2.8.0+incompatible
	github.com/ghodss/yaml v1.0.0
	github.com/google/go-containerregistry v0.8.1-0.20220310143843-f1fa40b162a1
	github.com/onsi/gomega v1.16.0
	github.com/opencontainers/image-spec v1.0.3-0.20220114050600-8b9d41f48198
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/cobra v1.3.0
	github.com/ulikunitz/xz v0.5.10
	k8s.io/api v0.22.5
	k8s.io/apimachinery v0.22.5
	k8s.io/utils v0.0.0-20210930125809-cb0fa318a74b
	kubevirt.io/api v0.0.0-20211129173424-e2813e40f15a
)
