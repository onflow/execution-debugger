package debuggers

import (
	"fmt"
	debugger "github.com/onflow/execution-debugger"
	"github.com/onflow/execution-debugger/registers"
	"github.com/onflow/flow-dps/api/dps"
	"github.com/onflow/flow-go/model/flow"
	"github.com/rs/zerolog"
	"io"
)

type ExecutionDebugger struct {
	log           zerolog.Logger
	archiveClient dps.APIClient
	chain         flow.Chain
}

type DebugResult struct {
	RegisterReads []registers.RegisterReadEntry
}

func NewExecutionDebugger(
	chain flow.Chain,
	archiveClient dps.APIClient,
	log zerolog.Logger,
) (*ExecutionDebugger, error) {
	return &ExecutionDebugger{
		archiveClient: archiveClient,
		log:           log,
		chain:         chain,
	}, nil
}

func (e *ExecutionDebugger) DebugTransaction(
	txResolver debugger.TransactionResolver,
) (result *DebugResult, txErr, processError error) {

	blockHeight, err := txResolver.BlockHeight()
	if err != nil {
		return nil, nil, err
	}

	cache, err := registers.NewRemoteRegisterFileCache(blockHeight, e.log)
	if err != nil {
		return nil, nil, err
	}

	result = &DebugResult{}

	wrappers := []registers.RegisterGetWrapper{
		cache,
		registers.NewRemoteRegisterReadTracker(result.RegisterReads, e.log),
		//registers.NewCaptureContractWrapper(e.directory, e.log),
	}

	readFunc := registers.
		NewRemoteReader(e.archiveClient, blockHeight).
		Wrap(wrappers...)

	view := debugger.NewRemoteView(readFunc)

	dbg := NewRemoteDebugger(view, e.chain, e.log)
	defer func(debugger *RemoteDebugger) {
		err := debugger.Close()
		if err != nil {
			e.log.Warn().Err(err).Msg("Could not close debugger.")
		}
	}(dbg)

	txBody, err := txResolver.TransactionBody()
	if err != nil {
		return nil, nil, err
	}

	e.log.Info().Msg(fmt.Sprintf("Debugging transaction with ID %s at block height %d", txBody.ID(), blockHeight))

	txErr, err = dbg.RunTransaction(txBody)

	for _, wrapper := range wrappers {
		switch w := wrapper.(type) {
		case io.Closer:
			err := w.Close()
			if err != nil {
				e.log.Warn().Err(err).Msg("Could not close register read wrapper.")
			}
		}
	}

	return result, txErr, err
}
