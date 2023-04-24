package debuggers

import (
	"context"
	"fmt"
	"github.com/onflow/execution-debugger"
	"github.com/onflow/execution-debugger/registers"
	"github.com/onflow/flow-dps/api/dps"
	"github.com/onflow/flow-go/model/flow"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"io"
	"math/rand"
	"os"
	"path/filepath"
)

type TransactionDebugger struct {
	txResolver  debugger.TransactionResolver
	dpsClient   dps.APIClient
	archiveHost string
	chain       flow.Chain
	directory   string
	log         zerolog.Logger
}

func NewTransactionDebugger(
	txResolver debugger.TransactionResolver,
	archiveHost string,
	dpsClient dps.APIClient,
	chain flow.Chain,
	logger zerolog.Logger) *TransactionDebugger {

	return &TransactionDebugger{
		txResolver:  txResolver,
		archiveHost: archiveHost,
		chain:       chain,
		dpsClient:   dpsClient,

		directory: fmt.Sprintf("t_%d", rand.Intn(1000)), // TODO remove

		log: logger,
	}
}

type clientWithConnection struct {
	dps.APIClient
	*grpc.ClientConn
}

func (d *TransactionDebugger) RunTransaction(ctx context.Context) (txErr, processError error) {
	client, err := d.getClient()
	if err != nil {
		return nil, err
	}
	defer func() {
		err := client.Close()
		if err != nil {
			d.log.Warn().
				Err(err).
				Msg("Could not close client connection.")
		}
	}()

	blockHeight, err := d.txResolver.BlockHeight()
	if err != nil {
		return nil, err
	}

	cache, err := registers.NewRemoteRegisterFileCache(blockHeight, d.log)
	if err != nil {
		return nil, err
	}

	wrappers := []registers.RegisterGetWrapper{
		cache,
		registers.NewRemoteRegisterReadTracker(d.log),
		registers.NewCaptureContractWrapper(d.log),
	}

	readFunc := registers.NewRemoteReader(client, blockHeight)
	readFunc.Wrap(wrappers...)

	view := debugger.NewRemoteView(readFunc)

	logInterceptor := debugger.NewLogInterceptor(d.log, d.directory)
	defer func() {
		err := logInterceptor.Close()
		if err != nil {
			d.log.Warn().
				Err(err).
				Msg("Could not close log interceptor.")
		}
	}()

	dbg := NewRemoteDebugger(view, d.chain, d.log.Output(logInterceptor), nil)
	defer func(debugger *RemoteDebugger) {
		err := debugger.Close()
		if err != nil {
			d.log.Warn().
				Err(err).
				Msg("Could not close debugger.")
		}
	}(dbg)

	//err = d.dumpTransactionToFile(txBody)

	txBody, err := d.txResolver.TransactionBody()
	if err != nil {
		return nil, err
	}

	d.log.Info().Msg(fmt.Sprintf("Debugging transaction with ID %s at block height %d", txBody.ID(), blockHeight))

	txErr, err = dbg.RunTransaction(txBody)

	for _, wrapper := range wrappers {
		switch w := wrapper.(type) {
		case io.Closer:
			err := w.Close()
			if err != nil {
				d.log.Warn().
					Err(err).
					Msg("Could not close register read wrapper.")
			}
		}
	}

	return txErr, err
}

func (d *TransactionDebugger) getClient() (clientWithConnection, error) {
	conn, err := grpc.Dial(
		d.archiveHost,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		d.log.Error().
			Err(err).
			Str("host", d.archiveHost).
			Msg("Could not connect to server.")
		return clientWithConnection{}, err
	}
	client := dps.NewAPIClient(conn)

	return clientWithConnection{
		APIClient:  client,
		ClientConn: conn,
	}, nil
}

func (d *TransactionDebugger) dumpTransactionToFile(body flow.TransactionBody) error {
	filename := d.directory + "/transaction.cdc"
	err := os.MkdirAll(filepath.Dir(filename), os.ModePerm)
	if err != nil {
		return err
	}
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer func(csvFile *os.File) {
		err := csvFile.Close()
		if err != nil {
			d.log.Warn().
				Err(err).
				Msg("Could not close file.")
		}
	}(file)

	_, err = file.WriteString(string(body.Script))
	return err
}
