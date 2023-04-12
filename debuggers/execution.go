package debuggers

import (
	"fmt"
	debugger "github.com/onflow/execution-debugger"
	"github.com/onflow/execution-debugger/registers"
	"github.com/onflow/flow-dps/api/dps"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"io"
)

type ExecutionDebugger struct {
	logger        zerolog.Logger
	archiveClient dps.APIClient
}

func NewExecutionDebugger(
	archiveHost string,
	txResolver debugger.TransactionResolver,
	logger zerolog.Logger,
) (*ExecutionDebugger, error) {
	conn, err := grpc.Dial(
		archiveHost,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		logger.Error().
			Err(err).
			Str("host", archiveHost).
			Msg("Could not connect to server.")
		return nil, err
	}
	client := dps.NewAPIClient(conn)

	return &ExecutionDebugger{
		archiveClient: client,
	}, nil
}

func (e *ExecutionDebugger) DebugTransaction(txResolver debugger.TransactionResolver) (txErr, processError error) {
	blockHeight, err := txResolver.BlockHeight()
	if err != nil {
		return nil, err
	}

	cache, err := registers.NewRemoteRegisterFileCache(blockHeight, d.log)
	if err != nil {
		return nil, err
	}

	wrappers := []registers.RegisterGetWrapper{
		cache,
		registers.NewRemoteRegisterReadTracker(d.directory, d.log),
		registers.NewCaptureContractWrapper(d.directory, d.log),
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

	dbg := NewRemoteDebugger(view, d.chain, d.directory, d.log.Output(logInterceptor))
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
