//go:build unit || !integration

package bacalhau

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/computenode"
	"github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/publicapi"
	"github.com/filecoin-project/bacalhau/pkg/requesternode"
	"github.com/filecoin-project/bacalhau/pkg/system"
	devstack_tests "github.com/filecoin-project/bacalhau/pkg/test/devstack"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type CreateSuite struct {
	suite.Suite
	rootCmd *cobra.Command
}

func TestCreateSuite(t *testing.T) {
	suite.Run(t, new(CreateSuite))
}

// before each test
func (s *CreateSuite) SetupTest() {
	logger.ConfigureTestLogging(s.T())
	require.NoError(s.T(), system.InitConfigForTesting(s.T()))
	s.rootCmd = RootCmd
}

func (s *CreateSuite) TestCreateJSON_GenericSubmit() {
	tests := []struct {
		numberOfJobs int
	}{
		{numberOfJobs: 1}, // Test for one
		{numberOfJobs: 5}, // Test for five
	}

	Fatal = FakeFatalErrorHandler

	for i, tc := range tests {
		func() {
			ctx := context.Background()
			devstack, _ := devstack_tests.SetupTest(ctx, s.T(), 1, 0, false,
				computenode.NewDefaultComputeNodeConfig(),
				requesternode.NewDefaultRequesterNodeConfig(),
			)

			*OC = *NewCreateOptions()

			host := devstack.Nodes[0].APIServer.Host
			port := fmt.Sprint(devstack.Nodes[0].APIServer.Port)
			_, out, err := ExecuteTestCobraCommand(s.T(), s.rootCmd, "create",
				"--api-host", host,
				"--api-port", port,
				"../../testdata/job.json",
			)
			require.NoError(s.T(), err, "Error submitting job. Run - Number of Jobs: %d. Job number: %d", tc.numberOfJobs, i)

			jobID := system.FindJobIDInTestOutput(out)
			uuidRegex := regexp.MustCompile(`[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}`)
			require.Regexp(s.T(), uuidRegex, jobID, "Job ID should be a UUID")

			c := publicapi.NewAPIClient(devstack.Nodes[0].APIServer.GetURI())
			job, _, err := c.Get(ctx, strings.TrimSpace(jobID))
			require.NoError(s.T(), err)
			require.NotNil(s.T(), job, "Failed to get job with ID: %s", jobID)
		}()
	}
}

func (s *CreateSuite) TestCreateYAML_GenericSubmit() {
	tests := []struct {
		numberOfJobs int
	}{
		{numberOfJobs: 1}, // Test for one
		{numberOfJobs: 5}, // Test for five
	}

	Fatal = FakeFatalErrorHandler

	for i, tc := range tests {

		testFiles := []string{"../../testdata/job.yaml", "../../testdata/job-url.yaml"}

		for _, testFile := range testFiles {
			func() {
				ctx := context.Background()
				devstack, _ := devstack_tests.SetupTest(ctx, s.T(), 1, 0, false,
					computenode.NewDefaultComputeNodeConfig(),
					requesternode.NewDefaultRequesterNodeConfig(),
				)

				*OC = *NewCreateOptions()

				host := devstack.Nodes[0].APIServer.Host
				port := fmt.Sprint(devstack.Nodes[0].APIServer.Port)
				_, out, err := ExecuteTestCobraCommand(s.T(), s.rootCmd, "create",
					"--api-host", host,
					"--api-port", port,
					testFile,
				)

				require.NoError(s.T(), err, "Error submitting job. Run - Number of Jobs: %d. Job number: %d", tc.numberOfJobs, i)

				jobID := system.FindJobIDInTestOutput(out)
				uuidRegex := regexp.MustCompile(`[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}`)
				require.Regexp(s.T(), uuidRegex, jobID, "Job ID should be a UUID")

				c := publicapi.NewAPIClient(devstack.Nodes[0].APIServer.GetURI())
				job, _, err := c.Get(ctx, strings.TrimSpace(jobID))
				require.NoError(s.T(), err)
				require.NotNil(s.T(), job, "Failed to get job with ID: %s\nOutput: %s", out)
			}()
		}
	}
}

