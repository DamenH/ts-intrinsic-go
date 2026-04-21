package intrinsicdsl

import (
	"errors"
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/microsoft/typescript-go/internal/collections"
	"github.com/microsoft/typescript-go/internal/jsnum"
)

var (
	ErrBudgetExceeded = errors.New("evaluation exceeded step budget")
	ErrMemoryExceeded = errors.New("evaluation exceeded memory budget")
)

type Result struct {
	Value Value
	Steps int
}

const (
	DefaultBudget       = 10000
	DefaultMemoryBudget = 1000000
)

func Run(program *Node, args []Value, budget int) (Result, error) {
	if budget <= 0 {
		budget = DefaultBudget
	}
	e := &evaluator{budget: budget, memBudget: DefaultMemoryBudget}

	copiedArgs := make([]Value, len(args))
	for i, a := range args {
		copiedArgs[i] = DeepCopy(a)
	}

	env := NewEnv()
	env.Set("NumberType", NumberType)
	env.Set("StringType", StringType)
	env.Set("BooleanType", BooleanType)
	env.Set("NeverType", NeverType)
	env.Set("UnknownType", UnknownType)
	env.Set("VoidType", VoidType)

	// Execute preamble (dependency declarations) before the main function
	for _, stmt := range program.Preamble {
		_, _, err := e.execStmt(stmt, env)
		if err != nil {
			return Result{Steps: e.steps}, err
		}
	}

	for i, name := range program.Params {
		if i < len(copiedArgs) {
			env.Set(name, copiedArgs[i])
		} else {
			env.Set(name, Undefined)
		}
	}

	val, sig, err := e.evalNode(program.Body, env)
	if err != nil {
		return Result{Steps: e.steps}, err
	}
	if sig == sigReturn {
		return Result{Value: val, Steps: e.steps}, nil
	}
	return Result{Value: val, Steps: e.steps}, nil
}

type signal int

const (
	sigNone signal = iota
	sigReturn
	sigBreak
	sigContinue
)

type evaluator struct {
	steps     int
	budget    int
	mem       int
	memBudget int
}

func (e *evaluator) step() error {
	e.steps++
	if e.steps > e.budget {
		return fmt.Errorf("%w of %d", ErrBudgetExceeded, e.budget)
	}
	return nil
}

func (e *evaluator) alloc(n int) error {
	e.mem += n
	if e.mem > e.memBudget {
		return fmt.Errorf("%w", ErrMemoryExceeded)
	}
	return nil
}

func (e *evaluator) evalNode(node *Node, env *Env) (Value, signal, error) {
	if err := e.step(); err != nil {
		return Value{}, sigNone, err
	}

	switch node.Kind {
	case NodeNumberLit:
		return NumVal(node.NumVal), sigNone, nil
	case NodeStringLit:
		return StrVal(node.StrVal), sigNone, nil
	case NodeBooleanLit:
		return BoolVal(node.BoolVal), sigNone, nil
	case NodeNullLit:
		return Null, sigNone, nil
	case NodeUndefinedLit:
		return Undefined, sigNone, nil
	case NodeIdent:
		v, ok := env.Get(node.StrVal)
		if !ok {
			return Value{}, sigNone, fmt.Errorf("undefined variable '%s'", node.StrVal)
		}
		return v, sigNone, nil

	case NodeBinary:
		return e.evalBinary(node, env)
	case NodeUnary:
		return e.evalUnary(node, env)
	case NodeTernary:
		return e.evalTernary(node, env)
	case NodePropAccess:
		return e.evalPropAccess(node, env)
	case NodeIndexAccess:
		return e.evalIndexAccess(node, env)
	case NodeCall:
		return e.evalCall(node, env)
	case NodeObjectLit:
		return e.evalObjectLit(node, env)
	case NodeArrayLit:
		return e.evalArrayLit(node, env)
	case NodeLambda:
		return e.evalLambda(node, env)
	case NodeBlock:
		return e.evalBlock(node.Stmts, env)
	}
	panic("unreachable")
}

