package registers

import (
	"context"
	"github.com/onflow/flow-dps/api/dps"
	"github.com/onflow/flow-go/engine/execution/state"
	"github.com/onflow/flow-go/ledger/common/pathfinder"
	"github.com/onflow/flow-go/ledger/complete"
	"github.com/onflow/flow-go/model/flow"
)

type RegisterGetRegisterFunc func(string, string) (flow.RegisterValue, error)

func (r RegisterGetRegisterFunc) Wrap(wrappers ...RegisterGetWrapper) {
	for _, wrapper := range wrappers {
		r = wrapper.Wrap(r) // TODO check otherwise return r
	}
}

type RegisterGetWrapper interface {
	Wrap(RegisterGetRegisterFunc) RegisterGetRegisterFunc
}

func NewRemoteReader(client dps.APIClient, blockHeight uint64) RegisterGetRegisterFunc {
	return func(key string, address string) (flow.RegisterValue, error) {
		ledgerKey := state.RegisterIDToKey(flow.RegisterID{Key: key, Owner: address})
		ledgerPath, err := pathfinder.KeyToPath(ledgerKey, complete.DefaultPathFinderVersion)
		if err != nil {
			return nil, err
		}

		resp, err := client.GetRegisterValues(context.Background(), &dps.GetRegisterValuesRequest{
			Height: blockHeight,
			Paths:  [][]byte{ledgerPath[:]},
		})
		if err != nil {
			return nil, err
		}
		return resp.Values[0], nil
	}
}
