package images

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	kvirtv1 "kubevirt.io/api/core/v1"
	kvirtcli "kubevirt.io/client-go/kubecli"
	"kubevirt.io/containerdisks/cmd/medius/common"
	"kubevirt.io/containerdisks/pkg/api"
	"kubevirt.io/containerdisks/pkg/build"
	"kubevirt.io/containerdisks/pkg/repository"
)

func NewVerifyImagesCommand(options *common.Options) *cobra.Command {
	options.VerifyImagesOptions = common.VerifyImageOptions{
		Workers:   1,
		Namespace: "kubevirt",
		Timeout:   600,
	}

	verifyCmd := &cobra.Command{
		Use:   "verify",
		Short: "Verify that containerdisks are bootable and guests are working",
		Run: func(cmd *cobra.Command, args []string) {
			if options.VerifyImagesOptions.ClusterRegistry == "" {
				options.VerifyImagesOptions.ClusterRegistry = options.Registry
			}

			client, err := kvirtcli.GetKubevirtClient()
			if err != nil {
				logrus.Fatal(err)
			}

			wg, errChan := spawnWorkers(options.VerifyImagesOptions.Workers, options.Focus, func(a api.Artifact) error {
				return verifyArtifact(a, options, client)
			})
			wg.Wait()

			select {
			case <-errChan:
				os.Exit(1)
			default:
				os.Exit(0)
			}
		},
	}
	verifyCmd.Flags().IntVar(&options.VerifyImagesOptions.Workers, "workers", options.VerifyImagesOptions.Workers, "Number of parallel workers")
	verifyCmd.Flags().StringVar(&options.VerifyImagesOptions.ClusterRegistry, "cluster-registry", "", "Registry to use inside the cluster")
	verifyCmd.Flags().StringVar(&options.VerifyImagesOptions.Namespace, "namespace", options.VerifyImagesOptions.Namespace, "Namespace to run verify in")
	verifyCmd.Flags().IntVar(&options.VerifyImagesOptions.Timeout, "timeout", options.VerifyImagesOptions.Timeout, "Maximum seconds to wait for VM to be running")
	verifyCmd.Flags().AddGoFlagSet(kvirtcli.FlagSet())

	return verifyCmd
}

func verifyArtifact(artifact api.Artifact, options *common.Options, client kvirtcli.KubevirtClient) (err error) {
	log := common.Logger(artifact)

	imgRefs, err := findImgRefs(artifact, options, log)
	if err != nil {
		log.WithError(err).Error("Failed to find containerdisks")
		return err
	}

	if len(imgRefs) == 0 {
		log.Infof("Found no containerdisks to verify")
		return nil
	}

	imgRef := imgRefs[0]
	if options.Registry != options.VerifyImagesOptions.ClusterRegistry {
		imgRef = strings.Replace(imgRef, options.Registry, options.VerifyImagesOptions.ClusterRegistry, 1)
	}

	log.Info("Creating VMI")
	vmi := artifact.VMI(imgRef)
	vmiClient := client.VirtualMachineInstance(options.VerifyImagesOptions.Namespace)
	if vmi, err = vmiClient.Create(vmi); err != nil {
		log.WithError(err).Error("Failed to create VMI")
		return err
	}

	defer func() {
		if err = vmiClient.Delete(vmi.Name, &metav1.DeleteOptions{}); err != nil {
			log.WithError(err).Error("Failed to delete VMI")
		}
	}()

	log.Info("Waiting for VMI to be running")
	if err = waitVMIRunning(vmi.Name, vmiClient, options.VerifyImagesOptions.Timeout); err != nil {
		log.WithError(err).Error("VMI not running")
		return err
	}

	log.Info("Running tests on VMI")
	for _, testFn := range artifact.Tests() {
		if err = testFn(vmi); err != nil {
			log.WithError(err).Error("Failed to verify VMI")
			return err
		}
	}

	imgRefs = append(imgRefs, prepareTimestampTag(options.Registry, artifact.Metadata()))
	if err = pushImages(imgRefs, options, log); err != nil {
		log.WithError(err).Error("Failed to update containerdisks")
		return err
	}

	return err
}

func findImgRefs(artifact api.Artifact, options *common.Options, log *logrus.Entry) ([]string, error) {
	metadata := artifact.Metadata()
	artifactInfo, err := artifact.Inspect()
	if err != nil {
		return nil, fmt.Errorf("error introspecting artifact %q: %v", metadata.Describe(), err)
	}
	log.Infof("Remote artifact checksum: %q", artifactInfo.SHA256Sum)

	repo := repository.RepositoryImpl{}
	imgRefs := []string{}
	for _, imgRef := range prepareTags(options.Registry, metadata, artifactInfo) {
		imgInfo, err := repo.ImageMetadata(imgRef, options.AllowInsecureRegistry)
		if err != nil {
			log.WithError(err).Errorf("Failed to get metadata of %s", imgRef)
			continue
		}

		if artifactInfo.SHA256Sum == imgInfo.Labels[build.LabelShaSum] {
			verified, err := strconv.ParseBool(imgInfo.Annotations[build.AnnotationVerified])
			if err != nil {
				log.WithError(err).Errorf("Failed to parse verified annotation of %s", imgRef)
				continue
			}

			if !verified {
				imgRefs = append(imgRefs, imgRef)
			}
		}
	}

	return imgRefs, nil
}

func pushImages(imgRefs []string, options *common.Options, log *logrus.Entry) (err error) {
	repo := repository.RepositoryImpl{}

	img, err := repo.PullImage(imgRefs[0], options.AllowInsecureRegistry)
	if err != nil {
		log.WithError(err).Error("Failed to pull image")
		return err
	}

	img, err = repo.MutateAnnotations(img, map[string]string{build.AnnotationVerified: "true"})
	if err != nil {
		log.WithError(err).Error("Failed to mutate annotations")
		return err
	}

	for _, imgRef := range imgRefs {
		if !options.DryRun {
			log.Infof("Pushing %s", imgRef)
			if err = repo.PushImage(img, imgRef); err != nil {
				log.WithError(err).Error("Failed to push image")
				return err
			}
		} else {
			log.Infof("Dry run enabled, not pushing %s", imgRef)
		}
	}

	return nil
}

func waitVMIRunning(name string, client kvirtcli.VirtualMachineInstanceInterface, timeout int) error {
	return wait.PollImmediate(time.Second, time.Duration(timeout)*time.Second, func() (bool, error) {
		vmi, err := client.Get(name, &metav1.GetOptions{})

		if err != nil {
			return false, err
		}

		switch vmi.Status.Phase {
		case kvirtv1.Running:
			return true, nil
		default:
			return false, nil
		}
	})
}