func (e *evaluator) evalBinary(node *Node, env *Env) (Value, signal, error) {
	left, _, err := e.evalNode(node.Left, env)
	if err != nil {
		return Value{}, sigNone, err
	}

	if node.Op == "&&" {
		if !IsTruthy(left) {
			return left, sigNone, nil
		}
		return e.evalNode(node.Right, env)
	}
	if node.Op == "||" {
		if IsTruthy(left) {
			return left, sigNone, nil
		}
		return e.evalNode(node.Right, env)
	}

	right, _, err := e.evalNode(node.Right, env)
	if err != nil {
		return Value{}, sigNone, err
	}

	if node.Op == "==" || node.Op == "!=" {
		eq := ValuesEqual(left, right)
		if node.Op == "!=" {
			eq = !eq
		}
		return BoolVal(eq), sigNone, nil
	}

	if node.Op == "+" {
		if left.Kind == KindString && right.Kind == KindString {
			if err := e.alloc(len(left.Str) + len(right.Str)); err != nil {
				return Value{}, sigNone, err
			}
			return StrVal(left.Str + right.Str), sigNone, nil
		}
		if left.Kind == KindString || right.Kind == KindString {
			l, err := coerceToString(left)
			if err != nil {
				return Value{}, sigNone, err
			}
			r, err := coerceToString(right)
			if err != nil {
				return Value{}, sigNone, err
			}
			if err := e.alloc(len(l) + len(r)); err != nil {
				return Value{}, sigNone, err
			}
			return StrVal(l + r), sigNone, nil
		}
	}

	l, err := requireNum(left, node.Op)
	if err != nil {
		return Value{}, sigNone, err
	}
	r, err := requireNum(right, node.Op)
	if err != nil {
		return Value{}, sigNone, err
	}

	switch node.Op {
	case "+":
		return NumVal(l + r), sigNone, nil
	case "-":
		return NumVal(l - r), sigNone, nil
	case "*":
		return NumVal(l * r), sigNone, nil
	case "/":
		return NumVal(l / r), sigNone, nil
	case "%":
		return NumVal(math.Mod(l, r)), sigNone, nil
	case "<":
		return BoolVal(l < r), sigNone, nil
	case ">":
		return BoolVal(l > r), sigNone, nil
	case "<=":
		return BoolVal(l <= r), sigNone, nil
	case ">=":
		return BoolVal(l >= r), sigNone, nil
	}
	panic("unreachable")
}

func (e *evaluator) evalUnary(node *Node, env *Env) (Value, signal, error) {
	val, _, err := e.evalNode(node.Left, env)
	if err != nil {
		return Value{}, sigNone, err
	}
	switch node.Op {
	case "-":
		n, err := requireNum(val, "unary -")
		if err != nil {
			return Value{}, sigNone, err
		}
		return NumVal(-n), sigNone, nil
	case "!":
		return BoolVal(!IsTruthy(val)), sigNone, nil
	case "typeof":
		return StrVal(TypeofValue(val)), sigNone, nil
	case "void":
		if val.Kind == KindString && val.Str == "never" {
			return NeverType, sigNone, nil
		}
		if val.Kind == KindObject && val.Props != nil {
			if errVal, ok := val.Props.Get("error"); ok && errVal.Kind == KindString {
				return Value{Kind: KindTypeSentinel, Str: "error:" + errVal.Str}, sigNone, nil
			}
		}
		return Undefined, sigNone, nil
	}
	panic("unreachable")
}

func (e *evaluator) evalTernary(node *Node, env *Env) (Value, signal, error) {
	cond, _, err := e.evalNode(node.Cond, env)
	if err != nil {
		return Value{}, sigNone, err
	}
	if IsTruthy(cond) {
		return e.evalNode(node.Then, env)
	}
	return e.evalNode(node.Else, env)
}

func (e *evaluator) evalPropAccess(node *Node, env *Env) (Value, signal, error) {
	obj, _, err := e.evalNode(node.Left, env)
	if err != nil {
		return Value{}, sigNone, err
	}
	if obj.Kind == KindObject {
		v, ok := obj.Props.Get(node.Prop)
		if ok {
			return v, sigNone, nil
		}
		return Undefined, sigNone, nil
	}
	if obj.Kind == KindTuple && node.Prop == "length" {
		return NumVal(float64(len(obj.Elems))), sigNone, nil
	}
	if obj.Kind == KindString && node.Prop == "length" {
		return NumVal(float64(len(obj.Str))), sigNone, nil
	}
	return Value{}, sigNone, fmt.Errorf("cannot access property '%s' on %s", node.Prop, TypeofValue(obj))
}

