/*
  Copyright (c) 2021 Resurgent Technologies
*/

package main

import (
	"fmt"
	flag "github.com/spf13/pflag"
)

/*
  ParseCli Handle commandline arguments
*/
func (o *Options) ParseCli() {
	o.configfile = flag.StringP("configfile", "c", "", "Config file")
	o.action = flag.StringP("action", "a", "", "up - Builds VM's\ndown - Destroy VM's\nkillthemall - Clean up any old stuff.")
	o.inventory = flag.StringP("inventory", "i", "inventory.yaml", "Output inventory file for ansible.")
	o.basedir = flag.StringP("basedir", "b", "", "Directory to store assets in.")
	o.absolute_ssh_key_path = flag.Bool("absolute_ssh_key_path", false, "ssh keys stored as an absolute path instead of relative")
	o.esxi_hostname = flag.String("esxi_hostname", "", "[Optional] esxi hostname to pass into Vagrantfile when using vmware_esxi provider.")
	o.esxi_username = flag.String("esxi_username", "", "[Optional] esxi username to pass into Vagrantfile when using vmware_esxi provider.")
	o.esxi_password = flag.String("esxi_password", "", "[Optional] esxi password to pass into Vagrantfile when using vmware_esxi provider.")
	o.version = flag.Bool("version", false, "Print version number.")
	flag.Parse()
}

var GitCommit string

/*
  Putting it all together
*/
func main() {
	var options Options
	options.ParseCli()

	if *options.version {
		fmt.Printf("vagrant_tools, git commit: %s\n", GitCommit)
		return
	}

	fmt.Println(*options.configfile)
	options.ReadConfig()

	ml, err := NewMachineList(&options)
	if err != nil {
		panic(err)
	}

	if *options.action == "killthemall" {
		ml.KillThemAll()
		return
	}
	if *options.action == "download_boxes" {
		ml.DownloadBoxes()
		return
	}

	if *options.action == "up" {
		fmt.Println("----Up----")
		ml.Up()
		fmt.Println("----Status----")
		ml.Status()
	}
	if *options.action == "down" {
		fmt.Println("----Destroy----")
		ml.Destroy()
		fmt.Println("----Status----")
		ml.Status()
	}
	if *options.inventory != "" {
		fmt.Println("----inventory----")
		ml.Inventory(*options.absolute_ssh_key_path)
		fmt.Println("----Status----")
		ml.Status()
	}
}
