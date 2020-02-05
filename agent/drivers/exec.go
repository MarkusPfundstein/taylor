package drivers

import (
	"fmt"
	"os"
	"io"
	"bufio"
	"os/exec"
	"errors"
	"strings"

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

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}

	// read stderr in background and stdout on background
	waitStderr := make(chan int, 0)
	go func () {
		readPipe(stderr, func (text string) {
			onJobUpdate(job, 0, fmt.Sprintf("STDERR >> %s", text))
		})
		waitStderr<- 1
	}()
	readPipe(stdout, func (text string) {
		onJobUpdate(job, 0, fmt.Sprintf("STDOUT >> %s", text))
	})

	<-waitStderr

	if err := cmd.Wait(); err != nil {
		return err
	}

	fmt.Printf("Exec success %s\n", driver.Name)

	return nil
}

func readPipe(pipe io.ReadCloser, onText func (line string)) {
	scanner := bufio.NewScanner(pipe)
	for scanner.Scan() {
		line := scanner.Text()
		onText(strings.TrimRight(line, "\r\n"))
	}
}


func NewExecDriver(ctx interface{}) *structs.Driver {
	return &structs.Driver{
		Name:		"exec",
		Run:		run,
		Ctx:		ctx,
	}
}