func (e *evaluator) evalIndexAccess(node *Node, env *Env) (Value, signal, error) {
	obj, _, err := e.evalNode(node.Left, env)
	if err != nil {
		return Value{}, sigNone, err
	}
	idx, _, err := e.evalNode(node.Right, env)
	if err != nil {
		return Value{}, sigNone, err
	}

	switch obj.Kind {
	case KindTuple:
		if idx.Kind == KindNumber {
			i := int(idx.Num)
			if i >= 0 && i < len(obj.Elems) {
				return obj.Elems[i], sigNone, nil
			}
			return Undefined, sigNone, nil
		}
		if idx.Kind == KindString && idx.Str == "length" {
			return NumVal(float64(len(obj.Elems))), sigNone, nil
		}
	case KindObject:
		if idx.Kind == KindString {
			v, ok := obj.Props.Get(idx.Str)
			if ok {
				return v, sigNone, nil
			}
			return Undefined, sigNone, nil
		}
	case KindString:
		if idx.Kind == KindNumber {
			i := int(idx.Num)
			if i >= 0 && i < len(obj.Str) {
				return StrVal(string(obj.Str[i])), sigNone, nil
			}
			return Undefined, sigNone, nil
		}
	}
	return Undefined, sigNone, nil
}

func (e *evaluator) evalArgs(nodes []*Node, env *Env) ([]Value, error) {
	args := make([]Value, len(nodes))
	for i, a := range nodes {
		v, _, err := e.evalNode(a, env)
		if err != nil {
			return nil, err
		}
		args[i] = v
	}
	return args, nil
}

func (e *evaluator) evalCall(node *Node, env *Env) (Value, signal, error) {
	if node.Callee.Kind == NodeIdent {
		name := node.Callee.StrVal
		args, err := e.evalArgs(node.Args, env)
		if err != nil {
			return Value{}, sigNone, err
		}

		fn, ok := env.Get(name)
		if !ok {
			return Value{}, sigNone, fmt.Errorf("undefined function '%s'", name)
		}
		if fn.Kind != KindFunction {
			return Value{}, sigNone, fmt.Errorf("'%s' is not a function", name)
		}
		return e.callClosure(fn.Fn, args)
	}

	// Method calls: receiver.method(args)
	if node.Callee.Kind == NodePropAccess {
		// Static namespace calls: Object.keys(x), Math.abs(x)
		if node.Callee.Left.Kind == NodeIdent {
			ns := node.Callee.Left.StrVal
			method := node.Callee.Prop
			if ns == "Object" || ns == "Math" {
				args, err := e.evalArgs(node.Args, env)
				if err != nil {
					return Value{}, sigNone, err
				}
				if result, ok, err := e.callStatic(ns, method, args); ok {
					return result, sigNone, err
				}
			}
		}

		receiver, _, err := e.evalNode(node.Callee.Left, env)
		if err != nil {
			return Value{}, sigNone, err
		}
		method := node.Callee.Prop
		args, err := e.evalArgs(node.Args, env)
		if err != nil {
			return Value{}, sigNone, err
		}
		if result, ok, err := e.callMethod(receiver, method, args); ok {
			return result, sigNone, err
		}
		// Fall through: maybe the prop access yields a callable value (e.g. object with function property)
		if receiver.Kind == KindObject {
			callee, ok := receiver.Props.Get(method)
			if ok && callee.Kind == KindFunction {
				return e.callClosure(callee.Fn, args)
			}
		}
		return Value{}, sigNone, fmt.Errorf("'%s' is not a method on %s", method, TypeofValue(receiver))
	}

	callee, _, err := e.evalNode(node.Callee, env)
	if err != nil {
		return Value{}, sigNone, err
	}
	args, err := e.evalArgs(node.Args, env)
	if err != nil {
		return Value{}, sigNone, err
	}

	if callee.Kind != KindFunction {
		return Value{}, sigNone, fmt.Errorf("cannot call non-function value of type %s", TypeofValue(callee))
	}
	return e.callClosure(callee.Fn, args)
}

