package drivers

import (
	"fmt"
	"os"
	"os/exec"
	"errors"

	"taylor/lib/structs"
)

func getBool(cfg map[string]interface{}, key string, def bool) (bool, error) {
	v, in := cfg[key]
	if in == false {
		return def, nil
	}
	casted, ok := v.(bool)
	if ok == false {
		return def, errors.New(fmt.Sprintf("Error casting %s to bool", key))
	}
	return casted, nil
}

func getString(cfg map[string]interface{}, key string, def string) (string, error) {
	v, in := cfg[key]
	if in == false {
		return def, nil
	}
	casted, ok := v.(string)
	if ok == false {
		return def, errors.New(fmt.Sprintf("Error casting %s to string", key))
	}
	return casted, nil
}

func getArrayOfStrings(cfg map[string]interface{}, key string, def []string) ([]string, error) {
	v, in := cfg[key]
	if in == false {
		return def, nil
	}

	var tmp []interface{}
	tmp, ok := v.([]interface{})
	if ok == false {
		return def, errors.New(fmt.Sprintf("Error casting %s to []string", key))
	}
	if len(tmp) == 0 {
		return def, nil
	}

	casted := make([]string, len(tmp))
	for i, arg := range tmp {
		casted[i], ok = arg.(string)
		if ok == false {
			return def, errors.New(fmt.Sprintf("Error casting array element %d of %s to string", i, key))
		}
	}
	return casted, nil
}

func run(job *structs.Job, driver *structs.Driver, ctx interface{}) error {
	
	fmt.Printf("Exec driver %s\n", driver.Name)

	fmt.Printf("%+v\n", *job)

	driverConfig := job.DriverConfig

	shellCmd, err := getString(driverConfig, "cmd", "")
	if err != nil {
		return err
	}
	if shellCmd == "" {
		return errors.New("no cmd found")
	}

	shellArgs, err := getArrayOfStrings(driverConfig, "args", []string{})
	if err != nil {
		return err
	}

	cmd := exec.Command(shellCmd, shellArgs...)

	inheritEnv, err := getBool(driverConfig, "inherit_env", false)
	if err != nil {
		return err
	}
	if inheritEnv == true {
		cmd.Env = append(cmd.Env, os.Environ()...)
	}

	envVars, err := getArrayOfStrings(driverConfig, "env", []string{})
	if err != nil {
		return err
	}
	if len(envVars) > 0 {
		cmd.Env = append(cmd.Env, envVars...)
	}

	shellDir, err := getString(driverConfig, "dir", "")
	if err != nil {
		return err
	}
	if shellDir != "" {
		cmd.Dir = shellDir
	}

	fmt.Printf("Exec %s with Args %v, [Env: %v, Dir: %s]\n", shellCmd, shellArgs, cmd.Env, cmd.Dir)

	if err := cmd.Run(); err != nil {
		return err
	}

	fmt.Printf("Exec success %s\n", driver.Name)

	return nil
}

func NewExecDriver() *structs.Driver {
	return &structs.Driver{
		Name: "exec",
		Run:   run,
		Ctx:   nil,
	}
}

