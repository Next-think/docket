//High-level functionality

package trion

import (
	"polydawn.net/dockctrl/crocker"
	. "fmt"
)

const ExportPath = "./" //Where to export docker images

//Helps run anyything that requires a docker connection.
//Handles creation & cleanup in one place.
func WithDocker(fn func(TrionConfig, *crocker.Dock) error ) error {
	//Load configuration, then find or start a docker
	config := FindConfig(".")
	dock, dir, ours := crocker.FindDock()

	//Announce the docker
	if ours {
		Println("Started a docker in", dir)
	} else {
		Println("Connecting to docker", dir)
	}

	//Run the closure, kill the docker if needed, and return any errors.
	err := fn(config, dock)
	dock.Slay()
	return err
}

//Helper function: maps a TrionConfig struct to crocker function.
//Kinda ugly; this situation may improve once our config shenanigans solidifies a bit.
func Run(dock *crocker.Dock, config TrionConfig) *crocker.Container {

	return crocker.Launch(dock, config.Image, config.Command, config.Attach, config.Privileged, config.StartIn, config.DNS, config.Mounts, config.Ports, config.Environment)
}


//Launches a docker
func Launch(config TrionConfig, dock *crocker.Dock) error {
	//Start the docker and wait for it to finish
	container := Run(dock, config)
	container.Wait()

	//Remove if desired
	if config.Purge {
		container.Purge()
	}

	return nil
}

//Builds a docker
func Build(config TrionConfig, dock *crocker.Dock) error {
	//Use the build command and upstream image
	buildConfig        := config
	buildConfig.Command = config.Build
	buildConfig.Image   = config.Upstream

	//Run the build
	container := Run(dock, buildConfig)
	container.Wait()

	//Create a tar
	container.Export(ExportPath)

	//Import the built docker
	// Todo: add --noImport option to goflags
	container.ImportFromString(ExportPath, config.Image)

	//Remove if desired
	if config.Purge {
		container.Purge()
	}

	return nil
}
