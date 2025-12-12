package web

import (
	"net/http"
	"sort"

	"github.com/gin-gonic/gin"
	"github.com/m-adawi/swarm-cd/swarmcd"
)

func getStacks(ctx *gin.Context) {
	stacksStatus := swarmcd.GetStackStatus()
	var stacks []map[string]string
	for k, v := range stacksStatus {
		stacks = append(stacks, map[string]string{
			"Name":     k,
			"Error":    v.Error,
			"RepoURL":  v.RepoURL,
			"Revision": v.Revision,
		})
	}
	sort.Slice(stacks, func(i, j int) bool {
		return stacks[i]["Name"] < stacks[j]["Name"]
	})
	ctx.JSON(http.StatusOK, stacks)
}

func updateStack(ctx *gin.Context) {
	secret := ctx.Param("secret")
	if secret != swarmcd.Config.Secret {
		ctx.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	name := ctx.Param("name")

	swarmcd.UpdateAllStackInRepo(name)
	ctx.Status(200)
}