// callMethod dispatches method calls on primitive types (strings, tuples, objects).
func (e *evaluator) callMethod(receiver Value, method string, args []Value) (Value, bool, error) {
	switch receiver.Kind {
	case KindString:
		s := receiver.Str
		switch method {
		case "toUpperCase":
			return StrVal(strings.ToUpper(s)), true, nil
		case "toLowerCase":
			return StrVal(strings.ToLower(s)), true, nil
		case "trim":
			return StrVal(strings.TrimSpace(s)), true, nil
		case "split":
			if err := requireArgCount(args, 1, "split"); err != nil {
				return Value{}, true, err
			}
			sep, err := requireStr(args[0], "split")
			if err != nil {
				return Value{}, true, err
			}
			parts := strings.Split(s, sep)
			elems := make([]Value, len(parts))
			for i, p := range parts {
				elems[i] = StrVal(p)
			}
			return TupleVal(elems...), true, nil
		case "includes":
			if err := requireArgCount(args, 1, "includes"); err != nil {
				return Value{}, true, err
			}
			needle, err := requireStr(args[0], "includes")
			if err != nil {
				return Value{}, true, err
			}
			return BoolVal(strings.Contains(s, needle)), true, nil
		case "indexOf":
			if err := requireArgCount(args, 1, "indexOf"); err != nil {
				return Value{}, true, err
			}
			needle, err := requireStr(args[0], "indexOf")
			if err != nil {
				return Value{}, true, err
			}
			return NumVal(float64(strings.Index(s, needle))), true, nil
		case "startsWith":
			if err := requireArgCount(args, 1, "startsWith"); err != nil {
				return Value{}, true, err
			}
			prefix, err := requireStr(args[0], "startsWith")
			if err != nil {
				return Value{}, true, err
			}
			return BoolVal(strings.HasPrefix(s, prefix)), true, nil
		case "endsWith":
			if err := requireArgCount(args, 1, "endsWith"); err != nil {
				return Value{}, true, err
			}
			suffix, err := requireStr(args[0], "endsWith")
			if err != nil {
				return Value{}, true, err
			}
			return BoolVal(strings.HasSuffix(s, suffix)), true, nil
		case "slice":
			if err := requireArgCount(args, 1, "slice"); err != nil {
				return Value{}, true, err
			}
			return e.callBuiltinSlice(receiver, args)
		case "replace":
			if err := requireArgCount(args, 2, "replace"); err != nil {
				return Value{}, true, err
			}
			old, err := requireStr(args[0], "replace")
			if err != nil {
				return Value{}, true, err
			}
			newStr, err := requireStr(args[1], "replace")
			if err != nil {
				return Value{}, true, err
			}
			return StrVal(strings.Replace(s, old, newStr, 1)), true, nil
		case "charAt":
			if err := requireArgCount(args, 1, "charAt"); err != nil {
				return Value{}, true, err
			}
			n, err := requireNum(args[0], "charAt")
			if err != nil {
				return Value{}, true, err
			}
			i := int(n)
			if i >= 0 && i < len(s) {
				return StrVal(string(s[i])), true, nil
			}
			return StrVal(""), true, nil
		}

	case KindTuple:
		switch method {
		case "map":
			if err := requireArgCount(args, 1, "map"); err != nil {
				return Value{}, true, err
			}
			return e.callBuiltinMap(receiver, args)
		case "filter":
			if err := requireArgCount(args, 1, "filter"); err != nil {
				return Value{}, true, err
			}
			return e.callBuiltinFilter(receiver, args)
		case "reduce":
			if err := requireArgCount(args, 2, "reduce"); err != nil {
				return Value{}, true, err
			}
			fn := args[0]
			acc := args[1]
			for _, el := range receiver.Elems {
				if err := e.step(); err != nil {
					return Value{}, true, err
				}
				v, err := e.callValue(fn, []Value{acc, el})
				if err != nil {
					return Value{}, true, err
				}
				acc = v
			}
			return acc, true, nil
		case "find":
			if err := requireArgCount(args, 1, "find"); err != nil {
				return Value{}, true, err
			}
			fn := args[0]
			for _, el := range receiver.Elems {
				if err := e.step(); err != nil {
					return Value{}, true, err
				}
				v, err := e.callValue(fn, []Value{el})
				if err != nil {
					return Value{}, true, err
				}
				if IsTruthy(v) {
					return el, true, nil
				}
			}
			return Undefined, true, nil
		case "some":
			if err := requireArgCount(args, 1, "some"); err != nil {
				return Value{}, true, err
			}
			fn := args[0]
			for _, el := range receiver.Elems {
				if err := e.step(); err != nil {
					return Value{}, true, err
				}
				v, err := e.callValue(fn, []Value{el})
				if err != nil {
					return Value{}, true, err
				}
				if IsTruthy(v) {
					return BoolVal(true), true, nil
				}
			}
			return BoolVal(false), true, nil
		case "every":
			if err := requireArgCount(args, 1, "every"); err != nil {
				return Value{}, true, err
			}
			fn := args[0]
			for _, el := range receiver.Elems {
				if err := e.step(); err != nil {
					return Value{}, true, err
				}
				v, err := e.callValue(fn, []Value{el})
				if err != nil {
					return Value{}, true, err
				}
				if !IsTruthy(v) {
					return BoolVal(false), true, nil
				}
			}
			return BoolVal(true), true, nil
		case "flat":
			var result []Value
			for _, el := range receiver.Elems {
				if el.Kind == KindTuple {
					result = append(result, el.Elems...)
				} else {
					result = append(result, el)
				}
			}
			return TupleVal(result...), true, nil
		case "concat":
			if err := requireArgCount(args, 1, "concat"); err != nil {
				return Value{}, true, err
			}
			b, err := requireArr(args[0], "concat")
			if err != nil {
				return Value{}, true, err
			}
			result := make([]Value, 0, len(receiver.Elems)+len(b))
			result = append(result, receiver.Elems...)
			result = append(result, b...)
			return TupleVal(result...), true, nil
		case "reverse":
			result := make([]Value, len(receiver.Elems))
			copy(result, receiver.Elems)
			slices.Reverse(result)
			return TupleVal(result...), true, nil
		case "sort":
			result := make([]Value, len(receiver.Elems))
			copy(result, receiver.Elems)
			SortValues(result)
			return TupleVal(result...), true, nil
		case "includes":
			if err := requireArgCount(args, 1, "includes"); err != nil {
				return Value{}, true, err
			}
			for _, el := range receiver.Elems {
				if ValuesEqual(el, args[0]) {
					return BoolVal(true), true, nil
				}
			}
			return BoolVal(false), true, nil
		case "indexOf":
			if err := requireArgCount(args, 1, "indexOf"); err != nil {
				return Value{}, true, err
			}
			for i, el := range receiver.Elems {
				if ValuesEqual(el, args[0]) {
					return NumVal(float64(i)), true, nil
				}
			}
			return NumVal(-1), true, nil
		case "slice":
			if err := requireArgCount(args, 1, "slice"); err != nil {
				return Value{}, true, err
			}
			return e.callBuiltinSlice(receiver, args)
		case "join":
			if err := requireArgCount(args, 1, "join"); err != nil {
				return Value{}, true, err
			}
			sep, err := requireStr(args[0], "join")
			if err != nil {
				return Value{}, true, err
			}
			strs := make([]string, len(receiver.Elems))
			for i, v := range receiver.Elems {
				s, err := requireStr(v, "join")
				if err != nil {
					return Value{}, true, err
				}
				strs[i] = s
			}
			return StrVal(strings.Join(strs, sep)), true, nil
		}
	}

	return Value{}, false, nil
}

