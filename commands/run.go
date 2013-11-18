package commands

import (
	"polydawn.net/docket/confl"
	"polydawn.net/docket/dex"
	. "polydawn.net/docket/util"
)

type RunCmdOpts struct {
	Source      string `short:"s" long:"source" default:"graph" description:"Container source."`
}

const DefaultRunTarget = "default"

//Runs a container
func (opts *RunCmdOpts) Execute(args []string) error {
	//Get configuration
	target   := GetTarget(args, DefaultRunTarget)
	settings := confl.NewConfigLoad(".")
	config   := settings.GetConfig(target)
	var sourceGraph *dex.Graph

	//Parse input URI
	sourceScheme, sourcePath := ParseURI(opts.Source)
	_ = sourcePath //remove later

	//Prepare input
	switch sourceScheme {
		case "docker":
			//TODO: check that docker has the image loaded
		case "graph":
			//Look up the graph, and clear any unwanted state
			sourceGraph = dex.NewGraph(settings.Graph)
			sourceGraph.Cleanse()
		case "file":
			//If the user did not specify an image path, set one
			if sourcePath == "" {
				sourcePath = "./image.tar"
			}
	}

	//Start or connect to a docker daemon
	dock := StartDocker(settings)

	//Prepare cache
	switch sourceScheme {
		case "graph":
			//Import the latest lineage
			dock.Import(sourceGraph.Load(config.Image), config.Image, "latest")
		case "file":
			//Load image from file
			dock.ImportFromFilenameTagstring(sourcePath, config.Image)
		case "index":
			//TODO: check that docker doesn't already have the image loaded
			dock.Pull(config.Image)
	}

	//Run the container and wait for it to finish
	container := Launch(dock, config)
	container.Wait()

	//Remove if desired
	if config.Purge {
		container.Purge()
	}

	//Stop the docker daemon if it's a child process
	dock.Slay()

	return nil
}
