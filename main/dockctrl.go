package main

import (
	"github.com/jessevdk/go-flags"
	"polydawn.net/dockctrl/trion"
	"os"
)

var parser = flags.NewNamedParser("dockctrl", flags.Default)

var EXIT_BADARGS = 1
var EXIT_PANIC = 2

func main() {
	_, err := parser.Parse()
	if err != nil {
		os.Exit(EXIT_BADARGS)
	}
	os.Exit(0)
}


func init() {
	// parser.AddCommand(
	// 	"command",
	// 	"description",
	// 	"long description",
	// 	&whateverCmd{}
	// )
	parser.AddCommand(
		"launch",
		"launch a container",
		"launch a container based on configuration in the current directory.",
		&launchCmd{},
	)
	parser.AddCommand(
		"build",
		"build an image and export tar",
		"launch a container, and after the container has completed, export a tar of the filesystem.",
		&buildCmd{},
	)
	parser.AddCommand(
		"publish",
		"build and publish a versioned-controlled image",
		"build a container, and place the exported tar into a git repository.",
		&struct{}{},
	)
	parser.AddCommand(
		"unpack",
		"unpack a base image from versioned-controlled storage",
		"unpack a base image from versioned-controlled storage so that it's ready to be used to launch a container.",
		&struct{}{},
	)
}

type launchCmd struct {}
func (opts *launchCmd) Execute(args []string) error {
	config := trion.FindConfig(".")

	CID := trion.Run(config)
	trion.Wait(CID)

	if config.Purge {
		trion.Purge(CID)
	}

	return nil
}

type buildCmd struct {}
func (opts *buildCmd) Execute(args []string) error {
	config := trion.FindConfig(".")
	path := "./"

	//Use the build command and upstream image
	buildConfig        := config
	buildConfig.Command = config.Build
	buildConfig.Image   = config.Upstream

	//Run the build
	CID := trion.Run(buildConfig)
	trion.Wait(CID)

	//Create a tar
	trion.Export(CID, path)

	//Import the built docker
	// Todo: add --noImport option to goflags
	trion.Import(config, path)

	if config.Purge {
		trion.Purge(CID)
	}

	return nil
}