package server

import (
	"fmt"
	"os"
	"bufio"
	"errors"
	"path"

	"taylor/lib/structs"
)

type DiskLog struct {
	dir		string
	files		map[string]*os.File
}

func NewDiskLog(dir string) *DiskLog {
	return &DiskLog{
		dir:	dir,
		files:  make(map[string]*os.File, 0),
	}
}
func (d *DiskLog) makeLogfilePath(job *structs.Job) string {
	return path.Join(d.dir, fmt.Sprintf("%s.log", job.Id))
}

func (d *DiskLog) Open(job *structs.Job) error {
	_, in := d.files[job.Id]
	if in == true {
		return errors.New(fmt.Sprintf("Log already open for job: %s\n", job.Id))
	}
	f, err := os.Create(d.makeLogfilePath(job))
	if err != nil {
		return err
	}
	d.files[job.Id] = f
	return nil
}

func (d *DiskLog) WriteString(job *structs.Job, str string) (int, error) {
	f, in := d.files[job.Id]
	if in == false {
		return 0, errors.New(fmt.Sprintf("No open log for job: %s\n", job.Id))
	}
	return f.WriteString(str)
}

func (d *DiskLog) Close(job *structs.Job) error {
	f, in := d.files[job.Id]
	if in == false {
		return errors.New(fmt.Sprintf("No open log for job: %s\n", job.Id))
	}
	f.Close()
	delete(d.files, job.Id)
	return nil
}

func (d *DiskLog) GetLogs(job *structs.Job) ([]string, error) {
	lines := []string{}

	f, err := os.Open(d.makeLogfilePath(job))
	if err != nil {
		return lines, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return lines, err
	}

	return lines, nil
}
