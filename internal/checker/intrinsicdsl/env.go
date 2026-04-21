package intrinsicdsl

type Env struct {
	parent *Env
	vars   map[string]Value
}

func NewEnv() *Env {
	return &Env{vars: make(map[string]Value)}
}

func (e *Env) Child() *Env {
	return &Env{parent: e, vars: make(map[string]Value)}
}

func (e *Env) Set(name string, val Value) {
	e.vars[name] = val
}

func (e *Env) Get(name string) (Value, bool) {
	if v, ok := e.vars[name]; ok {
		return v, true
	}
	if e.parent != nil {
		return e.parent.Get(name)
	}
	return Value{}, false
}

// Update sets a variable in the nearest enclosing scope that defines it.
func (e *Env) Update(name string, val Value) bool {
	if _, ok := e.vars[name]; ok {
		e.vars[name] = val
		return true
	}
	if e.parent != nil {
		return e.parent.Update(name, val)
	}
	return false
}
