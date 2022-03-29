package build

import (
	"errors"
	"fmt"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/google/go-containerregistry/pkg/v1/types"
)

const (
	LabelShaSum        = "shasum"
	AnnotationVerified = "verified"
)

func BuildContainerDisk(imgPath string, checksum string) (v1.Image, error) {
	img := empty.Image
	img = mutate.MediaType(img, types.OCIManifestSchema1)
	img = mutate.ConfigMediaType(img, types.OCIConfigJSON)

	layerStream, errChan := StreamLayer(imgPath)
	layer, err := tarball.LayerFromReader(layerStream, tarball.WithMediaType(types.OCILayer))
	if err != nil {
		return nil, fmt.Errorf("error creating an image layer from disk: %v", err)
	}

	img, err = mutate.AppendLayers(img, layer)
	if err != nil {
		return nil, fmt.Errorf("error appending the image layer: %v", err)
	}

	img, err = mutate.Config(img, v1.Config{Labels: map[string]string{LabelShaSum: checksum}})
	if err != nil {
		return nil, fmt.Errorf("error appending labels to the image: %v", err)
	}

	img, ok := mutate.Annotations(img, map[string]string{AnnotationVerified: "false"}).(v1.Image)
	if !ok {
		return nil, errors.New("error appending annotations to the image")
	}

	if err := <-errChan; err != nil {
		return nil, fmt.Errorf("error creating the tar file with the disk: %v", err)
	}

	return img, nil
}
