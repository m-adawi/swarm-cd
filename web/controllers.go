package web

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/m-adawi/swarm-cd/swarmcd"
)

func getStacks(ctx *gin.Context) {
	ctx.IndentedJSON(http.StatusOK, swarmcd.GetStackStatus())
}