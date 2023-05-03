package debugger

import (
	"context"
	"github.com/onflow/flow-dps/api/dps"
	"github.com/onflow/flow-dps/codec/zbor"
	"github.com/onflow/flow-go/model/flow"
	"github.com/pkg/errors"
)

type TransactionResolver interface {
	TransactionBody() (*flow.TransactionBody, error)
	BlockHeight() (uint64, error)
}

var _ TransactionResolver = &NetworkTransactions{}

// NetworkTransactions implements transaction resolver that fetches existing transaction
// from the Flow network using the archive node client.
type NetworkTransactions struct {
	Client dps.APIClient
	ID     flow.Identifier
}

func (n *NetworkTransactions) TransactionBody() (*flow.TransactionBody, error) {
	response, err := n.Client.GetTransaction(
		context.Background(),
		&dps.GetTransactionRequest{
			TransactionID: n.ID[:],
		},
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get transaction from the network")
	}

	codec := zbor.NewCodec()
	var txBody flow.TransactionBody
	err = codec.Unmarshal(response.Data, &txBody)
	if err != nil {
		return nil, errors.Wrap(err, "failed decoding transaction")
	}

	return &txBody, nil
}

func (n *NetworkTransactions) BlockHeight() (uint64, error) {
	response, err := n.Client.GetHeightForTransaction(
		context.Background(),
		&dps.GetHeightForTransactionRequest{
			TransactionID: n.ID[:],
		},
	)
	if err != nil {
		return 0, errors.Wrap(err, "failed to get transaction height from the network")
	}

	return response.Height, nil
}

var _ TransactionResolver = &CustomTransaction{}

// CustomTransaction implements transaction resolver that returns a transaction that was
// provided on the initialization, used for custom transactions you manually build.
type CustomTransaction struct {
	Tx     *flow.TransactionBody
	Height uint64
}

func (c *CustomTransaction) TransactionBody() (*flow.TransactionBody, error) {
	return c.Tx, nil
}

func (c *CustomTransaction) BlockHeight() (uint64, error) {
	return c.Height, nil
}
