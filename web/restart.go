package web

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/gin-gonic/gin"
	"github.com/m-adawi/swarm-cd/swarmcd"
)

func restartStack(ctx *gin.Context) {
	name := ctx.Param("name")
	if !swarmcd.StackExists(name) {
		ctx.JSON(http.StatusNotFound, gin.H{
			"error": fmt.Sprintf("stack '%s' not found", name),
		})
		return
	}

	api := swarmcd.GetDockerServiceAPI()
	count, err := forceUpdateServices(api, name, "")
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("failed to restart stack '%s': %v", name, err),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message":            fmt.Sprintf("restarted %d service(s) in stack '%s'", count, name),
		"services_restarted": count,
	})
}

func restartService(ctx *gin.Context) {
	stackName := ctx.Param("name")
	serviceName := ctx.Param("service")

	if !swarmcd.StackExists(stackName) {
		ctx.JSON(http.StatusNotFound, gin.H{
			"error": fmt.Sprintf("stack '%s' not found", stackName),
		})
		return
	}

	api := swarmcd.GetDockerServiceAPI()
	count, err := forceUpdateServices(api, stackName, serviceName)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("failed to restart service '%s' in stack '%s': %v", serviceName, stackName, err),
		})
		return
	}

	if count == 0 {
		ctx.JSON(http.StatusNotFound, gin.H{
			"error": fmt.Sprintf("no service matching '%s' found in stack '%s'", serviceName, stackName),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message":            fmt.Sprintf("restarted service '%s' in stack '%s'", serviceName, stackName),
		"services_restarted": count,
	})
}

func restartAll(ctx *gin.Context) {
	api := swarmcd.GetDockerServiceAPI()
	names := swarmcd.GetStackNames()

	total := 0
	for _, name := range names {
		count, err := forceUpdateServices(api, name, "")
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("failed to restart stack '%s': %v", name, err),
			})
			return
		}
		total += count
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message":            fmt.Sprintf("restarted %d service(s) across %d stack(s)", total, len(names)),
		"services_restarted": total,
	})
}

// forceUpdateServices lists Docker services for the given stack and increments
// their ForceUpdate counter, causing Docker to redeploy them. If serviceName
// is non-empty, only services whose name starts with "<stack>_<serviceName>"
// are restarted.
func forceUpdateServices(api swarmcd.ServiceAPI, stackName, serviceName string) (int, error) {
	bg := context.Background()

	services, err := api.ServiceList(bg, types.ServiceListOptions{
		Filters: filters.NewArgs(
			filters.Arg("label", "com.docker.stack.namespace="+stackName),
		),
	})
	if err != nil {
		return 0, fmt.Errorf("failed to list services: %w", err)
	}

	count := 0
	for _, svc := range services {
		if serviceName != "" {
			prefix := stackName + "_" + serviceName
			if !strings.HasPrefix(svc.Spec.Name, prefix) {
				continue
			}
		}

		svc.Spec.TaskTemplate.ForceUpdate++
		if _, err := api.ServiceUpdate(bg, svc.ID, svc.Version, svc.Spec, types.ServiceUpdateOptions{}); err != nil {
			return count, fmt.Errorf("failed to update service %s: %w", svc.Spec.Name, err)
		}
		count++
	}

	return count, nil
}
