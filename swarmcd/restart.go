package swarmcd

import (
	"context"
	"sync"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
)

// ServiceAPI is the subset of the Docker API used for service operations.
// It is satisfied by the client returned from dockerCli.Client().
type ServiceAPI interface {
	ServiceList(ctx context.Context, options types.ServiceListOptions) ([]swarm.Service, error)
	ServiceUpdate(ctx context.Context, serviceID string, version swarm.Version, service swarm.ServiceSpec, options types.ServiceUpdateOptions) (swarm.ServiceUpdateResponse, error)
}

var (
	serviceAPI     ServiceAPI
	serviceAPIMu   sync.Mutex
	serviceAPIInit bool
)

// GetDockerServiceAPI returns a lazily-initialized Docker service client.
func GetDockerServiceAPI() ServiceAPI {
	serviceAPIMu.Lock()
	defer serviceAPIMu.Unlock()
	if !serviceAPIInit {
		serviceAPI = dockerCli.Client()
		serviceAPIInit = true
	}
	return serviceAPI
}

// SetServiceAPIForTest replaces the Docker service client for testing
// and returns a cleanup function that restores the original.
func SetServiceAPIForTest(api ServiceAPI) func() {
	serviceAPIMu.Lock()
	old := serviceAPI
	oldInit := serviceAPIInit
	serviceAPI = api
	serviceAPIInit = true
	serviceAPIMu.Unlock()
	return func() {
		serviceAPIMu.Lock()
		serviceAPI = old
		serviceAPIInit = oldInit
		serviceAPIMu.Unlock()
	}
}

// StackExists reports whether the named stack is known to SwarmCD.
func StackExists(name string) bool {
	stateMu.RLock()
	defer stateMu.RUnlock()
	_, ok := stackStatus[name]
	return ok
}

// GetStackNames returns the names of all managed stacks.
func GetStackNames() []string {
	stateMu.RLock()
	defer stateMu.RUnlock()
	names := make([]string, 0, len(stackStatus))
	for k := range stackStatus {
		names = append(names, k)
	}
	return names
}