func (e *evaluator) callStatic(ns string, method string, args []Value) (Value, bool, error) {
	switch ns {
	case "Object":
		switch method {
		case "keys":
			obj, err := requireObj(args[0], "Object.keys")
			if err != nil {
				return Value{}, true, err
			}
			var elems []Value
			for k := range obj.Keys() {
				elems = append(elems, StrVal(k))
			}
			return TupleVal(elems...), true, nil
		case "values":
			obj, err := requireObj(args[0], "Object.values")
			if err != nil {
				return Value{}, true, err
			}
			var elems []Value
			for _, v := range obj.Entries() {
				elems = append(elems, v)
			}
			return TupleVal(elems...), true, nil
		case "entries":
			obj, err := requireObj(args[0], "Object.entries")
			if err != nil {
				return Value{}, true, err
			}
			var elems []Value
			for k, v := range obj.Entries() {
				elems = append(elems, TupleVal(StrVal(k), v))
			}
			return TupleVal(elems...), true, nil
		case "fromEntries":
			arr, err := requireArr(args[0], "Object.fromEntries")
			if err != nil {
				return Value{}, true, err
			}
			m := NewOrderedMap()
			for _, entry := range arr {
				pair, err := requireArr(entry, "Object.fromEntries")
				if err != nil {
					return Value{}, true, err
				}
				k, err := requireStr(pair[0], "Object.fromEntries")
				if err != nil {
					return Value{}, true, err
				}
				m.Set(k, pair[1])
			}
			return ObjectVal(m), true, nil
		}
	case "Math":
		if fn, ok := numToNumBuiltins[method]; ok {
			n, err := requireNum(args[0], "Math."+method)
			if err != nil {
				return Value{}, true, err
			}
			return NumVal(fn(n)), true, nil
		}
		if fn, ok := numNumToNumBuiltins[method]; ok {
			a, err := requireNum(args[0], "Math."+method)
			if err != nil {
				return Value{}, true, err
			}
			b, err := requireNum(args[1], "Math."+method)
			if err != nil {
				return Value{}, true, err
			}
			return NumVal(fn(a, b)), true, nil
		}
	}
	return Value{}, false, nil
}

