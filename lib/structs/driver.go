package structs

type Driver struct {
	Name	string			`json:"name"`

	// not exported via json
	Run	func (job *Job, driver *Driver, ctx interface{}) error
	Ctx	interface{}	
}
