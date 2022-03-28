package images

import (
	"sync"

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
