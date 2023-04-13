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

type statementHandlerFunc func(fvmEnv fvmRuntime.Environment, inter *interpreter.Interpreter, statement ast.Statement)

type RemoteDebugger struct {
	vm   *fvm.VirtualMachine
	ctx  fvm.Context
	view state.View
}

func NewRemoteDebugger(
	view *debugger.RemoteView,
	chain flow.Chain,
	logger zerolog.Logger,
	statementHandlers []statementHandlerFunc,
) *RemoteDebugger {
	vm := fvm.NewVirtualMachine()

	// no signature processor here
	// TODO Maybe we add fee-deduction step as well
	ctx := fvm.NewContext(
		fvm.WithLogger(logger),
		fvm.WithChain(chain),
		fvm.WithTransactionProcessors(fvm.NewTransactionInvoker()),
		fvm.WithReusableCadenceRuntimePool(fvmRuntime.NewReusableCadenceRuntimePool(
			1,
			fvmRuntime.ReusableCadenceRuntimePoolConfig{
				OnCadenceStatement: func(fvmEnv fvmRuntime.Environment, inter *interpreter.Interpreter, statement ast.Statement) {
					for _, handler := range statementHandlers {
						handler(fvmEnv, inter, statement) // change to interface
					}
				},
			},
		)),
	)

	return &RemoteDebugger{
		ctx:  ctx,
		vm:   vm,
		view: view,
	}
}

// RunTransaction runs the transaction given the latest sealed block data
func (d *RemoteDebugger) RunTransaction(txBody *flow.TransactionBody) (txErr, processError error) {
	blockCtx := fvm.NewContextFromParent(d.ctx, fvm.WithBlockHeader(d.ctx.BlockHeader))
	tx := fvm.Transaction(txBody, 0)
	err := d.vm.Run(blockCtx, tx, d.view)
	if err != nil {
		return nil, err
	}
	return tx.Err, nil
}

func (d *RemoteDebugger) RunScript(code []byte, arguments [][]byte) (value cadence.Value, scriptError, processError error) {
	scriptCtx := fvm.NewContextFromParent(d.ctx, fvm.WithBlockHeader(d.ctx.BlockHeader))
	script := fvm.Script(code).WithArguments(arguments...)
	err := d.vm.Run(scriptCtx, script, d.view)
	if err != nil {
		return nil, nil, err
	}
	return script.Value, script.Err, nil
}

func (d *RemoteDebugger) Close() error {
	return nil
}
