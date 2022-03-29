package fedora

import (
	"encoding/json"
	"fmt"
	"strings"

	kvirtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/containerdisks/pkg/api"
	"kubevirt.io/containerdisks/pkg/docs"
	"kubevirt.io/containerdisks/pkg/http"
	"kubevirt.io/containerdisks/pkg/tbu"
	"kubevirt.io/containerdisks/pkg/tests"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/libvmi"
)

type Releases []Release

type Release struct {
	Subvariant string `json:"subvariant"`
	Variant    string `json:"variant"`
	Version    string `json:"version"`
	Link       string `json:"link"`
	Sha256     string `json:"sha256,omitempty"`
	Arch       string `json:"arch"`
	Size       string `json:"size,omitempty"`
}

type fedora struct {
	Version string
	getter  http.Getter
	Arch    string
	Variant string
}

var description string = `<img src="https://upload.wikimedia.org/wikipedia/commons/thumb/3/3f/Fedora_logo.svg/240px-Fedora_logo.svg.png" alt="drawing" width="15"/> Fedora [Cloud](https://alt.fedoraproject.org/cloud/) images for KubeVirt.
<br />
<br />
Visit [getfedora.org](https://getfedora.org/) to learn more about the Fedora project.`

func (f *fedora) Metadata() *api.Metadata {
	return &api.Metadata{
		Name:                    "fedora",
		Version:                 f.Version,
		Description:             description,
		ExampleCloudInitPayload: docs.CloudInit(),
	}
}

func (f *fedora) Inspect() (*api.ArtifactDetails, error) {
	raw, err := f.getter.GetAll("https://getfedora.org/releases.json")
	if err != nil {
		return nil, fmt.Errorf("error downloading the fedora releases.json file: %v", err)
	}
	releases := Releases{}
	if err := json.Unmarshal(raw, &releases); err != nil {
		return nil, fmt.Errorf("error parsing the releases.json file: %v", err)
	}
	for _, release := range releases {
		if f.releaseMatches(&release) {
			components := strings.Split(release.Link, "/")
			fileName := components[len(components)-1]
			additionalTag := strings.TrimSuffix(strings.TrimPrefix(fileName, "Fedora-Cloud-Base-"), ".x86_64.qcow2")

			return &api.ArtifactDetails{
				SHA256Sum:            release.Sha256,
				DownloadURL:          release.Link,
				AdditionalUniqueTags: []string{additionalTag},
			}, nil
		}
	}
	return nil, fmt.Errorf("no release information in releases.json for fedora:%q found", f.Version)
}

func (f *fedora) VMI(imgRef string) *kvirtv1.VirtualMachineInstance {
	options := []libvmi.Option{
		tbu.WithSMM(),
		libvmi.WithRng(),
		libvmi.WithUefi(true),
		libvmi.WithContainerImage(imgRef),
		libvmi.WithResourceMemory("1024M"),
		libvmi.WithTerminationGracePeriod(libvmi.DefaultTestGracePeriod),
		libvmi.WithCloudInitNoCloudUserData(
			"#cloud-config\nsystem_info:\n  default_user:\n    name: fedora\n    plain_text_passwd: fedora\n    lock_passwd: False\nwrite_files:\n  - path: /etc/profile.d/disable-bracketed-paste.sh\n    content: |\n      bind 'set enable-bracketed-paste off'\n    permissions: '0755'\n",
			false,
		),
	}

	return libvmi.New(libvmi.RandName(f.Metadata().Name), options...)
}

func (f *fedora) Tests() []api.ArtifactTest {
	return []api.ArtifactTest{
		console.SecureBootExpecter,
		console.LoginToFedora,
		tests.GuestOsInfo,
	}
}

func (f *fedora) releaseMatches(release *Release) bool {
	return release.Version == f.Version &&
		release.Arch == f.Arch &&
		release.Variant == f.Variant &&
		strings.HasSuffix(release.Link, "qcow2")
}

func New(release string) *fedora {
	return &fedora{
		Version: release,
		Arch:    "x86_64",
		Variant: "Cloud",
		getter:  &http.HTTPGetter{},
	}
}
