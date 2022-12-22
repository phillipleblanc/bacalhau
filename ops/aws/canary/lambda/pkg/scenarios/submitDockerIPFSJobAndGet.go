package scenarios

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/filecoin-project/bacalhau/pkg/ipfs"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/rs/zerolog/log"
)

// This test submits a job that uses the Docker executor with an IPFS input.
func SubmitDockerIPFSJobAndGet(ctx context.Context) error {
	client := getClient()

	cm := system.NewCleanupManager()
	j := getSampleDockerIPFSJob()

	// Tests use the cid of the file we uploaded in scenarios_test.go
	if system.GetEnvironment() == system.EnvironmentTest {
		j.Spec.Inputs[0].CID = os.Getenv("BACALHAU_CANARY_TEST_CID")
	}

	submittedJob, err := client.Submit(ctx, j, nil)
	if err != nil {
		return err
	}

	log.Info().Msgf("submitted job: %s", submittedJob.Metadata.ID)

	err = waitUntilCompleted(ctx, client, submittedJob)
	if err != nil {
		return fmt.Errorf("waiting until completed: %s", err)
	}

	results, err := client.GetResults(ctx, submittedJob.Metadata.ID)
	if err != nil {
		return fmt.Errorf("getting results: %s", err)
	}

	if len(results) == 0 {
		return fmt.Errorf("no results found")
	}

	outputDir, err := os.MkdirTemp(os.TempDir(), "submitAndGet")
	if err != nil {
		return fmt.Errorf("making temporary dir: %s", err)
	}
	defer os.RemoveAll(outputDir)

	downloadSettings, err := getIPFSDownloadSettings()
	if err != nil {
		return fmt.Errorf("getting download settings: %s", err)
	}
	downloadSettings.OutputDir = outputDir
	downloadSettings.TimeoutSecs = 600

	err = ipfs.DownloadJob(ctx, cm, submittedJob.Spec.Outputs, results, *downloadSettings)
	if err != nil {
		return fmt.Errorf("downloading job: %s", err)
	}
	files, err := os.ReadDir(filepath.Join(downloadSettings.OutputDir, ipfs.DownloadVolumesFolderName, j.Spec.Outputs[0].Name))
	if err != nil {
		return fmt.Errorf("reading results directory: %s", err)
	}

	for _, file := range files {
		log.Debug().Msgf("downloaded files: %s", file)
	}
	if len(files) != 3 {
		return fmt.Errorf("expected 2 files in output dir, got %d", len(files))
	}
	body, err := os.ReadFile(filepath.Join(downloadSettings.OutputDir, ipfs.DownloadVolumesFolderName, j.Spec.Outputs[0].Name, "checksum.txt"))
	if err != nil {
		return err
	}

	// Tests use the checksum of the data we uploaded in scenarios_test.go
	if system.GetEnvironment() == system.EnvironmentProd {
		err = compareOutput(body, "ea1efa312267e09809ae13f311970863  /inputs/data.tar.gz")
	} else if system.GetEnvironment() == system.EnvironmentTest {
		err = compareOutput(body, "c639efc1e98762233743a75e7798dd9c  /inputs/data.tar.gz")
	}
	if err != nil {
		return fmt.Errorf("testing md5 of input: %s", err)
	}
	body, err = os.ReadFile(filepath.Join(downloadSettings.OutputDir, ipfs.DownloadVolumesFolderName, j.Spec.Outputs[0].Name, "stat.txt"))
	if err != nil {
		return err
	}
	// Tests use the stat of the data we uploaded in scenarios_test.go
	if system.GetEnvironment() == system.EnvironmentProd {
		err = compareOutput(body, "62731802")
	} else if system.GetEnvironment() == system.EnvironmentTest {
		err = compareOutput(body, "21")
	}
	if err != nil {
		return fmt.Errorf("testing ls of input: %s", err)
	}

	return nil
}