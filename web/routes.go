package web

import (
	"github.com/gin-gonic/gin"
	"github.com/m-adawi/swarm-cd/util"
	sloggin "github.com/samber/slog-gin"
)

var router *gin.Engine = gin.New()

func init() {
	router.Use(sloggin.New(util.Logger))
	router.GET("/stacks", getStacks)
	router.StaticFile("/ui", "ui/dist/index.html")
	router.Static("/assets", "ui/dist/assets")
	router.GET("/", func(c *gin.Context) {
		c.Redirect(302, "/ui")
	})
}

func RunServer() {
	router.Run("localhost:8080")
}