func (e *evaluator) callBuiltinSlice(receiver Value, args []Value) (Value, bool, error) {
	startN, err := requireNum(args[0], "slice")
	if err != nil {
		return Value{}, true, err
	}
	start := int(startN)
	if receiver.Kind == KindString {
		s := receiver.Str
		if start < 0 {
			start = len(s) + start
		}
		start = max(start, 0)
		start = min(start, len(s))
		end := len(s)
		if len(args) > 1 {
			endN, err := requireNum(args[1], "slice")
			if err != nil {
				return Value{}, true, err
			}
			end = int(endN)
			if end < 0 {
				end = len(s) + end
			}
			end = min(end, len(s))
		}
		if start >= end {
			return StrVal(""), true, nil
		}
		return StrVal(s[start:end]), true, nil
	}
	elems := receiver.Elems
	if start < 0 {
		start = len(elems) + start
	}
	start = max(start, 0)
	start = min(start, len(elems))
	end := len(elems)
	if len(args) > 1 {
		endN, err := requireNum(args[1], "slice")
		if err != nil {
			return Value{}, true, err
		}
		end = int(endN)
		if end < 0 {
			end = len(elems) + end
		}
		end = min(end, len(elems))
	}
	if start >= end {
		return TupleVal(), true, nil
	}
	return TupleVal(elems[start:end]...), true, nil
}

func (e *evaluator) callBuiltinMap(receiver Value, args []Value) (Value, bool, error) {
	fn := args[0]
	result := make([]Value, len(receiver.Elems))
	for i, el := range receiver.Elems {
		if err := e.step(); err != nil {
			return Value{}, true, err
		}
		v, err := e.callValue(fn, []Value{el, NumVal(float64(i))})
		if err != nil {
			return Value{}, true, err
		}
		result[i] = v
	}
	return TupleVal(result...), true, nil
}

func (e *evaluator) callBuiltinFilter(receiver Value, args []Value) (Value, bool, error) {
	fn := args[0]
	var result []Value
	for i, el := range receiver.Elems {
		if err := e.step(); err != nil {
			return Value{}, true, err
		}
		v, err := e.callValue(fn, []Value{el, NumVal(float64(i))})
		if err != nil {
			return Value{}, true, err
		}
		if IsTruthy(v) {
			result = append(result, el)
		}
	}
	return TupleVal(result...), true, nil
}

func (e *evaluator) callClosure(fn *Closure, args []Value) (Value, signal, error) {
	fnEnv := fn.Env.Child()
	for i, name := range fn.Params {
		if i < len(args) {
			fnEnv.Set(name, args[i])
		} else {
			fnEnv.Set(name, Undefined)
		}
	}
	val, _, err := e.evalNode(&fn.Body, fnEnv)
	if err != nil {
		return Value{}, sigNone, err
	}
	return val, sigNone, nil
}

func (e *evaluator) evalObjectLit(node *Node, env *Env) (Value, signal, error) {
	m := NewOrderedMap()
	for _, prop := range node.ObjProps {
		if prop.Spread {
			spread, _, err := e.evalNode(prop.Value, env)
			if err != nil {
				return Value{}, sigNone, err
			}
			if spread.Kind != KindObject {
				return Value{}, sigNone, fmt.Errorf("cannot spread non-object")
			}
			for k, v := range spread.Props.Entries() {
				m.Set(k, v)
			}
		} else {
			key, _, err := e.evalNode(prop.Key, env)
			if err != nil {
				return Value{}, sigNone, err
			}
			val, _, err := e.evalNode(prop.Value, env)
			if err != nil {
				return Value{}, sigNone, err
			}
			var keyStr string
			switch key.Kind {
			case KindString:
				keyStr = key.Str
			default:
				keyStr = Display(key)
			}
			m.Set(keyStr, val)
		}
	}
	return ObjectVal(m), sigNone, nil
}

