// Package ale embeds Ale as toe's command scripting language
package ale

import (
	"fmt"

	"github.com/kode4food/ale"
	"github.com/kode4food/ale/core/bootstrap"
	"github.com/kode4food/ale/core/builtin"
	"github.com/kode4food/ale/data"
	"github.com/kode4food/ale/env"
	"github.com/kode4food/ale/eval"
	"github.com/kode4food/ale/macro"

	"github.com/kode4food/toe/internal/i18n"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/view"
)

type (
	// Runtime evaluates Ale bindings against toe's command registry
	Runtime struct {
		editor      *view.Editor
		keymaps     *command.Keymaps
		environment *env.Environment
	}

	bindingOptions struct {
		action data.Procedure
		when   data.Procedure
		modes  []view.Mode
		keys   []string
		doc    string
	}

	cmdResult struct {
		result command.Result
	}
)

const (
	optMode  = "mode"
	optModes = "modes"
	optKey   = "key"
	optKeys  = "keys"
	optDoc   = "doc"
	optWhen  = "when"

	modeNormal   = "normal"
	modeInsert   = "insert"
	modeSelect   = "select"
	modeTerminal = "terminal"
	modeImage    = "image"
)

var (
	errCommandArgument    = i18n.NewError(i18n.ErrorAleCommandArguments)
	errCommandUnavailable = i18n.NewError(
		i18n.ErrorAleCommandUnavailable,
	)
)

// NewRuntime returns an ale interpreter wired to the editor and keymaps
func NewRuntime(e *view.Editor, km *command.Keymaps) (*Runtime, error) {
	r := &Runtime{
		editor:      e,
		keymaps:     km,
		environment: env.NewEnvironment(),
	}

	bootstrap.Into(r.environment)
	ns, err := r.environment.NewQualified("toe")
	if err != nil {
		return nil, err
	}
	if err := env.BindPublic(
		ns, "bind", macro.Call(bindMacro),
	); err != nil {
		return nil, err
	}
	if err := env.BindPublic(
		ns, "bind*", data.MakeProcedure(r.bind),
	); err != nil {
		return nil, err
	}
	for _, cmd := range km.CommandsIn(command.AllModes) {
		if len(cmd.Aliases) == 0 {
			continue
		}
		if err := env.BindPublic(
			ns, data.Local(cmd.Aliases[0]), r.wrapCommand(cmd),
		); err != nil {
			return nil, err
		}
	}
	return r, nil
}

// Eval evaluates Ale configuration source
func (r *Runtime) Eval(src string) (err error) {
	defer recoverError(&err)
	_, err = eval.String(r.environment.GetAnonymous(), data.String(src))
	return err
}

// Equal reports whether other is the same command result
func (r *cmdResult) Equal(other ale.Value) bool {
	return r == other
}

func (r *Runtime) bind(args ...ale.Value) ale.Value {
	opts, err := parseBinding(args)
	if err != nil {
		panic(err)
	}
	arguments := data.ToString(data.Vector(args))
	seqs := make([][]command.KeyEvent, len(opts.keys))
	for i, key := range opts.keys {
		seq, err := command.ParseKeySequence(key)
		if err != nil {
			panic(aleError(aleErrorArgs{
				kind:      i18n.ErrorInvalidKey,
				reason:    key,
				arguments: arguments,
			}))
		}
		seqs[i] = seq
	}
	bind := command.BindActionArgs{
		Modes:  opts.modes,
		Action: r.action(opts.action),
		Label:  opts.doc,
		Seqs:   seqs,
	}
	if opts.when != nil {
		bind.When = r.availability(opts.when)
	}
	if err := r.keymaps.BindResultAction(bind); err != nil {
		panic(err)
	}
	return data.Null
}

func (r *Runtime) wrapCommand(cmd *command.Command) data.Procedure {
	return data.MakeProcedure(func(values ...ale.Value) ale.Value {
		mode := r.editor.Mode()
		if cmd.Modes&mode == 0 {
			panic(errCommandUnavailable.WithVars(i18n.Vars{
				"command": cmd.Name,
				"mode":    mode.String(),
			}))
		}
		args := command.NewArgs(cmd.Signature, true)
		for _, value := range values {
			s, ok := value.(data.String)
			if !ok {
				panic(errCommandArgument)
			}
			if err := args.Push(string(s)); err != nil {
				panic(err)
			}
		}
		if err := args.Finish(); err != nil {
			panic(err)
		}
		res := cmd.Run(r.editor, args)
		if res.Error != nil {
			panic(res.Error)
		}
		return &cmdResult{result: res}
	})
}

func (r *Runtime) action(proc data.Procedure) command.KeyResultAction {
	return func(*view.Editor) (result command.Result) {
		defer func() {
			if rec := recover(); rec != nil {
				result = command.Result{Error: asError(rec)}
			}
		}()
		ctx := buildContext(r.editor)
		if value, ok := proc.Call(ctx).(*cmdResult); ok {
			return value.result
		}
		return command.Result{}
	}
}

func (r *Runtime) availability(pred data.Procedure) func(*view.Editor) bool {
	return func(e *view.Editor) (ok bool) {
		defer func() {
			if recover() != nil {
				ok = false
			}
		}()
		return pred.Call(buildContext(e)) != data.False
	}
}

