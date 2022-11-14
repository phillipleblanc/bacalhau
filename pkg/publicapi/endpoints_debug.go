package publicapi

import (
	"encoding/json"
	"net/http"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/requesternode"

	"github.com/filecoin-project/bacalhau/pkg/computenode"

	"github.com/filecoin-project/bacalhau/pkg/system"
)

type debugResponse struct {
	AvailableComputeCapacity model.ResourceUsageData   `json:"AvailableComputeCapacity"`
	RequesterJobs            []requesternode.ActiveJob `json:"RequesterJobs"`
	ComputeJobs              []computenode.ActiveJob   `json:"ComputeJobs"`
}

// Returns debug information on what the current node is doing.
func (a *APIServer) debug(res http.ResponseWriter, req *http.Request) {
	ctx, span := system.GetSpanFromRequest(req, "apiServer/debug")
	defer span.End()

	responseObj := debugResponse{
		AvailableComputeCapacity: a.ComputeNode.GetAvailableCapacity(ctx),
		RequesterJobs:            a.Requester.GetActiveJobs(ctx),
		ComputeJobs:              a.ComputeNode.GetActiveJobs(ctx),
	}

	res.WriteHeader(http.StatusOK)
	err := json.NewEncoder(res).Encode(responseObj)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
	}
}
