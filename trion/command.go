package trion

import (
	. "polydawn.net/gosh/psh"
	"polydawn.net/dockctrl/crocker"
	. "fmt"
	"os"
	"path/filepath"
	"strings"
)

//Default docker command template
var docker = crocker.NewDock("dock").Client()

//Where to place & call CIDfiles
const TempDir    = "/tmp"
const TempPrefix = "trion-"

//Default tar filename amd image tag
const TarFile    = "image.tar"
const DefaultTag = "latest"

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
	dockRun()
	getCID := pollCid(CIDfilename)
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
	tar := path + TarFile

	//Check for existing file
	file, _ := os.Open(tar)
	_, err  := file.Stat()
	file.Close()

	//Delete tar if it exists
	if err == nil {
		Println("Warning: Output image.tar already exists. Overwriting...")
		err = os.Remove("./image.tar")
		if err != nil {
			Println("Fatal: Could not delete tar file.")
			os.Exit(1)
		}
	}

	out, err := os.OpenFile(tar, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		panic(err);
	}

	docker("export", CID)(Opts{Out: out})()
}

//Import an image into docker's repository.
func Import(config TrionConfig, path string) {
	tar := path + TarFile

	//Open the file
	in, err := os.Open(tar)
	if err != nil {
		Println("Fatal: Could not open file for import:", tar)
	}

	//Get the repository and tag from the config
	repo, tag := "", ""
	sp := strings.Split(config.Image, ":")

	//If both a name and version are specified, use them, otherwise just tag it as 'latest'
	if len(sp) == 2 {
		repo = sp[0]
		tag = sp[1]
	} else {
		repo = config.Image
		tag = DefaultTag
	}

	Println("Importing", repo + ":" + tag)
	docker("import", "-", repo, tag)(Opts{In: in, Out: os.Stdout })()
}
