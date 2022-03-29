package images

import (
	"fmt"
	"path"
	"sync"
	"time"

	"kubevirt.io/containerdisks/cmd/medius/common"
	"kubevirt.io/containerdisks/pkg/api"
)

func spawnWorkers(workers int, focus string, workerFn func(api.Artifact) error) (*sync.WaitGroup, chan error) {
	count := len(common.Registry)
	errChan := make(chan error, count)
	jobChan := make(chan api.Artifact, count)

	wg := &sync.WaitGroup{}
	wg.Add(workers)
	for x := 0; x < workers; x++ {
		go func() {
			defer wg.Done()
			for a := range jobChan {
				if err := workerFn(a); err != nil {
					common.Logger(a).Error(err)
					errChan <- err
				}
			}
		}()
	}

	fillJobChan(jobChan, focus)
	close(jobChan)

	return wg, errChan
}

func fillJobChan(jobChan chan api.Artifact, focus string) {
	for i, desc := range common.Registry {
		if focus == "" && desc.SkipWhenNotFocused {
			continue
		}

		if focus != "" && focus != desc.Artifact.Metadata().Describe() {
			continue
		}

		jobChan <- common.Registry[i].Artifact
	}
}

func prepareTags(registry string, metadata *api.Metadata, artifactDetails *api.ArtifactDetails) []string {
	imageName := path.Join(registry, metadata.Describe())
	names := []string{}
	for _, tag := range artifactDetails.AdditionalUniqueTags {
		if tag == "" {
			continue
		}
		names = append(names, fmt.Sprintf("%s:%s", path.Join(registry, metadata.Name), tag))
	}
	// the least specific tag is last
	names = append(names, imageName)
	return names
}

func prepareTimestampTag(registry string, metadata *api.Metadata) string {
	return fmt.Sprintf(
		"%s-%s",
		path.Join(registry, metadata.Describe()),
		time.Now().Format("0601021504"),
	)
}
