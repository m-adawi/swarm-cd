package web

import (
	"errors"
	"net/http"
	"sort"

	"github.com/gin-gonic/gin"
	"github.com/m-adawi/swarm-cd/swarmcd"
	"github.com/m-adawi/swarm-cd/util"
)

// getStackServicesFn is a seam over swarmcd.GetStackServices so the controller
// can be unit-tested without a live Docker daemon.
var getStackServicesFn = swarmcd.GetStackServices

func getStacks(ctx *gin.Context) {
	stacksStatus := swarmcd.GetStackStatus()
	var stacks []map[string]string
	for k, v := range stacksStatus {
		stacks = append(stacks, map[string]string{
			"Name": k,
			"Error": v.Error,
			"RepoURL": v.RepoURL,
			"Revision": v.Revision,
		})
	}
	sort.Slice(stacks, func(i, j int) bool {
		return stacks[i]["Name"] < stacks[j]["Name"]
	})
	ctx.JSON(http.StatusOK, stacks)
}

func getStackServices(ctx *gin.Context) {
	name := ctx.Param("name")
	util.Logger.Debug("handling stack services request", "stack", name)

	services, err := getStackServicesFn(name)
	if err != nil {
		if errors.Is(err, swarmcd.ErrStackNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "stack not found"})
			return
		}
		util.Logger.Error("failed to get stack services", "stack", name, "error", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "could not list services"})
		return
	}

	ctx.JSON(http.StatusOK, services)
}