package main

import (
	"fmt"
	yaml "gopkg.in/yaml.v2"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
)

/*
  MachineList is an array of Machines being managed
*/
type MachineList struct {
	options  *Options
	machines []*Machine
	root     string
}

/*
  NewMachineList initializes a MachineList structure
*/
func NewMachineList(options *Options) (*MachineList, error) {
	var ml MachineList
	ml.options = options
	pwd := GetRootPath()
	ml.root = path.Join(pwd, ml.options.machinesPath)
	inventory := path.Join(pwd, *ml.options.basedir, *ml.options.inventory)
	absinventory, err := filepath.Abs(inventory)
	if err != nil {
		panic(err)
	}
	ml.options.inventory = &absinventory
	if *ml.options.action == "killthemall" {
		return &ml, nil
	}
	err = ml.MakeMachines()
	if err != nil {
		return nil, err
	}
	return &ml, nil
}

/*
  GetExistingMachineFolders will scan a directory for existing Machines by looking for subdirectories with
  vagrant artifacts, Vagrantfile, etc.
*/
func (ml *MachineList) GetExistingMachineFolders() ([]string, error) {
	var paths []string
	initalpaths, err := filepath.Glob(path.Join(ml.root, "*"))
	if err != nil {
		return nil, err
	}
	for _, pathname := range initalpaths {
		info, err := os.Lstat(pathname)
		if err != nil {
			return nil, err
		}
		if info.IsDir() {
			paths = append(paths, pathname)
		}
	}
	return paths, nil
}

/*
  MakeMachines loops through config file data and initializes Machines.
*/
func (ml *MachineList) MakeMachines() error {
	for name := range ml.options.machineConfigurations {
		pathname := path.Join(ml.root, name)
		m, err := NewMachine(pathname, ml.options)
		if err != nil {
			return err
		}
		ml.machines = append(ml.machines, m)
	}
	return nil
}

/*
  Status loops and prints out the status from the the Machines in the array
*/
func (ml *MachineList) Status() {
	for _, m := range ml.machines {
		status, err := m.Status()
		if err != nil {
			panic(err)
		}
		if status == 2 {
			fmt.Printf("%s = Running\n", m.name)
		} else {
			fmt.Printf("%s: status: %v\n", m.name, status)
		}

	}
}

/*
  Inventory will generate a yaml file for ansible from the list of managed machines
*/
func (ml *MachineList) Inventory(absolute_ssh_key_path bool) {
	// create empty inventory structure
	var inventory = make(map[string]interface{})
	inventory["all"] = make(map[string]interface{})
	allInventory := inventory["all"].(map[string]interface{})
	allInventory["hosts"] = make(map[string]interface{})
	allInventory["children"] = make(map[interface{}]interface{})
	hostsInventory := allInventory["hosts"].(map[string]interface{})

	// generate entries for each running host in inventory structure
	for _, m := range ml.machines {
		status, err := m.Status()
		if err != nil {
			panic(err)
		}
		// Running == 2 in latest vagrantutils
		if status != 2 {
			continue
		}
		hostEntry := m.HostEntry(absolute_ssh_key_path, ml.options, ml.root)
		if _, ok := hostsInventory[m.name]; !ok {
			hostsInventory[m.name] = make(map[interface{}]interface{})
		}
		hostsInventory[m.name] = hostEntry
	}

	// Dump structure to yaml file
	allInventory["children"] = ml.options.childrenTemplate
	outputfile, err := yaml.Marshal(&inventory)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(outputfile))
	f, err := os.Create(*ml.options.inventory)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	boutputfile := []byte(strings.ReplaceAll(string(outputfile), "null", ""))
	_, err = f.Write(boutputfile)
	if err != nil {
		panic(err)
	}
	err = f.Sync()
	if err != nil {
		panic(err)
	}
}

/*
  Downloads all the boxes
*/
func (ml *MachineList) DownloadBoxes() {
	provider := ml.options.config["machines"].(map[interface{}]interface{})["vagrant_provider"].(string)
	blist := MakeBoxList(ml.options)
	for name, boxname := range blist {
		newpath := filepath.Join(ml.root, "testbox")
		os.MkdirAll(newpath, os.ModePerm)
		fmt.Println("--------add box------------------------------------------------")
		var args = []string{"box", "add", "--no-tty", "--provider", provider, boxname}
		s, _ := RunCommand("vagrant", args, newpath)
		fmt.Println(s)
		fmt.Println(name)
		fmt.Println("---------init box-----------------------------------------------")
		args = []string{"init", boxname}
		s, _ = RunCommand("vagrant", args, newpath)
		fmt.Println(s)
		fmt.Println("----------box up----------------------------------------------")
		args = []string{"up"}
		s, _ = RunCommand("vagrant", args, newpath)
		fmt.Println(s)
		fmt.Println("----------box destroy----------------------------------------------")
		args = []string{"destroy", "--force"}
		s, _ = RunCommand("vagrant", args, newpath)
		fmt.Println(s)
		fmt.Println(newpath)
		err := os.RemoveAll(newpath)
		if err != nil {
			log.Fatal(err)
		}
	}
}

/*
   Destroy loops through and tears down the Machines stops and deletes them.
*/
func (ml *MachineList) Destroy() {
	for _, m := range ml.machines {
		fmt.Println(m.name)
		output, err := m.Destroy()
		if err != nil {
			panic(err)
		}
		for line := range output {
			log.Println(line)
		}
	}
}

/*
   Up creates and runs the Machines
*/
func (ml *MachineList) Up() {
	for _, m := range ml.machines {
		fmt.Println(m.name)
		status, err := m.Status()
		if err != nil {
			panic(err)
		}
		// NotCreated == 1 in latest vagrantutils
		if status == 1 {
			err := m.Create(m.vagrantfile)
			if err != nil {
				panic(err)
			}
		}
		output, err := m.Up()
		if err != nil {
			panic(err)
		}
		for line := range output {
			log.Println(line)
		}
	}
}

/*
   KillThemAll is a more aggressive destroy/clean
*/
func (ml *MachineList) KillThemAll() {
	paths, err := ml.GetExistingMachineFolders()
	if err != nil {
		panic(err)
	}
	for _, pathname := range paths {
		m, err := NewMachine(pathname, ml.options)
		if err != nil {
			panic(err)
		}
		fmt.Println(m)
		ml.machines = append(ml.machines, m)
	}
	ml.Destroy()
	for _, pathname := range paths {
		err := os.RemoveAll(pathname)
		if err != nil {
			fmt.Printf("Failed to delete %v\n\n", err)
		}
	}
}
