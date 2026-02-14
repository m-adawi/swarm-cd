package web

import (
	"encoding/json"
	"errors"
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

// StackConfig represents the webhook request payload
type StackConfig struct {
	Type  string  `json:"type"`
	Stack *string `json:"stack"`
}

// UnmarshalJSON implements custom validation logic
func (s *StackConfig) UnmarshalJSON(data []byte) error {
	type Alias StackConfig
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(s),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	if s.Type == "all" && s.Stack != nil {
		return errors.New("validation error: 'stack' must be undefined when type is 'all'")
	}

	if s.Type == "stack" && s.Stack == nil {
		return errors.New("validation error: 'stack' is required when type is 'stack'")
	}

	return nil
}

func postWebhook(ctx *gin.Context) {
	var req StackConfig
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	switch req.Type {
	case "all":
		util.Logger.Info("webhook: triggered update for all stacks")
		swarmcd.UpdateAllStacks()
		ctx.JSON(http.StatusOK, gin.H{"message": "all stacks update triggered"})
	case "stack":
		util.Logger.Info("webhook: triggered update for stack", "stack", *req.Stack)
		err := swarmcd.UpdateStack(*req.Stack)
		if err != nil {
			ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusOK, gin.H{"message": "stack update triggered", "stack": *req.Stack})
	default:
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid type: must be 'all' or 'stack'"})
	}
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
