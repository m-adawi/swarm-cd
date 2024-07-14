package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	sloggin "github.com/samber/slog-gin"
)

var router *gin.Engine = gin.New()

func init(){
	router.Use(sloggin.New(logger))
	router.GET("/stacks", func(ctx *gin.Context) {ctx.IndentedJSON(http.StatusOK, stackStatus)})
}

func runWebServer() {
	err := router.Run("localhost:8080")
	handleInitError(err)
}