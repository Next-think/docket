package commands

import (
	. "fmt"
)

const Version = "0.2.0"

type VersionCmdOpts struct { }

//Version command
func (opts *VersionCmdOpts) Execute(args []string) error {
	Println("docket version", Version)
	return nil
}