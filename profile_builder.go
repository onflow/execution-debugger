package debugger

import (
	"fmt"
	"github.com/google/pprof/profile"
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/flow-go/fvm/environment"
	fvmRuntime "github.com/onflow/flow-go/fvm/runtime"
)

type ProfileBuilder struct {
	Profile            *profile.Profile
	profileFunctionMap map[string]uint64
	lastComputation    uint64
	profileLocationMap map[string]uint64

	nextLocID uint64
	nextFunID uint64
}

func NewProfileBuilder() *ProfileBuilder {
	// https://www.polarsignals.com/blog/posts/2021/08/03/diy-pprof-profiles-using-go/
	p := &profile.Profile{
		Function: []*profile.Function{},
		Location: []*profile.Location{},
	}
	p.SampleType = []*profile.ValueType{{
		Type: "execution effort",
		Unit: "effort",
	}}

	return &ProfileBuilder{
		Profile:            p,
		profileFunctionMap: make(map[string]uint64),
		profileLocationMap: make(map[string]uint64),
	}
}

func (p *ProfileBuilder) Close() error {
	/*
		filename := p.directory + "/profile.pb.gz"
		err := os.MkdirAll(filepath.Dir(filename), os.ModePerm)
		if err != nil {
			return err
		}

		f, err := os.Create(filename)
		if err != nil {
			return err
		}
		defer func() {
			err := f.Close()
			if err != nil {
				// log
			}
		}()

		// Write the profile to the file.
		err = p.Profile.Write(f)
		if err != nil {
			return err
		}
	*/
	return nil
}

func (p *ProfileBuilder) OnStatement(fvmEnv fvmRuntime.Environment, inter *interpreter.Interpreter, statement ast.Statement) {
	stack := inter.CallStack()
	if len(stack) == 0 {
		// what now?
		return
	}

	newComputation := fvmEnv.(environment.Environment).ComputationUsed()
	computation := newComputation - p.lastComputation
	p.lastComputation = newComputation

	fmt.Println("---------- stack ---------- ")
	fmt.Println("location", inter.Location)
	fmt.Println("program", inter.Program)
	fmt.Println("statement", statement.String())
	fmt.Println(inter.CallStack())
	fmt.Println("---------- stack ---------- ")

	locationIds := make([]uint64, 0, len(stack))

	// var lastFrame interpreter.Invocation
	for _, frame := range stack {
		// lastFrame = frame
		fn := p.toFunction(inter, frame)
		fnIndex, ok := p.profileFunctionMap[p.fnID(fn)]
		if !ok {
			p.Profile.Function = append(p.Profile.Function, fn)
			p.Profile.Location = append(p.Profile.Location,
				&profile.Location{
					ID:      p.nextLocID + 1,
					Address: p.nextLocID + 1,
					Line: []profile.Line{
						{
							Function: fn,
							Line:     fn.StartLine,
						},
					},
				},
			)
			fnIndex = p.nextFunID
			p.profileFunctionMap[p.fnID(fn)] = p.nextFunID
			p.profileLocationMap[p.fnID(fn)] = p.nextLocID
			p.nextFunID++
			p.nextLocID++
		}
		locationIds = append(locationIds, fnIndex)
	}

	locations := make([]*profile.Location, 0, len(locationIds))
	// revers iterate locations
	for i := len(locationIds) - 1; i >= 0; i-- {
		locations = append(locations, p.Profile.Location[locationIds[i]])
	}

	p.Profile.Sample = append(p.Profile.Sample, &profile.Sample{
		Location: locations,
		Value:    []int64{int64(computation)},
	})
}

func (p *ProfileBuilder) fnID(fn *profile.Function) string {
	return fn.Filename + "_" + fn.Name
}

func (p *ProfileBuilder) toFunction(inter *interpreter.Interpreter, frame interpreter.Invocation) *profile.Function {
	filename := frame.Self.StaticType(inter).String()
	name := ""
	line := int64(0)

	if frame.LocationRange.HasPosition != nil {
		switch frame.LocationRange.HasPosition.(type) {
		case *ast.InvocationExpression:
			expression := frame.LocationRange.HasPosition.(*ast.InvocationExpression)
			line = int64(expression.InvokedExpression.StartPosition().Line)

			switch expression.InvokedExpression.(type) {
			case *ast.MemberExpression:
				me := expression.InvokedExpression.(*ast.MemberExpression)
				name = me.Identifier.String()
			case *ast.IdentifierExpression:
				ie := expression.InvokedExpression.(*ast.IdentifierExpression)
				name = ie.Identifier.String()
			default:
				panic("")
			}
		default:
			panic("")
		}
	}

	return &profile.Function{
		ID:         p.nextFunID + 1,
		Name:       name,
		SystemName: name,
		Filename:   filename,
		StartLine:  line,
	}
}
