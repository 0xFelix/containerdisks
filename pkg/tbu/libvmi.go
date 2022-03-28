package tbu

import (
	"k8s.io/utils/pointer"
	kvirtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/kubevirt/tests/libvmi"
)

// To be upstreamed
// Additions for kubevirt.io/kubevirt/tests/libvmi

func WithSMM() libvmi.Option {
	return func(vmi *kvirtv1.VirtualMachineInstance) {
		if vmi.Spec.Domain.Features == nil {
			vmi.Spec.Domain.Features = &kvirtv1.Features{}
		}

		vmi.Spec.Domain.Features.SMM = &kvirtv1.FeatureState{
			Enabled: pointer.Bool(true),
		}
	}
}
