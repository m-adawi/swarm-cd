package main

import (
	"fmt"
	"os"

	"github.com/m-adawi/swarm-cd/swarmcd"
	"github.com/m-adawi/swarm-cd/util"
	"github.com/m-adawi/swarm-cd/web"
)

func init() {
	err := util.LoadConfigs()
	handleInitError(err)
	err = swarmcd.Init()
	handleInitError(err)
}

func main() {
	go swarmcd.Run()
	web.RunServer()
}

func handleInitError(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}