package restify

// Feature struct represents a set of features that can be enabled or disabled.
// Each feature has a corresponding boolean field that indicates whether it is enabled or disabled.
type Feature struct {
	DisableCreate    bool
	DisableUpdate    bool
	DisableList      bool
	DisableDelete    bool
	DisableSet       bool
	DisableAggregate bool
	API              bool
}

// DisableCreate is a flag to disable the creation of new objects.
type DisableCreate struct{}

// DisableUpdate is a flag to disable update of existing objects.
type DisableUpdate struct{}

// DisableList is a flag to disable listing objects.
type DisableList struct{}

// DisableDelete is a flag to disable the deletion of existing objects
type DisableDelete struct{}

// DisableSet is a flag to disable set endpoint
type DisableSet struct{}

// API is a flag to enable restful API endpoints.
type API struct{}

// DisableAggregate is a flag to disable aggregation endpoints.
type DisableAggregate struct{}
