package main

import (
	"context"
	"flag"
	debugger "github.com/janezpodhostnik/flow-transaction-info"
	"github.com/janezpodhostnik/flow-transaction-info/debuggers"
	"github.com/onflow/flow-dps/api/dps"
	"github.com/onflow/flow-go/model/flow"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"os"
)

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	var host string
	flag.StringVar(&host, "host", "", "host url with port")

	var tx string
	flag.StringVar(&tx, "tx", "", "transaction id")

	flag.Parse()

	txid, err := flow.HexStringToIdentifier(tx)
	if err != nil {
		log.Error().
			Err(err).
			Msg("Could not parse transaction ID.")
		return
	}

	chain := flow.Mainnet.Chain()
	ctx := context.Background()

	conn, err := grpc.Dial(
		host,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		err = errors.Wrap(err, "could not connect to archive node")
		panic(err)
	}
	client := dps.NewAPIClient(conn)

	txResolver := &debugger.NetworkTransactions{
		Client: client,
		ID:     txid,
	}

	txErr, err := debuggers.
		NewTransactionDebugger(txResolver, host, client, chain, log.Logger).
		RunTransaction(ctx)

	if txErr != nil {
		log.Error().
			Err(txErr).
			Msg("Transaction error.")
		return
	}
	if err != nil {
		log.Error().
			Err(txErr).
			Msg("Implementation error.")
		return
	}
}
