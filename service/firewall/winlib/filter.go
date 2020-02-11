package winlib

import (
	"syscall"
)

// Condition represents filter condition
type Condition interface {
	Apply(filter syscall.Handle, conditionIndex uint32) error
}

// Filter - WFP filter
type Filter struct {

	// TODO: make fiels not visible outside of package

	Key         syscall.GUID
	KeyLayer    syscall.GUID
	KeySublayer syscall.GUID
	KeyProvider syscall.GUID

	DisplayDataName        string
	DisplayDataDescription string

	Action     FwpActionType
	Weight     byte
	Flags      FwpmFilterFlags
	Conditions []Condition
}

// NewFilter - create new filter
func NewFilter(
	keyProvider syscall.GUID,
	keyLayer syscall.GUID,
	keySublayer syscall.GUID,
	dispName string,
	dispDescription string) Filter {

	return Filter{
		Key:                    NewGUID(),
		Conditions:             make([]Condition, 0, 1),
		KeyProvider:            keyProvider,
		KeyLayer:               keyLayer,
		KeySublayer:            keySublayer,
		DisplayDataName:        dispName,
		DisplayDataDescription: dispDescription}
}

// AddCondition adds filter condition
func (f *Filter) AddCondition(c Condition) {
	f.Conditions = append(f.Conditions, c)
}

// SetDisplayData adds filter display data
func (f *Filter) SetDisplayData(name string, description string) {
	f.DisplayDataName = name
	f.DisplayDataDescription = description
}
