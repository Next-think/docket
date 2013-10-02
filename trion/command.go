package trion

import (
	. "polydawn.net/gosh/psh"
	"polydawn.net/dockctrl/crocker"
	. "fmt"
	"os"
	"path/filepath"
)

//Default docker command template
var docker = crocker.NewDock("dock").Client()

//Where to place & call CIDfiles
const TempDir    = "/tmp"
const TempPrefix = "trion-"

//Executes 'docker run' and returns the container's CID.
func Run(config TrionConfig) string {
	dockRun := docker("run")

	//Find the absolute path for each host mount
	for i, j := range config.Mount {
		cwd, err := filepath.Abs(j[0])
		if err != nil {
			Println("Fatal: Cannot determine absolute path:", j[0])
			os.Exit(1)
		}

		config.Mount[i][0] = cwd
	}

	//Where should docker write the new CID?
	CIDfilename := createCIDfile()
	dockRun = dockRun("-cidfile", CIDfilename)

	//Where should the container start?
	dockRun = dockRun("-w", config.StartIn)

	//Is the docker in privleged (pwn ur box) mode?
	if (config.Privileged) {
		dockRun = dockRun("-privileged")
	}

	//Custom DNS servers?
	for i := range config.DNS {
		dockRun = dockRun ("-dns", config.DNS[i])
	}

	//What folders get mounted?
	for i := range config.Mount {
		dockRun = dockRun("-v", config.Mount[i][0] + ":" + config.Mount[i][1] + ":" + config.Mount[i][2])
	}

	//What environment variables are set?
	for i:= range config.Environment {
		dockRun = dockRun("-e", config.Environment[i][0] + "=" + config.Environment[i][1])
	}

	//Are we attaching?
	if config.Attach {
		dockRun = dockRun("-i", "-t")
	}

	//Add image name
	dockRun = dockRun(config.Image)

	//What command should it run?
	for i := range config.Command {
		dockRun = dockRun(config.Command[i])
	}

	//Poll for the CID and run the docker
	getCID := pollCid(CIDfilename)
	dockRun()
	return <- getCID
}

//Executes 'docker wait'
func Wait(CID string) {
	docker("wait", CID)()
}

//Executes 'docker rm'
func Purge(CID string) {
	docker("rm", CID)()
}

//Executes 'docker export', after ensuring there is no image.tar in the way.
//	This is because docker will *happily* export into an existing tar.
func Export(CID, path string) {
	//Check for existing file
	file, _ := os.Open("./image.tar")
	_, err  := file.Stat()
	file.Close()

	//Delete tar if it exists
	if err == nil {
		Println("Warning: output image.tar already exists. Overwriting...")
		err = os.Remove("./image.tar")
		if err != nil {
			Println("Fatal: Could not delete tar file.")
			os.Exit(1)
		}
	}

	out, err := os.OpenFile(path + "image.tar", os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		panic(err);
	}

	docker("export", CID)(Opts{Out: out})()
}
