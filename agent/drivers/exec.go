package drivers

import (
	"fmt"
	"os"
	"os/exec"
	"errors"

	"taylor/lib/structs"
	"taylor/lib/util"
)

func run(job *structs.Job, driver *structs.Driver, onJobUpdate func (job *structs.Job, progress float32, message string), ctx interface{}) error {
	
	fmt.Printf("Exec driver %s\n", driver.Name)

	fmt.Printf("%+v\n", *job)

	driverConfig := job.DriverConfig

	shellCmd, err := util.GetString(driverConfig, "cmd", "")
	if err != nil {
		return err
	}
	if shellCmd == "" {
		return errors.New("no cmd found")
	}

	shellArgs, err := util.GetArrayOfStrings(driverConfig, "args", []string{})
	if err != nil {
		return err
	}

	cmd := exec.Command(shellCmd, shellArgs...)

	inheritEnv, err := util.GetBool(driverConfig, "inherit_env", false)
	if err != nil {
		return err
	}
	if inheritEnv == true {
		cmd.Env = append(cmd.Env, os.Environ()...)
	}

	envVars, err := util.GetArrayOfStrings(driverConfig, "env", []string{})
	if err != nil {
		return err
	}
	if len(envVars) > 0 {
		cmd.Env = append(cmd.Env, envVars...)
	}

	shellDir, err := util.GetString(driverConfig, "dir", "")
	if err != nil {
		return err
	}
	if shellDir != "" {
		cmd.Dir = shellDir
	}

	fmt.Printf("Exec %s with Args %v, [Env: %v, Dir: %s]\n", shellCmd, shellArgs, cmd.Env, cmd.Dir)

	//onJobUpdate(job, 0, "Start")
	if err := cmd.Run(); err != nil {
		return err
	}
	//onJobUpdate(job, 1, "End")

	fmt.Printf("Exec success %s\n", driver.Name)

	return nil
}

func NewExecDriver(ctx interface{}) *structs.Driver {
	return &structs.Driver{
		Name:		"exec",
		Run:		run,
		Ctx:		ctx,
	}
}
