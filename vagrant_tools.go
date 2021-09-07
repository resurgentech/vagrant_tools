/*
  Copyright (c) 2021 Resurgent Technologies
*/

package main

import (
	"fmt"
	flag "github.com/spf13/pflag"
	"gopkg.in/yaml.v2"
	"log"

	//"github.com/jinzhu/copier"
	"github.com/koding/vagrantutil"
	"github.com/smallfish/simpleyaml"
	"regexp"

	//"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	//"regexp"
	"strings"
)


/*
  DeepCopyMap is a helper that lets us copy a map and makes it independent.
  You can't just copy a map, you really just point to it.  Which means edits to one 'copy' member edit all copies.
  This makes a recursive copy of each.
 */
func DeepCopyMap(m map[interface{}]interface{}) map[interface{}]interface{} {
	cp := make(map[interface{}]interface{})
	for k, v := range m {
		vm, ok := v.(map[interface{}]interface{})
		if ok {
			cp[k] = DeepCopyMap(vm)
		} else {
			cp[k] = v
		}
	}

	return cp
}


/*
  GetRootPath is a helper method that finds the present working dir and uses it for the root path for activity.
 */
func GetRootPath() string {
	pwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	path, err := filepath.Abs(pwd)
	if err != nil {
		panic(err)
	}
	return path
}


/*
  Machine implements a container for a vagrantutil.Vagrant object with context.
 */
type Machine struct {
	machine *vagrantutil.Vagrant
	name string
	path string
	vagrantfile string
}


/*
  NewMachine creates a new Machine and sets up the values for the context. Takes path as the directory vagrant runs in,
  vagrantfile contains the contents of the Vagrantfile for this Machine, provider is the vagra
 */
func NewMachine(path string, vagrantfile string, provider string) (*Machine, error) {
	machine, err := vagrantutil.NewVagrant(path)
	if err != nil {
		return nil, err
	}
	machine.ProviderName = provider
	return &Machine{
		machine: machine,
		name: filepath.Base(path),
		path: path,
		vagrantfile: vagrantfile,
	}, nil
}


/*
  MachineList is an array of Machines being managed
 */
type MachineList struct {
	options *Options
	machines []*Machine
	root string
}


/*
  NewMachineList initializes a MachineList structure
 */