func (s *CreateSuite) TestCreateFromStdin() {
	testFile := "../../testdata/job.yaml"

	Fatal = FakeFatalErrorHandler

	devstack, _ := devstack_tests.SetupTest(context.Background(), s.T(), 1, 0, false,
		computenode.NewDefaultComputeNodeConfig(),
		requesternode.NewDefaultRequesterNodeConfig(),
	)

	*OC = *NewCreateOptions()

	testSpec, err := os.Open(testFile)
	require.NoError(s.T(), err)

	host := devstack.Nodes[0].APIServer.Host
	port := fmt.Sprint(devstack.Nodes[0].APIServer.Port)
	_, out, err := ExecuteTestCobraCommandWithStdin(s.T(), s.rootCmd, testSpec, "create",
		"--api-host", host,
		"--api-port", port,
	)

	require.NoError(s.T(), err, "Error submitting job.")

	jobID := system.FindJobIDInTestOutput(out)
	uuidRegex := regexp.MustCompile(`[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}`)
	require.Regexp(s.T(), uuidRegex, jobID, "Job ID should be a UUID")

	// Now run describe on the ID we got back
	_, out, err = ExecuteTestCobraCommand(s.T(), s.rootCmd, "describe",
		"--api-host", host,
		"--api-port", port,
		jobID,
	)

	require.NoError(s.T(), err, "Error describing job.")
}

func (s *CreateSuite) TestCreateDontPanicOnNoInput() {
	Fatal = FakeFatalErrorHandler

	type commandReturn struct {
		c   *cobra.Command
		out string
		err error
	}

	commandChan := make(chan commandReturn, 1)

	go func() {
		c, out, err := ExecuteTestCobraCommand(s.T(), RootCmd, "create")

		commandChan <- commandReturn{c: c, out: out, err: err}
	}()
	time.Sleep(1 * time.Second)

	stdinErr := os.Stdin.Close()
	if stdinErr != nil && !errors.Is(stdinErr, os.ErrClosed) {
		require.NoError(s.T(), stdinErr, "Error closing stdin")
	}

	commandReturnValue := <-commandChan

	// For some reason I can't explain, this only works when running in debug.
	// require.Contains(s.T(), commandReturnValue.out, "Ctrl+D", "Waiting message should contain Ctrl+D")

	errorOutputMap := make(map[string]interface{})
	for _, o := range strings.Split(commandReturnValue.out, "\n") {
		err := model.YAMLUnmarshalWithMax([]byte(o), &errorOutputMap)
		if err != nil {
			continue
		}
	}

	require.Contains(s.T(), errorOutputMap["Message"], "The job provided is invalid", "Output message should error properly.")
	require.Equal(s.T(), int(errorOutputMap["Code"].(float64)), 1, "Expected no error when no input is provided")
}

func (s *CreateSuite) TestCreateDontPanicOnEmptyFile() {
	Fatal = FakeFatalErrorHandler

	type commandReturn struct {
		c   *cobra.Command
		out string
		err error
	}

	commandChan := make(chan commandReturn, 1)

	go func() {
		c, out, err := ExecuteTestCobraCommand(s.T(), RootCmd, "create", "../../testdata/empty.yaml")

		commandChan <- commandReturn{c: c, out: out, err: err}
	}()
	time.Sleep(1 * time.Second)

	stdinErr := os.Stdin.Close()
	require.NoError(s.T(), stdinErr, "Error closing stdin")

	commandReturnValue := <-commandChan

	errorOutputMap := make(map[string]interface{})
	for _, o := range strings.Split(commandReturnValue.out, "\n") {
		err := model.YAMLUnmarshalWithMax([]byte(o), &errorOutputMap)
		if err != nil {
			continue
		}
	}

	require.Contains(s.T(), errorOutputMap["Message"], "The job provided is invalid", "Output message should error properly.")
	require.Equal(s.T(), int(errorOutputMap["Code"].(float64)), 1, "Expected no error when no input is provided")
}
