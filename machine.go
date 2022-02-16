package main

import (
	"fmt"
	"github.com/koding/vagrantutil"
	"path/filepath"
	"regexp"
	"strings"
)

/*
  Machine implements a container for a vagrantutil.Vagrant object with context.
*/
type Machine struct {
	machine     *vagrantutil.Vagrant
	name        string
	path        string
	vagrantfile string
}

/*
   NewMachine creates a new Machine and sets up the values for the context. Takes path as the directory vagrant runs in,
   vagrantfile contains the contents of the Vagrantfile for this Machine, provider is the vagra
*/
func NewMachine(path string, options *Options) (*Machine, error) {
	machine, err := vagrantutil.NewVagrant(path)
	if err != nil {
		return nil, err
	}
	config := options.config
	provider := config["machines"].(map[interface{}]interface{})["vagrant_provider"]
	machine.ProviderName = provider.(string)
	name := filepath.Base(path)
	vagrantfile, err2 := MakeVagrantfile(name, options)
	if err2 != nil {
		return nil, err2
	}
	return &Machine{
		machine:     machine,
		name:        name,
		path:        path,
		vagrantfile: vagrantfile,
	}, nil
}

/*
  HostEntry creates a structure for an ansible inventory file using data from the running VM's
*/
func (m *Machine) HostEntry(absolute_ssh_key_path bool, options *Options, root string) map[string]string {
	var hostEntry = make(map[string]string)

	// Start by adding key/values from "ansible"
	if ansibleTokens, ok := options.machineConfigurations[m.name].(map[interface{}]interface{})["ansible"]; ok {
		for key, value := range ansibleTokens.(map[interface{}]interface{}) {
			hostEntry[key.(string)] = fmt.Sprintf("%v", value)
		}
	}

	// Take apart the sshconfig detail from vagrant and add
	output, err := m.SSHConfig()
	if err != nil {
		panic(err)
	}
	fmt.Println(m.name)
	a := regexp.MustCompile("\n")
	pairs := a.Split(output, -1)
	for _, pair := range pairs {
		tokens := strings.Fields(pair)
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
			if absolute_ssh_key_path {
				key = tokens[1]
			} else {
				basedir := filepath.Dir(root)
				key = "." + strings.TrimPrefix(tokens[1], basedir)
			}
			hostEntry["ansible_ssh_private_key_file"] = key
		}
	}
	return hostEntry
}

func (m *Machine) Create(vagrantFile string) error {
	return m.machine.Create(vagrantFile)
}

func (m *Machine) Status() (int, error) {
	st, err := m.machine.Status()
	return int(st), err
}

func (m *Machine) SSHConfig() (string, error) {
	return m.machine.SSHConfig()
}

func (m *Machine) Destroy() (<-chan *vagrantutil.CommandOutput, error) {
	return m.machine.Destroy()
}

func (m *Machine) Up() (<-chan *vagrantutil.CommandOutput, error) {
	return m.machine.Up()
}
