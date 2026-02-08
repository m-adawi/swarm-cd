package web

import (
	"net/http"
	"sort"

	"github.com/gin-gonic/gin"
	"github.com/m-adawi/swarm-cd/swarmcd"
	"github.com/m-adawi/swarm-cd/util"
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

type webhookRequest struct {
	Stack string `json:"stack"`
}

func postWebhook(ctx *gin.Context) {
	var req webhookRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		// If no body or invalid JSON, update all stacks
		req.Stack = ""
	}

	if req.Stack == "" {
		util.Logger.Info("webhook: triggered update for all stacks")
		swarmcd.UpdateAllStacks()
		ctx.JSON(http.StatusOK, gin.H{"message": "all stacks update triggered"})
		return
	}

	util.Logger.Info("webhook: triggered update for stack", "stack", req.Stack)
	err := swarmcd.UpdateStack(req.Stack)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"message": "stack update triggered", "stack": req.Stack})
}

func webhookAuthMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		webhookKey := util.GetWebhookKey()
		if webhookKey == "" {
			util.Logger.Warn("webhook: no webhook key configured, rejecting request")
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "webhook not configured"})
			return
		}

		authHeader := ctx.GetHeader("Authorization")
		if authHeader == "" {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing Authorization header"})
			return
		}

		// Support "Bearer <key>" format
		expectedAuth := "Bearer " + webhookKey
		if authHeader != expectedAuth && authHeader != webhookKey {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid webhook key"})
			return
		}

		ctx.Next()
	}
}
