package debuggers

import (
	"fmt"
	debugger "github.com/onflow/execution-debugger"
	"github.com/onflow/execution-debugger/registers"
	"github.com/onflow/flow-archive/api/archive"
	"github.com/onflow/flow-go/model/flow"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type ExecutionDebugger struct {
	log           zerolog.Logger
	archiveClient archive.APIClient
	chain         flow.Chain
}

type DebugResult struct {
	RegisterReads   *registers.RegisterReadTracker
	ContractImports *registers.ContractImportsTracker
	ProfileBuilder  *debugger.ProfileBuilder
	LogInterceptor  *debugger.LogInterceptor
	Execution       *Execution
}

func NewMainnetExecutionDebugger(log zerolog.Logger) (*ExecutionDebugger, error) {
	chain := flow.Mainnet.Chain()
	conn, err := grpc.Dial(
		"archive.mainnet.nodes.onflow.org:9000",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, errors.Wrap(err, "could not connect to archive node")
	}

	return NewExecutionDebugger(chain, archive.NewAPIClient(conn), log)
}

func NewTestnetExecutionDebugger(log zerolog.Logger) (*ExecutionDebugger, error) {
	chain := flow.Mainnet.Chain()
	conn, err := grpc.Dial(
		"archive.testnet.nodes.onflow.org:9000",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, errors.Wrap(err, "could not connect to archive node")
	}

	return NewExecutionDebugger(chain, archive.NewAPIClient(conn), log)
}

func NewExecutionDebugger(
	chain flow.Chain,
	archiveClient archive.APIClient,
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
) (*DebugResult, error) {

	blockHeight, err := txResolver.BlockHeight()
	if err != nil {
		return nil, err
	}

	cache, err := registers.NewRemoteRegisterFileCache(blockHeight, e.log)
	if err != nil {
		return nil, err
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
		return nil, err
	}

	e.log.Info().Msg(fmt.Sprintf("Debugging transaction with ID %s at block height %d", txBody.ID(), blockHeight))

	txResult, err := dbg.RunTransaction(txBody)

	return &DebugResult{
		RegisterReads:   registerReads,
		ContractImports: contractImports,
		ProfileBuilder:  profiler,
		LogInterceptor:  logInterceptor,
		Execution:       txResult,
	}, err
}

func (e *ExecutionDebugger) Client() archive.APIClient {
	return e.archiveClient
}

type FileSaver interface {
	Save(string) error
}