func (e *evaluator) evalArrayLit(node *Node, env *Env) (Value, signal, error) {
	var elems []Value
	for _, elem := range node.ArrElems {
		val, _, err := e.evalNode(elem.Expr, env)
		if err != nil {
			return Value{}, sigNone, err
		}
		if elem.Spread {
			if val.Kind != KindTuple {
				return Value{}, sigNone, fmt.Errorf("cannot spread non-array value")
			}
			elems = append(elems, val.Elems...)
		} else {
			elems = append(elems, val)
		}
	}
	if err := e.alloc(len(elems)); err != nil {
		return Value{}, sigNone, err
	}
	return TupleVal(elems...), sigNone, nil
}

func (e *evaluator) evalLambda(node *Node, env *Env) (Value, signal, error) {
	return Value{
		Kind: KindFunction,
		Fn: &Closure{
			Params: node.Params,
			Body:   *node.Body,
			Env:    env,
		},
	}, sigNone, nil
}

func (e *evaluator) evalBlock(stmts []Stmt, env *Env) (Value, signal, error) {
	return e.execBlock(stmts, env.Child())
}

func (e *evaluator) execStmt(stmt Stmt, env *Env) (Value, signal, error) {
	if err := e.step(); err != nil {
		return Value{}, sigNone, err
	}

	switch stmt.Kind {
	case StmtLet:
		val, _, err := e.evalNode(stmt.Init, env)
		if err != nil {
			return Value{}, sigNone, err
		}
		env.Set(stmt.Name, val)
		return val, sigNone, nil

	case StmtDestructureLet:
		val, _, err := e.evalNode(stmt.Init, env)
		if err != nil {
			return Value{}, sigNone, err
		}
		if val.Kind != KindTuple {
			return Value{}, sigNone, fmt.Errorf("cannot destructure non-array value")
		}
		for i, name := range stmt.Names {
			if i < len(val.Elems) {
				env.Set(name, val.Elems[i])
			} else {
				env.Set(name, Undefined)
			}
		}
		if stmt.Rest != "" {
			start := len(stmt.Names)
			if start < len(val.Elems) {
				env.Set(stmt.Rest, TupleVal(val.Elems[start:]...))
			} else {
				env.Set(stmt.Rest, TupleVal())
			}
		}
		return val, sigNone, nil

	case StmtAssign:
		val, _, err := e.evalNode(stmt.Value, env)
		if err != nil {
			return Value{}, sigNone, err
		}
		if !env.Update(stmt.Name, val) {
			return Value{}, sigNone, fmt.Errorf("cannot assign to undeclared variable '%s'", stmt.Name)
		}
		return val, sigNone, nil

	case StmtIndexAssign:
		obj, _, err := e.evalNode(stmt.Object, env)
		if err != nil {
			return Value{}, sigNone, err
		}
		idx, _, err := e.evalNode(stmt.Index, env)
		if err != nil {
			return Value{}, sigNone, err
		}
		val, _, err := e.evalNode(stmt.Value, env)
		if err != nil {
			return Value{}, sigNone, err
		}
		if obj.Kind == KindObject && idx.Kind == KindString {
			obj.Props.Set(idx.Str, val)
		} else if obj.Kind == KindTuple && idx.Kind == KindNumber {
			i := int(idx.Num)
			if i >= 0 && i < len(obj.Elems) {
				obj.Elems[i] = val
			}
		}
		return val, sigNone, nil

	case StmtIf:
		return e.execIf(stmt, env)

	case StmtForOf:
		iter, _, err := e.evalNode(stmt.Iter, env)
		if err != nil {
			return Value{}, sigNone, err
		}
		if iter.Kind != KindTuple {
			return Value{}, sigNone, fmt.Errorf("for-of requires an array")
		}
		var lastVal Value
		for _, elem := range iter.Elems {
			env.Set(stmt.Name, elem)
			val, sig, err := e.execBlock(stmt.Then, env)
			if err != nil {
				return Value{}, sigNone, err
			}
			if sig == sigReturn {
				return val, sigReturn, nil
			}
			if sig == sigBreak {
				break
			}
			lastVal = val
		}
		return lastVal, sigNone, nil

	case StmtWhile:
		var lastVal Value
		for {
			cond, _, err := e.evalNode(stmt.Cond, env)
			if err != nil {
				return Value{}, sigNone, err
			}
			if !IsTruthy(cond) {
				break
			}
			val, sig, err := e.execBlock(stmt.Then, env)
			if err != nil {
				return Value{}, sigNone, err
			}
			if sig == sigReturn {
				return val, sigReturn, nil
			}
			if sig == sigBreak {
				break
			}
			lastVal = val
		}
		return lastVal, sigNone, nil

	case StmtBreak:
		return Value{}, sigBreak, nil
	case StmtContinue:
		return Value{}, sigContinue, nil

	case StmtReturn:
		val, _, err := e.evalNode(stmt.Value, env)
		if err != nil {
			return Value{}, sigNone, err
		}
		return val, sigReturn, nil

	case StmtExpr:
		val, _, err := e.evalNode(stmt.Value, env)
		if err != nil {
			return Value{}, sigNone, err
		}
		return val, sigNone, nil
	}

	panic("unreachable")
}

