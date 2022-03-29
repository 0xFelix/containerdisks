package generic

import (
	kvirtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/containerdisks/pkg/api"
	"kubevirt.io/kubevirt/tests/libvmi"
)

type generic struct {
	artifactDetails *api.ArtifactDetails
	metadata        *api.Metadata
}

func (c *generic) Metadata() *api.Metadata {
	return c.metadata
}

func (c *generic) Inspect() (*api.ArtifactDetails, error) {
	return c.artifactDetails, nil
}

func (c *generic) VMI(imgRef string) *kvirtv1.VirtualMachineInstance {
	options := []libvmi.Option{
		libvmi.WithRng(),
		libvmi.WithContainerImage(imgRef),
		libvmi.WithResourceMemory("1024M"),
		libvmi.WithTerminationGracePeriod(libvmi.DefaultTestGracePeriod),
	}

	return libvmi.New(libvmi.RandName(c.Metadata().Name), options...)
}

func (c *generic) Tests() []api.ArtifactTest {
	return []api.ArtifactTest{}
}

func New(artifactDetails *api.ArtifactDetails, metadata *api.Metadata) *generic {
	return &generic{artifactDetails: artifactDetails, metadata: metadata}
}
