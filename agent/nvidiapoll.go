package agent

import (
	"fmt"
	"io"
	"bufio"
	"strings"
	"strconv"
	"os"
	"time"
	"os/exec"
	"errors"

	"taylor/lib/structs"
)

type OnGpuInfoFn func([]structs.GpuInfo)

func readPipe(pipe io.ReadCloser, onText func (line string)) {
	scanner := bufio.NewScanner(pipe)
	for scanner.Scan() {
		// To-DO: rewrite this to not use Scanner because of its maxline length problem
		line := scanner.Text()
		onText(strings.TrimRight(line, "\r\n"))
	}
}

func parseNvidiaOut(text string) (structs.GpuInfo, error) {
	data := structs.GpuInfo{}

	parts := strings.Split(text, ",")
	if len(parts) != 5 {
		return data, errors.New("invalid nvidia-smi string")
	}

	for i, p := range parts {
		parts[i] = strings.ReplaceAll(p, " ", "")
	}

	data.NameGPU = parts[0]
	temp, err := strconv.Atoi(parts[1])
	if err != nil {
		return data, err
	}
	data.Temperature = temp

	temp, err = strconv.Atoi(parts[2])
	if err != nil {
		return data, err
	}
	data.MemoryTotalMB = temp

	temp, err = strconv.Atoi(parts[3])
	if err != nil {
		return data, err
	}
	data.MemoryFreeMB = temp

	temp, err = strconv.Atoi(parts[4])
	if err != nil {
		return data, err
	}
	data.Utilization = temp

	return data, nil
}

func startPollGPUData(cfg NvidiaConfig, onGpuInfo OnGpuInfoFn) {
	fmt.Println("Start Poll GPU Data")
	for {
		time.Sleep(cfg.PollTimeMs * time.Millisecond)

		args := []string{
			 "--query-gpu=name,temperature.gpu,memory.total,memory.free,utilization.gpu",
			 "--format=csv,nounits,noheader",
		 }

		cmd := exec.Command(cfg.NvidiaSmiPath, args...)
		cmd.Env = append(cmd.Env, os.Environ()...)
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			fmt.Println(err)
			continue
		}
		stderr, err := cmd.StderrPipe()
		if err != nil {
			fmt.Println(err)
			continue
		}

		if err := cmd.Start(); err != nil {
			fmt.Println(err)
			continue
		}
		// read stderr in background and stdout in this thread
		waitStderr := make(chan int, 0)
		go func () {
			readPipe(stderr, func (text string) {
				fmt.Println("STDERR >> ", text)
			})
			waitStderr<- 1
		}()

		gpuData := make([]structs.GpuInfo, 0)
		readPipe(stdout, func (text string) {
			pollData, err := parseNvidiaOut(text)
			if err != nil {
				fmt.Println(err)
				return
			}
			gpuData = append(gpuData, pollData)
		})

		// wait for stderr
		<-waitStderr

		// wait for process
		err = cmd.Wait()
		if err != nil {
			fmt.Println(err)
			continue
		}

		onGpuInfo(gpuData)
	}
}
