package structs

type Driver struct {
	Name		string			`json:"name"`

	// not exported via json
	Run		func (job *Job, driver *Driver, onJobUpdate func (job *Job, progress float32, message string), ctx interface{}) error
	Ctx		interface{}	
}
