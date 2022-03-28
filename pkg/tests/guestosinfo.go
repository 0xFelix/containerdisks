package tests

import (
	kvirtv1 "kubevirt.io/api/core/v1"
	kvirtcli "kubevirt.io/client-go/kubecli"
)

func GuestOsInfo(vmi *kvirtv1.VirtualMachineInstance) error {
	client, err := kvirtcli.GetKubevirtClient()
	if err != nil {
		return err
	}

	if _, err = client.VirtualMachineInstance(vmi.Namespace).GuestOsInfo(vmi.Name); err != nil {
		return err
	}

	return nil
}
