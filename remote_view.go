package debugger

import (
	"github.com/onflow/execution-debugger/registers"
	"github.com/onflow/flow-go/fvm/state"
	storageState "github.com/onflow/flow-go/fvm/storage/state"
	"github.com/onflow/flow-go/model/flow"
)

type RemoteView struct {
	Parent *RemoteView
	Delta  map[flow.RegisterID]flow.RegisterValue

	registerReader registers.RegisterGetRegisterFunc
}

func NewRemoteView(reader registers.RegisterGetRegisterFunc) *RemoteView {
	return &RemoteView{
		Delta:          make(map[flow.RegisterID]flow.RegisterValue),
		registerReader: reader,
	}
}

func (v *RemoteView) NewChild() *storageState.ExecutionState {
	rv := &RemoteView{
		Parent: v,
		Delta:  make(map[flow.RegisterID]flow.RegisterValue),
	}

	return storageState.NewExecutionState(rv, storageState.DefaultParameters())
}

func (v *RemoteView) Set(id flow.RegisterID, value flow.RegisterValue) error {
	v.Delta[id] = value
	return nil
}

func (v *RemoteView) Get(id flow.RegisterID) (flow.RegisterValue, error) {
	// first check the delta
	value, found := v.Delta[id]
	if found {
		return value, nil
	}

	// then call the parent (if exist)
	if v.Parent != nil {
		return v.Parent.Get(id)
	}

	// last use the getRemoteRegister
	resp, err := v.registerReader(id.Owner, id.Key)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (v *RemoteView) DropChanges() error {
	v.Delta = make(map[flow.RegisterID]flow.RegisterValue)
	return nil
}

func (v *RemoteView) Merge(child *state.ExecutionSnapshot) error {
	// todo check
	for id, val := range child.WriteSet {
		v.Delta[id] = val
	}

	return nil
}

func (v *RemoteView) Finalize() *state.ExecutionSnapshot {
	//TODO implement me
	panic("implement me")
}

func (v *RemoteView) Peek(id flow.RegisterID) (flow.RegisterValue, error) {
	return v.Delta[id], nil
}

// returns all the registers that has been touched
func (v *RemoteView) AllRegisters() []flow.RegisterID {
	panic("Not implemented yet")
}

func (v *RemoteView) RegisterUpdates() ([]flow.RegisterID, []flow.RegisterValue) {
	panic("Not implemented yet")
}

func (v *RemoteView) Touch(owner, key string) error {
	// no-op for now
	return nil
}
