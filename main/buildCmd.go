package main

import (
	. "fmt"
	"io"
	"polydawn.net/docket/confl"
	"polydawn.net/docket/crocker"
	"polydawn.net/docket/dex"
)

type buildCmdOpts struct {
	Source      string `short:"s" long:"source"      default:"graph" description:"Container source."`
	Destination string `short:"d" long:"destination" default:"graph" description:"Container destination."`
}

const DefaultBuildTarget = "build"

//Transforms a container
func (opts *buildCmdOpts) Execute(args []string) error {
	//Get configuration
	target   := GetTarget(args, DefaultBuildTarget)
	settings := confl.NewConfigLoad(".")
	config := settings.GetConfig(target)
	saveAs := settings.GetDefaultImage()
	var sourceGraph, destinationGraph *dex.Graph

	//Right now, go-flags' default announation does not appear to work when in a sub-command.
	//	Will investigate and hopefully remove this later.
	if opts.Source == "" {
		opts.Source = "graph"
	}
	if opts.Destination == "" {
		opts.Destination = "graph"
	}

	//Parse input/output URIs
	sourceScheme, sourcePath           := ParseURI(opts.Source)
	destinationScheme, destinationPath := ParseURI(opts.Destination)

	_, _ = sourcePath, destinationPath //remove later

	//Prepare input
	switch sourceScheme {
		case "docker":
			//TODO: check that docker has the image loaded
		case "graph":
			//Look up the graph, and clear any unwanted state
			sourceGraph = dex.NewGraph(settings.Graph)
			sourceGraph.Cleanse()
		case "file", "index":
			return Errorf("Source " + sourceScheme + " is not supported yet.")
	}

	//Prepare output
	switch destinationScheme {
		case "docker":
			//Nothing required here until container has ran
		case "graph":
			//Look up the graph, and clear any unwanted state
			destinationGraph = dex.NewGraph(settings.Graph)

			//Don't run extra git commands if they'd be redundant.
			//Right now, we're ignoring the destinationPath, so this will never fire.
			if sourceScheme == "graph" && sourceGraph.GetDir() != destinationGraph.GetDir() {
				destinationGraph.Cleanse()
			}
		case "file", "index":
			return Errorf("Destination " + destinationScheme + " is not supported yet.")
	}

	//Start or connect to a docker daemon
	dock := StartDocker(settings)

	// Import the latest lineage
	if sourceScheme == "graph" {
		dock.Import(sourceGraph.Load(config.Image), config.Image, "latest")
	}

	// Launch the container and wait for it to finish
	container := Launch(dock, config)
	container.Wait()

	//Perform any destination operations required
	name, tag := crocker.SplitImageName(saveAs)
	switch destinationScheme {
		//If we're not exporting to the graph, there is no commit hash from which to generate a tag.
		//	Thus the docker import will have either a static tag (from docker.toml) or the default 'latest' tag.
		case "docker":
			Println("Exporting to", name, tag)
			container.Commit(name, tag)
		case "graph":
			// Export a tar of the filesystem
			exportStreamOut, exportStreamIn := io.Pipe()
			go container.Export(exportStreamIn)

			// Commit it to the image graph
			destinationGraph.Publish(exportStreamOut, saveAs, config.Image)
	}

	//Remove if desired
	if config.Purge {
		container.Purge()
	}

	//Stop the docker daemon if it's a child process
	dock.Slay()

	return nil
}
