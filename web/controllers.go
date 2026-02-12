package web

import (
	"net/http"
	"sort"

	"github.com/gin-gonic/gin"
	"github.com/m-adawi/swarm-cd/swarmcd"
)

func getStacks(ctx *gin.Context) {
	stacksStatus := swarmcd.GetStackStatus()
	var stacks []map[string]any
	for k, v := range stacksStatus {
		stacks = append(stacks, map[string]any{
			"Name":      k,
			"Error":     v.Error,
			"RepoURL":   v.RepoURL,
			"Revision":  v.Revision,
			"Templated": v.Templated,
		})
	}
	sort.Slice(stacks, func(i, j int) bool {
		return stacks[i]["Name"].(string) < stacks[j]["Name"].(string)
	})
	ctx.JSON(http.StatusOK, stacks)
}

func getCompose(ctx *gin.Context) {
	stackName := ctx.Param("stackName")
	swarmStack, err := swarmcd.GetSwarmStack(stackName)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	stackBytes, err := swarmStack.ReadStack()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.String(200, string(stackBytes))
}

func getRendered(ctx *gin.Context) {
	stackName := ctx.Param("stackName")
	swarmStack, err := swarmcd.GetSwarmStack(stackName)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	stackBytes, err := swarmStack.GenerateStack()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.String(200, string(stackBytes))
}
