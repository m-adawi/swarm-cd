package main

import (
	"log"
	"flag"
	"fmt"
	"os"

	"github.com/m-adawi/swarm-cd/swarmcd"
	"github.com/m-adawi/swarm-cd/util"
)

func init() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [OPTIONS]... INPUTFILE [OUTPUTFILE]\n", os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(), "If OUTPUTFILE is not provided or is equal to \"-\", result will be outputted to stdout\n\n")
		flag.PrintDefaults()
	}
}

func main() {
	var valueFile, globalPath, configPath, templateFolder string
	flag.StringVar(&valueFile, "valuefile", "", "Path to a value file")
	flag.StringVar(&globalPath, "global", "", "Path to a global value file")
	flag.StringVar(&configPath, "config", "", "Path to a config file (for globals)")
	flag.StringVar(&templateFolder, "templatefolder", "", "Path to the template folder")

	flag.Parse()

	if len(flag.Args()) == 0 {
		log.Fatal("INPUTFILE is required")
	}
	composeFile := flag.Args()[0]
	var err error
	var globalValuesMap map[string]any
	if configPath != "" {
		err = util.ReadConfig(configPath)
		if err != nil {
			log.Fatal("Could not parse config file: ", err)
		}
		if util.Configs.GlobalValues != nil && util.Configs.GlobalValuesPath != "" {
			log.Print("Both global_values and global_values_path provided, ignoring global_values_path")
			util.Configs.GlobalValuesPath = ""
		} else if util.Configs.GlobalValuesPath != "" {
			log.Print("Using global_values_path from config")
			err = util.ReadGlobalValues(util.Configs.GlobalValuesPath)
			if err != nil {
				log.Fatal("Could not parse global file: ", err)
			}
		}
		globalValuesMap = util.Configs.GlobalValues
	} else if globalPath != "" {
		err = util.ReadGlobalValues(globalPath)
		if err != nil {
			log.Fatal("Could not parse global file: ", err)
		}
		globalValuesMap = util.Configs.GlobalValues
	}

	outputFile := "-"
	if len(flag.Args()) > 1 {
		outputFile = flag.Args()[1]
	}

	stack := swarmcd.NewSwarmStack(
		"Template test",
		nil,
		"nobranch",
		composeFile,
		nil,
		valueFile,
		false,
		globalValuesMap,
		templateFolder,
	)

	stackBytes, err := stack.GenerateStack()
	if err != nil {
		log.Fatal("Could not generate stack: ", err)
	}
	if outputFile == "-" {
		fmt.Println(string(stackBytes))

	} else {
		err = os.WriteFile(outputFile, stackBytes, 0600)
		if err != nil {
			log.Fatal("Could not write file ", outputFile, ": ", err)
		}
	}
}

