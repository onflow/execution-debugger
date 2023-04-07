package debugger

import (
	"context"
	"encoding/json"
	"github.com/onflow/flow-dps/api/dps"
	"github.com/onflow/flow-go/model/flow"
	"github.com/pkg/errors"
)

type TransactionResolver interface {
	Body() (*flow.TransactionBody, error)
}

var _ TransactionResolver = &NetworkTransactions{}

type NetworkTransactions struct {
	Client dps.APIClient
	ID     flow.Identifier
}

func (n *NetworkTransactions) Body() (*flow.TransactionBody, error) {
	response, err := n.Client.GetTransaction(context.Background(), &dps.GetTransactionRequest{
		TransactionID: n.ID[:],
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get transaction from the network")
	}

	var body flow.TransactionBody
	err = json.Unmarshal(response.Data, &body)
	if err != nil {
		return nil, errors.Wrap(err, "failed decoding transaction")
	}

	return &body, nil
}

type CustomTransaction struct {
	Tx *flow.TransactionBody
}

func (c *CustomTransaction) Body() (*flow.TransactionBody, error) {
	return c.Tx, nil
}
