package debuggers

import (
	"fmt"
	debugger "github.com/onflow/execution-debugger"
	"github.com/onflow/execution-debugger/registers"
	"github.com/onflow/flow-dps/api/dps"
	"github.com/onflow/flow-go/model/flow"
	"github.com/rs/zerolog"
)

type ExecutionDebugger struct {
	log           zerolog.Logger
	archiveClient dps.APIClient
	chain         flow.Chain
}

type DebugResult struct {
	RegisterReads   *registers.RegisterReadTracker
	ContractImports *registers.ContractImportsTracker
	ProfileBuilder  *debugger.ProfileBuilder
	LogInterceptor  *debugger.LogInterceptor
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

	registerReads := registers.NewRemoteRegisterReadTracker(e.log)
	contractImports := registers.NewCaptureContractWrapper(e.log)

	readWrappers := []registers.RegisterGetWrapper{
		cache,
		registerReads,
		contractImports,
	}

	readFunc := registers.NewRemoteReader(e.archiveClient, blockHeight)
	readFunc.Wrap(readWrappers...)

	view := debugger.NewRemoteView(readFunc)
	profiler := debugger.NewProfileBuilder()
	logInterceptor := debugger.NewLogInterceptor(e.log)

	dbg := NewRemoteDebugger(view, e.chain, e.log.Output(logInterceptor), []CadenceStatementHandler{profiler})
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

	return &DebugResult{
		RegisterReads:   registerReads,
		ContractImports: contractImports,
		ProfileBuilder:  profiler,
		LogInterceptor:  logInterceptor,
	}, txErr, err
}

type FileSaver interface {
	Save(string) error
}
