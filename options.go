package main

import (
	"fmt"
	"github.com/smallfish/simpleyaml"
	"io/ioutil"
	"path"
)

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
	version               *bool                       // print version number
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
	pathname, err := config.GetPath("machines", "path").String()
	if err != nil {
		panic(err)
	}
	o.machinesPath = path.Join(*o.basedir, pathname)

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
	list, err := config.GetPath("machines", "list").Map()
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
		if !ok {
			continue
		}
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
