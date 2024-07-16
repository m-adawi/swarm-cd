package web

import (

	"github.com/gin-gonic/gin"
	"github.com/m-adawi/swarm-cd/util"
	sloggin "github.com/samber/slog-gin"
)

var router *gin.Engine = gin.New()

func init(){
	router.Use(sloggin.New(util.Logger))
	router.GET("/stacks", getStacks)
}

func RunServer() {
	router.Run("localhost:8080")
}
