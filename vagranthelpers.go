package main

import (
	"fmt"
	"strings"
)

/*
  MakeVagrantfile creates a unified Vagrantfile by merging defaults with machine specifics.
*/
func MakeVagrantfile(name string, options *Options) (string, error) {
	var output string
	var machinedef map[interface{}]interface{}

	if _, ok := options.machineConfigurations[name]; ok {
		machinedef = options.machineConfigurations[name].(map[interface{}]interface{})
	} else {
		return "", nil
	}

	templatename := machinedef["vagrantfile_template"]
	if templatename == nil {
		templatename = "default"
	}
	template := options.vagrantfileTemplates[templatename.(string)]
	vagrantfile := template

	// Override keys in Vagrantfile template
	vagrantKeyOverides, ok := machinedef["vagrant"]
	if ok {
		for k, v := range vagrantKeyOverides.(map[interface{}]interface{}) {
			output = strings.Replace(vagrantfile, "{"+k.(string)+"}", fmt.Sprintf("%v", v), -1)
			vagrantfile = output
		}
	}
	return output, nil
}

/*
  MakeBoxList Extract List of boxes from machines.yaml machine_templates
*/
func MakeBoxList(options *Options) map[string]string {
	output := make(map[string]string)
	for name, template := range options.machineTemplates.(map[interface{}]interface{}) {
		v := template.(map[interface{}]interface{})["vagrant"]
		b := v.(map[interface{}]interface{})["box"].(string)
		output[name.(string)] = b
	}
	return output
}
