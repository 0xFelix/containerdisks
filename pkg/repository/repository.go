package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/containers/image/v5/image"
	"github.com/containers/image/v5/manifest"
	"github.com/containers/image/v5/transports/alltransports"
	"github.com/containers/image/v5/types"
	"github.com/docker/distribution/registry/api/errcode"
	v2 "github.com/docker/distribution/registry/api/v2"
	"github.com/google/go-containerregistry/pkg/crane"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	imgspecv1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
)

type ImageInfo struct {
	Tag           string `json:",omitempty"`
	Created       *time.Time
	DockerVersion string
	Labels        map[string]string
	Annotations   map[string]string
	Architecture  string
	Os            string
	Layers        []string
	Env           []string
}

type Repository interface {
	ImageMetadata(imgRef string) (*ImageInfo, error)
	PullImage(imgRef string, insecure bool) (v1.Image, error)
	PushImage(img v1.Image, imgRef string) error
	MutateAnnotations(img v1.Image) (v1.Image, error)
}

type RepositoryImpl struct {
}

func (r RepositoryImpl) ImageMetadata(imgRef string, insecure bool) (imageInfo *ImageInfo, retErr error) {
	sys := &types.SystemContext{
		OCIInsecureSkipTLSVerify: insecure,
	}
	if insecure {
		sys.DockerInsecureSkipTLSVerify = types.OptionalBoolTrue
	}
	ctx := context.Background()
	src, err := parseImageSource(ctx, sys, fmt.Sprintf("docker://%s", imgRef))
	if err != nil {
		return nil, errors.Wrapf(err, "error parsing image")
	}

	defer func() {
		if err := src.Close(); err != nil {
			retErr = errors.Wrapf(retErr, fmt.Sprintf("(could not close image: %v) ", err))
		}
	}()

	img, err := image.FromUnparsedImage(ctx, sys, image.UnparsedInstance(src, nil))
	if err != nil {
		return nil, errors.Wrapf(err, "Error parsing manifest for image")
	}

	parsedManifest, err := parseManifest(ctx, img)
	if err != nil {
		return nil, errors.Wrapf(err, "Error parsing manifest of image")
	}

	imgInspect, err := img.Inspect(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "Error inspecting image")
	}
	imageInfo = &ImageInfo{
		Tag: imgInspect.Tag,
		// Digest is set below.
		Created:       imgInspect.Created,
		DockerVersion: imgInspect.DockerVersion,
		Labels:        imgInspect.Labels,
		Architecture:  imgInspect.Architecture,
		Os:            imgInspect.Os,
		Layers:        imgInspect.Layers,
		Env:           imgInspect.Env,
		Annotations:   parsedManifest.Annotations,
	}

	return imageInfo, nil
}

func (r RepositoryImpl) PullImage(imgRef string, insecure bool) (v1.Image, error) {
	options := []crane.Option{
		crane.WithContext(context.Background()),
	}

	if insecure {
		options = append(options, crane.Insecure)
	}

	img, err := crane.Pull(imgRef, options...)
	if err != nil {
		return nil, fmt.Errorf("error pulling image %s: %w", imgRef, err)
	}

	return img, nil
}

func (r RepositoryImpl) PushImage(img v1.Image, imgRef string) error {
	if err := crane.Push(img, imgRef, crane.WithContext(context.Background())); err != nil {
		return fmt.Errorf("error pushing image %q: %v", img, err)
	}

	return nil
}

func (r RepositoryImpl) MutateAnnotations(img v1.Image, annotations map[string]string) (v1.Image, error) {
	img, ok := mutate.Annotations(img, annotations).(v1.Image)
	if !ok {
		return nil, errors.New("error mutating annotations of the image")
	}

	return img, nil
}

func parseImageSource(ctx context.Context, sys *types.SystemContext, name string) (types.ImageSource, error) {
	ref, err := alltransports.ParseImageName(name)
	if err != nil {
		return nil, err
	}

	return ref.NewImageSource(ctx, sys)
}

func parseManifest(ctx context.Context, img types.Image) (*manifest.OCI1, error) {
	manifestBlob, manifestMIMEType, err := img.Manifest(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to get manifest of image")
	}

	if manifest.NormalizedMIMEType(manifestMIMEType) != imgspecv1.MediaTypeImageManifest {
		return nil, fmt.Errorf("unsupported image type %s, can only work with OCIv1 images", manifestMIMEType)
	}

	parsedManifest, err := manifest.OCI1FromManifest(manifestBlob)
	if err != nil {
		return nil, errors.Wrapf(err, "Error parsing OCIv1 manifest of image")
	}

	return parsedManifest, nil
}

func IsManifestUnknownError(err error) bool {
	ec := getErrorCode(err)
	if ec == nil {
		return false
	}

	switch ec.ErrorCode() {
	case v2.ErrorCodeManifestUnknown:
		return true
	default:
		return false
	}
}

func IsRepositoryUnknownError(err error) bool {
	ec := getErrorCode(err)
	if ec == nil {
		return false
	}

	switch ec.ErrorCode() {
	case v2.ErrorCodeNameUnknown:
		return true
	default:
		return false
	}
}

func IsTagUnknownError(err error) bool {
	ec := getErrorCode(err)
	if ec == nil {
		return false
	}

	if ec.ErrorCode().Error() == "unknown" {
		// errors like this have no explicit error handling: "unknown: Tag 5.2 was deleted or has expired. To pull, revive via time machine"
		if strings.Contains(err.Error(), "was deleted or has expired. To pull, revive via time machine") {
			return true
		}
	}
	return false
}

func getErrorCode(err error) errcode.ErrorCoder {
	for {
		if unwrapped := errors.Unwrap(err); unwrapped != nil {
			err = unwrapped
		} else {
			break
		}
	}

	errs, ok := err.(errcode.Errors)
	if !ok || len(errs) == 0 {
		return nil
	}
	err = errs[0]
	ec, ok := err.(errcode.ErrorCoder)
	if !ok {
		return nil
	}
	return ec
}
