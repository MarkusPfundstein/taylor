package drivers

import (
	"fmt"
	"time"

	"taylor/lib/structs"
)

func run(job *structs.Job, driver *structs.Driver, ctx interface{}) error {
	
	fmt.Printf("Exec driver %s\n", driver.Name)

	time.Sleep(60 * time.Second)

	fmt.Printf("Exec done %s\n", driver.Name)

	return nil
}

func NewExecDriver() *structs.Driver {
	return &structs.Driver{
		Name: "exec",
		Run:   run,
		Ctx:   nil,
	}
}