func (e *evaluator) execIf(stmt Stmt, env *Env) (Value, signal, error) {
	cond, _, err := e.evalNode(stmt.Cond, env)
	if err != nil {
		return Value{}, sigNone, err
	}
	if IsTruthy(cond) {
		return e.execBlock(stmt.Then, env)
	}
	for _, ei := range stmt.ElseIfs {
		cond, _, err := e.evalNode(ei.Cond, env)
		if err != nil {
			return Value{}, sigNone, err
		}
		if IsTruthy(cond) {
			return e.execBlock(ei.Body, env)
		}
	}
	if len(stmt.Else) > 0 {
		return e.execBlock(stmt.Else, env)
	}
	return Undefined, sigNone, nil
}

func (e *evaluator) execBlock(stmts []Stmt, env *Env) (Value, signal, error) {
	var lastVal Value
	for _, s := range stmts {
		val, sig, err := e.execStmt(s, env)
		if err != nil {
			return Value{}, sigNone, err
		}
		if sig != sigNone {
			return val, sig, nil
		}
		lastVal = val
	}
	return lastVal, sigNone, nil
}

var numToNumBuiltins = map[string]func(float64) float64{
	"abs": math.Abs, "floor": math.Floor, "ceil": math.Ceil,
	"round": math.Round, "sqrt": math.Sqrt,
}

var numNumToNumBuiltins = map[string]func(float64, float64) float64{
	"min": math.Min, "max": math.Max, "pow": math.Pow,
}

func requireArgCount(args []Value, min int, name string) error {
	if len(args) < min {
		return fmt.Errorf("%s requires at least %d argument(s), got %d", name, min, len(args))
	}
	return nil
}

func (e *evaluator) callValue(fn Value, args []Value) (Value, error) {
	if fn.Kind != KindFunction {
		return Value{}, fmt.Errorf("cannot call non-function")
	}
	val, _, err := e.callClosure(fn.Fn, args)
	return val, err
}

func requireNum(v Value, context string) (float64, error) {
	if v.Kind == KindNumber {
		return v.Num, nil
	}
	return 0, fmt.Errorf("%s requires a number, got %s", context, TypeofValue(v))
}

func requireStr(v Value, context string) (string, error) {
	if v.Kind == KindString {
		return v.Str, nil
	}
	return "", fmt.Errorf("%s requires a string, got %s", context, TypeofValue(v))
}

func requireArr(v Value, context string) ([]Value, error) {
	if v.Kind == KindTuple {
		return v.Elems, nil
	}
	return nil, fmt.Errorf("%s requires an array, got %s", context, TypeofValue(v))
}

func requireObj(v Value, context string) (*collections.OrderedMap[string, Value], error) {
	if v.Kind == KindObject {
		return v.Props, nil
	}
	return nil, fmt.Errorf("%s requires an object, got %s", context, TypeofValue(v))
}

func coerceToString(v Value) (string, error) {
	switch v.Kind {
	case KindString:
		return v.Str, nil
	case KindNumber:
		return jsnum.Number(v.Num).String(), nil
	default:
		return "", fmt.Errorf("cannot concatenate %s with string", TypeofValue(v))
	}
}