func bindMacro(_ env.Namespace, args ...ale.Value) ale.Value {
	split := 0
	for split+1 < len(args) {
		if _, ok := args[split].(data.Keyword); !ok {
			break
		}
		split += 2
	}
	body := args[split:]
	if len(body) == 0 {
		panic(bindingError(
			i18n.ErrorAleBindingAction,
			data.ToString(data.Vector(args)), nil,
		))
	}
	call := []ale.Value{data.NewQualifiedSymbol("bind*", "toe")}
	for i := 0; i < split; i += 2 {
		key, value := args[i], args[i+1]
		if k, ok := key.(data.Keyword); ok && k == optWhen {
			value = ctxLambda(value)
		}
		call = append(call, key, value)
	}
	return data.NewList(append(call, ctxLambda(body...))...)
}

func ctxLambda(body ...ale.Value) ale.Value {
	return data.NewList(append([]ale.Value{
		data.MustParseSymbol("lambda"),
		data.NewList(data.MustParseSymbol("ctx")),
	}, body...)...)
}

func bindingValues(value ale.Value) data.Vector {
	if values, ok := value.(data.Vector); ok {
		return values
	}
	return data.Vector{value}
}

func bindingMode(value ale.Value, arguments string) (view.Mode, error) {
	var mode string
	switch value := value.(type) {
	case data.Keyword:
		mode = string(value)
	case data.String:
		mode = string(value)
	default:
		return 0, bindingError(
			i18n.ErrorAleBindingModeType, arguments, nil,
		)
	}
	switch mode {
	case modeNormal:
		return view.ModeNormal, nil
	case modeInsert:
		return view.ModeInsert, nil
	case modeSelect:
		return view.ModeSelect, nil
	case modeTerminal:
		return view.ModeTerminal, nil
	case modeImage:
		return view.ModeImage, nil
	default:
		return 0, bindingError(
			i18n.ErrorAleBindingUnknownMode, arguments,
			i18n.Vars{"mode": mode},
		)
	}
}

func parseBinding(args []ale.Value) (*bindingOptions, error) {
	arguments := data.ToString(data.Vector(args))
	if len(args) < 5 || len(args)%2 == 0 {
		return nil, bindingError(
			i18n.ErrorAleBindingPairs, arguments, nil,
		)
	}
	actionValue := args[len(args)-1]
	if !isProcedure(actionValue) {
		return nil, bindingError(
			i18n.ErrorAleBindingAction, arguments, nil,
		)
	}
	action := actionValue.(data.Procedure)
	opts := &bindingOptions{action: action}
	seen := map[data.Keyword]bool{}
	for i := 0; i < len(args)-1; i += 2 {
		key, ok := args[i].(data.Keyword)
		if !ok {
			return nil, bindingError(
				i18n.ErrorAleBindingOptionNames, arguments, nil,
			)
		}
		switch key {
		case optMode, optModes:
			key = optModes
		case optKey, optKeys:
			key = optKeys
		}
		if seen[key] {
			return nil, bindingError(
				i18n.ErrorAleBindingDuplicate, arguments,
				i18n.Vars{"option": key},
			)
		}
		seen[key] = true
		value := args[i+1]
		switch key {
		case optModes:
			for _, value := range bindingValues(value) {
				mode, err := bindingMode(value, arguments)
				if err != nil {
					return nil, err
				}
				opts.modes = append(opts.modes, mode)
			}
		case optKeys:
			for _, value := range bindingValues(value) {
				key, ok := value.(data.String)
				if !ok {
					return nil, bindingError(
						i18n.ErrorAleBindingKeyTypes, arguments, nil,
					)
				}
				opts.keys = append(opts.keys, string(key))
			}
		case optDoc:
			doc, ok := value.(data.String)
			if !ok {
				return nil, bindingError(
					i18n.ErrorAleBindingDocType, arguments, nil,
				)
			}
			opts.doc = string(doc)
		case optWhen:
			if !isProcedure(value) {
				return nil, bindingError(
					i18n.ErrorAleBindingWhenType, arguments, nil,
				)
			}
			opts.when = value.(data.Procedure)
		default:
			return nil, bindingError(
				i18n.ErrorAleBindingUnknown, arguments,
				i18n.Vars{"option": key},
			)
		}
	}
	if !seen[optModes] {
		return nil, bindingError(
			i18n.ErrorAleBindingModesMissing, arguments, nil,
		)
	}
	if len(opts.modes) == 0 {
		return nil, bindingError(
			i18n.ErrorAleBindingModesEmpty, arguments, nil,
		)
	}
	if !seen[optKeys] {
		return nil, bindingError(
			i18n.ErrorAleBindingKeysMissing, arguments, nil,
		)
	}
	if len(opts.keys) == 0 {
		return nil, bindingError(
			i18n.ErrorAleBindingKeysEmpty, arguments, nil,
		)
	}
	return opts, nil
}

func isProcedure(value ale.Value) bool {
	pred := builtin.IsA.Call(builtin.ProcedureKey).(data.Procedure)
	return pred.Call(value) != data.False
}

func recoverError(err *error) {
	if rec := recover(); rec != nil {
		*err = asError(rec)
	}
}

func bindingError(
	key i18n.Key, arguments string, vars i18n.Vars,
) error {
	if vars == nil {
		vars = i18n.Vars{}
	}
	return aleError(aleErrorArgs{
		kind:      i18n.ErrorInvalidBinding,
		reason:    i18n.Text(key, vars),
		arguments: arguments,
	})
}

type aleErrorArgs struct {
	kind      i18n.Key
	reason    string
	arguments string
}

func aleError(args aleErrorArgs) error {
	return i18n.NewError(i18n.ErrorAleContext).WithVars(i18n.Vars{
		"arguments": args.arguments,
		"kind":      i18n.Text(args.kind),
		"reason":    args.reason,
	})
}

func asError(value any) error {
	if err, ok := value.(error); ok {
		return err
	}
	return fmt.Errorf("%v", value)
}