func NewMachineList (options *Options) (*MachineList, error) {
	var ml MachineList
	ml.options = options
	pwd := GetRootPath()
	ml.root = path.Join(pwd,ml.options.machinesPath)
	inventory := path.Join(pwd,*ml.options.basedir,*ml.options.inventory)
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
	initalpaths, err := filepath.Glob(path.Join(ml.root,"*"))
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
  MakeVagrantfile creates a unified Vagrantfile by merging defaults with machine specifics.
 */
func (ml *MachineList) MakeVagrantfile(name string) (string, error) {
	var output string
	machinedef := ml.options.machineConfigurations[name].(map[interface{}]interface{})
	templatename := machinedef["vagrantfile_template"]
	if templatename == nil {
		templatename = "default"
	}
	template := ml.options.vagrantfileTemplates[templatename.(string)]
	vagrantfile := template

	// Override keys in Vagrantfile template
	vagrantKeyOverides, ok := machinedef["vagrant"]
	if ok {
		for k,v := range vagrantKeyOverides.(map[interface{}]interface{}) {
			output = strings.Replace(vagrantfile,"{"+k.(string)+"}",fmt.Sprintf("%v",v),-1)
			vagrantfile = output
		}
	}
	return output, nil
}


/*
  MakeMachines loops through config file data and initializes Machines.
 */
func (ml *MachineList) MakeMachines() error {
	for name := range ml.options.machineConfigurations {
		pathname := path.Join(ml.root,name)
		vagrantfile, err := ml.MakeVagrantfile(name)
		config := ml.options.config
		provider := config["machines"].(map[interface{}]interface{})["vagrant_provider"]
		m, err := NewMachine(pathname,vagrantfile, provider.(string))
		if err != nil {
			return err
		}
		ml.machines = append(ml.machines,m)
	}
	return nil
}


/*
  Status loops and prints out the status from the the Machines in the array
 */
func (ml *MachineList) Status() {
	for _, m := range ml.machines {
		status, err := m.machine.Status()
		if err != nil {
			panic(err)
		}
		fmt.Println(m.name)
		fmt.Println(status)
	}
}


/*
  HostEntry creates a structure for an ansible inventory file using data from the running VM's
 */
func (ml *MachineList) HostEntry(m *Machine, absolute_ssh_key_path bool) map[string]string {
	var hostEntry = make(map[string]string)

	// Start by adding key/values from "ansible"
	if ansibleTokens, ok := ml.options.machineConfigurations[m.name].(map[interface{}]interface{})["ansible"]; ok {
		for key, value := range ansibleTokens.(map[interface{}]interface{}) {
			hostEntry[key.(string)] = fmt.Sprintf("%v",value)
		}
	}

	// Take apart the sshconfig detail from vagrant and add
	output, err := m.machine.SSHConfig()
	if err != nil {
		panic(err)
	}
	fmt.Println(m.name)
	a := regexp.MustCompile("\n")
	pairs := a.Split(output,-1)
	for _, pair := range pairs {
		tokens := strings.Fields(pair)
		fmt.Println(tokens)
		if len(tokens) < 2 {
			continue
		}
		if tokens[0] == "HostName" {
			hostEntry["ansible_host"] = tokens[1]
			hostEntry["ansible_ip"] = tokens[1]
			hostEntry["ip"] = tokens[1]
		} else if tokens[0] == "User" {
			hostEntry["ansible_user"] = tokens[1]
		} else if tokens[0] == "Port" {
			hostEntry["ansible_port"] = tokens[1]
		} else if tokens[0] == "IdentityFile" {
			var key = ""
			if (absolute_ssh_key_path) {
				key = tokens[1]
			} else {
				basedir := filepath.Dir(ml.root)
				key = "." + strings.TrimPrefix(tokens[1],basedir)
			}
			hostEntry["ansible_ssh_private_key_file"] = key
		}
	}
	return hostEntry
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
		status, err := m.machine.Status()
		if err != nil {
			panic(err)
		}
		if fmt.Sprintf("%v",status) != "Running" {
			continue
		}
		hostEntry := ml.HostEntry(m,absolute_ssh_key_path)
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

	boutputfile := []byte(strings.ReplaceAll(string(outputfile), "null",""))
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
  Destroy loops through and tears down the Machines stops and deletes them.
 */
func (ml *MachineList) Destroy() {
	for _, m := range ml.machines {
		fmt.Println(m.name)
		output, err := m.machine.Destroy()
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
		status, err := m.machine.Status()
		if err != nil {
			panic(err)
		}
		if fmt.Sprintf("%v",status) == "NotCreated" {
			err := m.machine.Create(m.vagrantfile)
			if err != nil {
				panic(err)
			}
		}
		output, err := m.machine.Up()
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
		m, err := NewMachine(pathname,"","")
		if err != nil {
			panic(err)
		}
		fmt.Println(m)
		ml.machines = append(ml.machines,m)
	}
	ml.Destroy()
	for _, pathname := range paths {
		err := os.RemoveAll(pathname)
		if err != nil {
			fmt.Printf("Failed to delete %v\n\n",err)
		}
	}
}


/*
  Options holds the configuration of this run.
 */
type Options struct {
	configfile            *string                     //From CLI - holds path to the yaml configuration file
	config                map[interface{}]interface{} // structure from yaml configfile
	action                *string                     // From CLI - action to execute
	inventory             *string                     // From CLI - path to inventory file to write
	basedir               *string                     // From CLI - directory as base path to machines and inventory
	absolute_ssh_key_path *bool                       // From CLI - path to inventory file to write
	esxi_hostname         *string                     // esxi hostname to connect to
	esxi_username         *string                     // esxi username to login into esxi_hostname
	esxi_password         *string                     // esxi password to login into esxi_hostname
	machinesPath          string                      // directory where vagrant machine directories are stored
	machineTemplates      interface{}                 //machine_templates templates
	vagrantfileTemplates  map[string]string           //Vagrantfile templates
	machineConfigurations map[string]interface{}      //stores machine configurations from config
	childrenTemplate      map[interface{}]interface{} //holds children section
}


/*
  MergeEsxiCredentials merge in esxi parameters
 */
func (o *Options) MergeEsxiCredentials(machineDefinition interface{}) interface{} {
//	var vagrant map[interface{}]interface{}
	if vagrant, ok := machineDefinition.(map[interface{}]interface{})["vagrant"].(map[interface{}]interface{}); !ok {
		// not going to splice this into an empty vagrant: section
		return machineDefinition
	} else {
		if *o.esxi_hostname != "" {
			vagrant["esxi_hostname"] = *o.esxi_hostname
		}
		if *o.esxi_username != "" {
			vagrant["esxi_username"] = *o.esxi_username
		}
		if *o.esxi_password != "" {
			vagrant["esxi_password"] = *o.esxi_password
		}
	}

	return machineDefinition
}

/*
  SetMachineTemplate merge different sources to a single machine definition.
 */
func (o *Options) SetMachineTemplate(machineDefinition interface{}) interface{} {
	var machine interface{}
	var machineTemplate string

	// Did the config specify a template, or do we use the default?
	if machineDefinition == nil { // if the machine is empty we assume "default"
		machineTemplate = "default"
	} else {
		if strmachineTemplate, ok := machineDefinition.(map[interface{}]interface{})["machine_template"]; !ok {
			// if machine not nil but machine_template isn't specified, we do this to have it skip
			machineTemplate = ""
		} else {
			machineTemplate = strmachineTemplate.(string)
		}
	}

	// Copy template to make a new machine definition
	machineDefault, ok := o.machineTemplates.(map[interface{}]interface{})[machineTemplate]
	if ok {
		machine = DeepCopyMap(machineDefault.(map[interface{}]interface{}))
	}

	// If we have nothing to add beyond the template, this is our stop.
	if machineDefinition == nil {
		return machine
	}

	// If we don't have a machine started by now lets start with an empty definition
	if machine == nil {
		machine = make(map[interface{}]interface{})
	}

	// Walk the machineDefinition to merge with any defaults.
	// machine2, machine3 are pointer that enable us to iterate nicely
	machine2 := machine.(map[interface{}]interface{})
	for key1, value1 := range machineDefinition.(map[interface{}]interface{}) {
		switch vt := value1.(type) {
		case map[interface{}]interface{}:
			if _, ok := machine2[key1]; !ok {
				machine2[key1] = make(map[interface{}]interface{})
			}
			for key2, value2 := range value1.(map[interface{}]interface{}) {
				machine3 := machine2[key1].(map[interface{}]interface{})
				machine3[key2] = value2
			}
		case interface{}:
			machine2[key1] = value1
		default:
			fmt.Printf("I don't know about type %T!\n", vt)
		}
	}

	// Merge ESXi credentials
	machine = o.MergeEsxiCredentials(machine)
	return machine
}


/*
  ReadConfig configures machines as determined by a yaml file
*/
func (o *Options) ReadConfig() {
	// Read configfile into memory
	data, err := ioutil.ReadFile(*o.configfile)
	if err != nil {
		*o.configfile = path.Join(GetRootPath(), *o.configfile)
	}
	data, err = ioutil.ReadFile(*o.configfile)
	if err != nil {
		panic(err)
	}
	config, err := simpleyaml.NewYaml(data)
	if err != nil {
		panic(err)
	}
	o.config, err = config.Map()
	if err != nil {
		panic(err)
	}

	// Set up pathname where vagrant machines are to be located
	pathname, err := config.GetPath("machines","path").String()
	if err != nil {
		panic(err)
	}
	o.machinesPath = path.Join(*o.basedir,pathname)

	// Read in vagrant file templates
	vagrantfileTemplates, err := config.GetPath("vagrantfile_templates").Map()
	if err != nil {
		panic(err)
	}
	o.vagrantfileTemplates = make(map[string]string)
	for name, vfTemplate := range vagrantfileTemplates {
		o.vagrantfileTemplates[name.(string)] = vfTemplate.(string)
	}
	if _, ok := o.vagrantfileTemplates["default"]; !ok {
		panic("default vagrantfile_template required")
	}

	// Read in machine templates
	machineTemplates, err := config.GetPath("machine_templates").Map()
	if err == nil {
		o.machineTemplates = machineTemplates
	}

	// Read in children section for ansible inventory
	childrenTemplate, err := config.GetPath("children").Map()
	if err != nil {
		fmt.Println("file needs children section")
		panic(err)
	} else {
		o.childrenTemplate = childrenTemplate
	}

	// Read in list of machine definitions
	list, err := config.GetPath("machines","list").Map()
	if err != nil {
		panic(err)
	}

	// reconcile templates into machines
	o.machineConfigurations = make(map[string]interface{})
	for name, inMachine := range list {
		fmt.Print("----", name, "---\n\n", inMachine, "\n\n\n")
		o.machineConfigurations[name.(string)] = o.SetMachineTemplate(inMachine)
	}
	//set defaults if not overwritten in config (conventions over configuration)
	for name, inMachine := range o.machineConfigurations {
		machine := inMachine.(map[interface{}]interface{})
		fmt.Println(machine)
		fmt.Println(machine["vagrant"])
		vM, ok := machine["vagrant"]
		if !ok {continue}
		vmachine := vM.(map[interface{}]interface{})
		if _, ok := vmachine["node"]; !ok {
			vmachine["node"] = name
		}
		if _, ok := vmachine["hostname"]; !ok {
			vmachine["hostname"] = name
		}
		if _, ok := machine["vagrantfile_template"]; !ok {
			machine["vagrantfile_template"] = "default"
		}
	}
	fmt.Println(o)
}


/*
  ParseCli Handle commandline arguments
*/
func (o *Options) ParseCli() {
	o.configfile = flag.StringP("configfile","c", path.Join("configs","sample.yaml"), "Config file")
	o.action = flag.StringP("action","a", "", "up - Builds VM's\ndown - Destroy VM's\nkillthemall - Clean up any old stuff.")
	o.inventory = flag.StringP("inventory","i","inventory.yaml", "Output inventory file for ansible.")
	o.basedir = flag.StringP("basedir", "b", "", "Directory to store assets in.")
	o.absolute_ssh_key_path = flag.Bool("absolute_ssh_key_path", false, "ssh keys stored as an absolute path instead of relative")
	o.esxi_hostname = flag.String("esxi_hostname", "", "[Optional] esxi hostname to pass into Vagrantfile when using vmware_esxi provider.")
	o.esxi_username = flag.String("esxi_username", "", "[Optional] esxi username to pass into Vagrantfile when using vmware_esxi provider.")
	o.esxi_password = flag.String("esxi_password", "", "[Optional] esxi password to pass into Vagrantfile when using vmware_esxi provider.")
	flag.Parse()
}


/*
  Putting it all together
*/
func main() {
	var options Options
	options.ParseCli()
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
