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
	err = util.InitVault()
	handleInitError(err)
	err = swarmcd.Init()
	handleInitError(err)
}

func main() {
	go swarmcd.Run()
	if err := web.RunServer(util.Configs.Address); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func handleInitError(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
