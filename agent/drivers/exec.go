package drivers

import (
	"fmt"
	"os"
	"io"
	"bufio"
	"os/exec"
	"errors"
	"strings"
	"sync"

	"taylor/lib/structs"
	"taylor/lib/util"
)

type ProcessWrapper struct {
	process		*os.Process
	interrupted	bool
}

type DriverContext struct {
	pidMapMtx	*sync.Mutex
	jobPidMap	map[string] *ProcessWrapper
}

func (c *DriverContext) HasProcess(jobId string) bool {
	_, in := c.jobPidMap[jobId]
	return in
}

func (c *DriverContext) GetProcess(jobId string) (*ProcessWrapper, bool) {
	p, in := c.jobPidMap[jobId]
	return p, in
}

func (c *DriverContext) RemoveProcess(jobId string) bool {
	c.pidMapMtx.Lock()
	defer c.pidMapMtx.Unlock()

	_, in := c.jobPidMap[jobId]
	if in == false {
		return false
	}

	delete(c.jobPidMap, jobId)

	return true
}

func (c *DriverContext) AddProcess(jobId string, process *os.Process) error {
	c.pidMapMtx.Lock()
	defer c.pidMapMtx.Unlock()

	_, in := c.jobPidMap[jobId]
	if in == true {
		return errors.New(fmt.Sprintf("jobPidMap already has pid for job: %s", jobId))
	}

	c.jobPidMap[jobId] = &ProcessWrapper{
		process: process,
		interrupted: false,
	}

	return nil
}

func cancel(job *structs.Job, driver *structs.Driver) error {
	context, _ := driver.Ctx.(*DriverContext)

	fmt.Printf("Cancel driver %s\n", driver.Name)

	fmt.Printf("%+v\n", *job)

	processWrapper, in := context.GetProcess(job.Id)
	if in == false {
		return errors.New(fmt.Sprintf("Job with id not executing: %s", job.Id))
	}

	// to-do: send os.Kill on windows
	err := processWrapper.process.Signal(os.Interrupt)
	if err != nil {
		return err
	}
	processWrapper.interrupted = true

	fmt.Println("Interrupt signal sent")

	return nil
}

func run(job *structs.Job, driver *structs.Driver, onJobUpdate func (job *structs.Job, progress float32, message string)) (bool, error) {
	context, _ := driver.Ctx.(*DriverContext)
	
	fmt.Printf("Exec driver %s\n", driver.Name)

	fmt.Printf("%+v\n", *job)

	if context.HasProcess(job.Id) {
		return false, errors.New(fmt.Sprintf("Job with id already executing: %s", job.Id))
	}

	driverConfig := job.DriverConfig

	shellCmd, err := util.GetString(driverConfig, "cmd", "")
	if err != nil {
		return false, err
	}
	if shellCmd == "" {
		return false, errors.New("no cmd found")
	}

	shellArgs, err := util.GetArrayOfStrings(driverConfig, "args", []string{})
	if err != nil {
		return false, err
	}

	cmd := exec.Command(shellCmd, shellArgs...)

	inheritEnv, err := util.GetBool(driverConfig, "inherit_env", false)
	if err != nil {
		return false, err
	}
	if inheritEnv == true {
		cmd.Env = append(cmd.Env, os.Environ()...)
	}

	envVars, err := util.GetArrayOfStrings(driverConfig, "env", []string{})
	if err != nil {
		return false, err
	}
	if len(envVars) > 0 {
		cmd.Env = append(cmd.Env, envVars...)
	}

	shellDir, err := util.GetString(driverConfig, "dir", "")
	if err != nil {
		return false, err
	}
	if shellDir != "" {
		cmd.Dir = shellDir
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return false, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return false, err
	}

	fmt.Printf("Exec %s with Args %v, [Env: %v, Dir: %s]\n", shellCmd, shellArgs, cmd.Env, cmd.Dir)
	// finally: Start the cmd. Now Process field will be set in cmd
	if err := cmd.Start(); err != nil {
		return false, err
	}

	// save process so that we can kill it on request
	context.AddProcess(job.Id, cmd.Process)
	defer context.RemoveProcess(job.Id)

	// read stderr in background and stdout in this thread
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

	// wait for stderr
	<-waitStderr

	// wait for process
	err = cmd.Wait()
	if err != nil {
		fmt.Printf("cmd.Wait err %s\n", err.Error())
		processWrapper, in := context.GetProcess(job.Id)
		if in == false {
			// nothing we can do
			return false, err
		}
		if processWrapper.interrupted == true {
			return true, err
		} else {
			return false, err
		}
	}
	// test if we have been interrupted

	fmt.Printf("Exec success %s\n", driver.Name)

	return false, nil
}

func readPipe(pipe io.ReadCloser, onText func (line string)) {
	scanner := bufio.NewScanner(pipe)
	for scanner.Scan() {
		// To-DO: rewrite this to not use Scanner because of its maxline length problem
		line := scanner.Text()
		onText(strings.TrimRight(line, "\r\n"))
	}
}


func NewExecDriver() *structs.Driver {
	ctx := &DriverContext{
		pidMapMtx:	&sync.Mutex{},
		jobPidMap:	make(map[string]*ProcessWrapper),
	}

	return &structs.Driver{
		Name:		"exec",
		Run:		run,
		Cancel:		cancel,
		Ctx:		ctx,
	}
}
