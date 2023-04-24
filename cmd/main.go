package main

import (
	"flag"
	"github.com/onflow/execution-debugger"
	"github.com/onflow/execution-debugger/debuggers"
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

	/*txid, err := flow.HexStringToIdentifier(tx)
	if err != nil {
		log.Error().
			Err(err).
			Msg("Could not parse transaction ID.")
		return
	}*/

	chain := flow.Mainnet.Chain()
	//ctx := context.Background()

	host = "archive.mainnet.nodes.onflow.org:9000"
	conn, err := grpc.Dial(
		host,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		err = errors.Wrap(err, "could not connect to archive node")
		panic(err)
	}
	client := dps.NewAPIClient(conn)

	/*
		txResolver := &debugger.CustomTransaction{
			Tx: &flow.TransactionBody{
				ReferenceBlockID: flow.MustHexStringToIdentifier("a9969efdf3eea714d648e206f62adee037f2a0a598f88b8e161a539e6489d5b4"),
				Script: []byte(`
					transaction {
						execute {
							var x: String = "Hello World"
							var z: String = "Nooo"
							log(x)
						}
					}
				`),
				GasLimit: 1000,
			},
			Height: 49956947,
		}*/

	txResolver := &debugger.NetworkTransactions{
		Client: client,
		ID:     flow.MustHexStringToIdentifier("79c1fcb19d5a56cc0515ffe65e4beb4980e141327f8924a0acce4075309ed52d"),
	}

	dbg, err := debuggers.NewExecutionDebugger(chain, client, log.Logger)
	if err != nil {
		log.Error().Err(err).Msg("New debugger error.")
		return
	}

	_, txErr, err := dbg.DebugTransaction(txResolver)
	if txErr != nil {
		log.Error().Err(txErr).Msg("Transaction error.")
		return
	}
	if err != nil {
		log.Error().Err(err).Msg("Implementation error.")
		return
	}

	//log.Info().Msgf("register reads: %v", result.RegisterReads)
	//log.Info().Msgf("contracts: %v", result.ContractImports.Contracts())
}
