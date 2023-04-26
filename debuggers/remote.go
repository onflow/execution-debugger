package debuggers

import (
	"github.com/onflow/cadence"
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/execution-debugger"
	"github.com/onflow/flow-go/fvm"
	fvmRuntime "github.com/onflow/flow-go/fvm/runtime"
	"github.com/onflow/flow-go/fvm/state"
	"github.com/onflow/flow-go/model/flow"
	"github.com/rs/zerolog"
)

type CadenceStatementHandler interface {
	OnStatement(fvmEnv fvmRuntime.Environment, inter *interpreter.Interpreter, statement ast.Statement)
}

type RemoteDebugger struct {
	vm   *fvm.VirtualMachine
	ctx  fvm.Context
	view state.View
}

func NewRemoteDebugger(
	view *debugger.RemoteView,
	chain flow.Chain,
	logger zerolog.Logger,
	statementHandlers []CadenceStatementHandler,
) *RemoteDebugger {
	vm := fvm.NewVirtualMachine()

	// no signature processor here
	// TODO Maybe we add fee-deduction step as well
	ctx := fvm.NewContext(
		fvm.WithLogger(logger),
		fvm.WithChain(chain),
		fvm.WithAuthorizationChecksEnabled(false),
		fvm.WithSequenceNumberCheckAndIncrementEnabled(false),
		fvm.WithReusableCadenceRuntimePool(
			fvmRuntime.NewReusableCadenceRuntimePoolWithConfig(
				1,
				fvmRuntime.ReusableCadenceRuntimePoolConfig{
					OnStatement: func(fvmEnv fvmRuntime.Environment, inter *interpreter.Interpreter, statement ast.Statement) {
						for _, handler := range statementHandlers {
							handler.OnStatement(fvmEnv, inter, statement) // change to interface
						}
					},
				},
			),
		),
	)

	return &RemoteDebugger{
		ctx:  ctx,
		vm:   vm,
		view: view,
	}
}

type TransactionResult struct {
	Events                flow.EventsList
	ComputationUsed       uint64
	MemoryEstimate        uint64
	Logs                  []string
	ReadRegisterIDs       []flow.RegisterID
	UpdatedRegisterIDs    []flow.RegisterID
	BytesWrittenToStorage uint64
	BytesReadFromStorage  uint64
}

// RunTransaction runs the transaction given the latest sealed block data
func (d *RemoteDebugger) RunTransaction(txBody *flow.TransactionBody) (result *TransactionResult, txErr, processError error) {
	blockCtx := fvm.NewContextFromParent(d.ctx, fvm.WithBlockHeader(d.ctx.BlockHeader))
	tx := fvm.Transaction(txBody, 0)
	snapshot, output, err := d.vm.RunV2(blockCtx, tx, d.view)
	if err != nil {
		return nil, nil, err
	}

	return &TransactionResult{
		Events:                output.Events,
		ComputationUsed:       output.ComputationUsed,
		MemoryEstimate:        output.MemoryEstimate,
		Logs:                  output.Logs,
		ReadRegisterIDs:       snapshot.ReadRegisterIDs(),
		UpdatedRegisterIDs:    snapshot.UpdatedRegisterIDs(),
		BytesWrittenToStorage: snapshot.TotalBytesWrittenToStorage(),
		BytesReadFromStorage:  snapshot.TotalBytesReadFromStorage(),
	}, tx.Err, nil
}

func (d *RemoteDebugger) RunScript(code []byte, arguments [][]byte) (value cadence.Value, scriptError, processError error) {
	scriptCtx := fvm.NewContextFromParent(d.ctx, fvm.WithBlockHeader(d.ctx.BlockHeader))
	script := fvm.Script(code).WithArguments(arguments...)
	_, _, err := d.vm.RunV2(scriptCtx, script, d.view)
	if err != nil {
		return nil, nil, err
	}
	return script.Value, script.Err, nil
}

func (d *RemoteDebugger) Close() error {
	return nil
}
