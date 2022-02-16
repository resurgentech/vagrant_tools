package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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

func RunCommand(cmd string, args []string, cwd string) (string, error) {
	c := exec.Command(cmd, args...)
	c.Dir = cwd

	out, err := c.CombinedOutput()
	if err != nil && len(out) != 0 {
		err = fmt.Errorf("%s: %s", err, out)
	}

	s := string(out)
	fmt.Println("execution of %v complete: %s", c.Args, s)

	return s, err
}
