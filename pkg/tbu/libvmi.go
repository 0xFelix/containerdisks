package tbu

import (
	"encoding/base64"

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

// WithCloudInitConfigDriveUserData adds cloud-init config-drive user data.
func WithCloudInitConfigDriveUserData(data string, b64Encoding bool) libvmi.Option {
	return func(vmi *kvirtv1.VirtualMachineInstance) {
		diskName, bus := "disk1", "virtio"
		addDiskVolumeWithCloudInitConfigDrive(vmi, diskName, bus)

		volume := getVolume(vmi, diskName)
		if b64Encoding {
			encodedData := base64.StdEncoding.EncodeToString([]byte(data))
			volume.CloudInitConfigDrive.UserDataBase64 = encodedData
		} else {
			volume.CloudInitConfigDrive.UserData = data
		}
	}
}

func addDiskVolumeWithCloudInitConfigDrive(vmi *kvirtv1.VirtualMachineInstance, diskName, bus string) {
	addDisk(vmi, newDisk(diskName, bus))
	v := newVolume(diskName)
	setCloudInitConfigDrive(&v, &kvirtv1.CloudInitConfigDriveSource{})
	addVolume(vmi, v)
}

func setCloudInitConfigDrive(volume *kvirtv1.Volume, source *kvirtv1.CloudInitConfigDriveSource) {
	volume.VolumeSource = kvirtv1.VolumeSource{CloudInitConfigDrive: source}
}

// Rest below is copied from kubevirt/tests/libvmi/storage.go

func addDisk(vmi *kvirtv1.VirtualMachineInstance, disk kvirtv1.Disk) {
	if !diskExists(vmi, disk) {
		vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, disk)
	}
}

func diskExists(vmi *kvirtv1.VirtualMachineInstance, disk kvirtv1.Disk) bool {
	for _, d := range vmi.Spec.Domain.Devices.Disks {
		if d.Name == disk.Name {
			return true
		}
	}
	return false
}

func newDisk(name, bus string) kvirtv1.Disk {
	return kvirtv1.Disk{
		Name: name,
		DiskDevice: kvirtv1.DiskDevice{
			Disk: &kvirtv1.DiskTarget{
				Bus: bus,
			},
		},
	}
}

func newVolume(name string) kvirtv1.Volume {
	return kvirtv1.Volume{Name: name}
}

func addVolume(vmi *kvirtv1.VirtualMachineInstance, volume kvirtv1.Volume) {
	if !volumeExists(vmi, volume) {
		vmi.Spec.Volumes = append(vmi.Spec.Volumes, volume)
	}
}

func volumeExists(vmi *kvirtv1.VirtualMachineInstance, volume kvirtv1.Volume) bool {
	for _, v := range vmi.Spec.Volumes {
		if v.Name == volume.Name {
			return true
		}
	}
	return false
}

func getVolume(vmi *kvirtv1.VirtualMachineInstance, name string) *kvirtv1.Volume {
	for i := range vmi.Spec.Volumes {
		if vmi.Spec.Volumes[i].Name == name {
			return &vmi.Spec.Volumes[i]
		}
	}
	return nil
}
