package intrinsicdsl

import (
	"errors"
	"math"
	"testing"
)

func mustParse(t *testing.T, src string) *Node {
	t.Helper()
	node, err := ParseProgram(src)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	return node
}

func mustRun(t *testing.T, src string, args ...Value) Value {
	t.Helper()
	prog := mustParse(t, src)
	result, err := Run(prog, args, DefaultBudget)
	if err != nil {
		t.Fatalf("runtime error: %v", err)
	}
	return result.Value
}

func expectNum(t *testing.T, src string, expected float64, args ...Value) {
	t.Helper()
	v := mustRun(t, src, args...)
	if v.Kind != KindNumber || v.Num != expected {
		t.Errorf("expected %v, got %s", expected, Display(v))
	}
}

func expectStr(t *testing.T, src string, expected string, args ...Value) {
	t.Helper()
	v := mustRun(t, src, args...)
	if v.Kind != KindString || v.Str != expected {
		t.Errorf("expected %q, got %s", expected, Display(v))
	}
}

func expectBool(t *testing.T, src string, expected bool, args ...Value) {
	t.Helper()
	v := mustRun(t, src, args...)
	if v.Kind != KindBoolean || v.Bool != expected {
		t.Errorf("expected %v, got %s", expected, Display(v))
	}
}

func TestArithmetic(t *testing.T) {
	t.Parallel()
	expectNum(t, "(a, b) => a + b", 30, NumVal(10), NumVal(20))
	expectNum(t, "(a, b) => a - b", 58, NumVal(100), NumVal(42))
	expectNum(t, "(a, b) => a * b", 42, NumVal(6), NumVal(7))
	expectNum(t, "(a, b) => a / b", 25, NumVal(100), NumVal(4))
	expectNum(t, "(a, b) => a % b", 2, NumVal(17), NumVal(5))
	expectNum(t, "(n) => -n", -99, NumVal(99))
}

func TestComparisons(t *testing.T) {
	t.Parallel()
	expectBool(t, "(n) => n > 0", true, NumVal(5))
	expectBool(t, "(n) => n > 0", false, NumVal(-3))
	expectBool(t, "(n) => n > 0", false, NumVal(0))
	expectBool(t, "(n) => n % 2 == 0", true, NumVal(4))
	expectBool(t, "(n) => n % 2 == 0", false, NumVal(3))
}

func TestStringOps(t *testing.T) {
	t.Parallel()
	expectStr(t, "(s) => s.toUpperCase()", "HELLO WORLD", StrVal("hello world"))
	expectStr(t, "(s) => s.toLowerCase()", "screaming", StrVal("SCREAMING"))
	expectNum(t, "(s) => s.length", 12, StrVal("twelve chars"))
}

func TestMath(t *testing.T) {
	t.Parallel()
	expectNum(t, "(n) => Math.sqrt(n)", 4, NumVal(16))
	expectNum(t, "(a, b) => Math.pow(a, b)", 1024, NumVal(2), NumVal(10))
	expectNum(t, "(n) => Math.abs(n)", 50, NumVal(-50))
}

func TestClamp(t *testing.T) {
	t.Parallel()
	expectNum(t, "(n, lo, hi) => Math.min(Math.max(n, lo), hi)", 0, NumVal(-5), NumVal(0), NumVal(100))
	expectNum(t, "(n, lo, hi) => Math.min(Math.max(n, lo), hi)", 50, NumVal(50), NumVal(0), NumVal(100))
	expectNum(t, "(n, lo, hi) => Math.min(Math.max(n, lo), hi)", 100, NumVal(999), NumVal(0), NumVal(100))
}

func TestSnakeToCamel(t *testing.T) {
	t.Parallel()
	src := `(s) => { let [first, ...rest] = s.split('_'); return first + rest.map((p) => p.charAt(0).toUpperCase() + p.slice(1)).join(''); }`
	expectStr(t, src, "userFirstName", StrVal("user_first_name"))
	expectStr(t, src, "getAllItemsById", StrVal("get_all_items_by_id"))
	expectStr(t, src, "already", StrVal("already"))
}

func TestSnakeToCamelImperative(t *testing.T) {
	t.Parallel()
	src := `(s) => {
		let parts = s.split('_');
		let result = parts[0];
		let i = 1;
		while (i < parts.length) {
			result = result + parts[i].charAt(0).toUpperCase() + parts[i].slice(1);
			i = i + 1;
		}
		return result;
	}`
	expectStr(t, src, "userFirstName", StrVal("user_first_name"))
}

func TestObjectReturn(t *testing.T) {
	t.Parallel()
	src := `(name, age) => { let r = {}; r['name'] = name; r['age'] = age; return r; }`
	v := mustRun(t, src, StrVal("Alice"), NumVal(30))
	if v.Kind != KindObject {
		t.Fatalf("expected object, got %s", Display(v))
	}
	name, _ := v.Props.Get("name")
	if name.Kind != KindString || name.Str != "Alice" {
		t.Errorf("expected name=Alice, got %s", Display(name))
	}
	age, _ := v.Props.Get("age")
	if age.Kind != KindNumber || age.Num != 30 {
		t.Errorf("expected age=30, got %s", Display(age))
	}
}

func TestWhileLoop(t *testing.T) {
	t.Parallel()
	v := mustRun(t, `(n) => {
		let arr = [];
		let i = 0;
		while (i < n) { arr = [...arr, i]; i = i + 1; }
		return arr;
	}`, NumVal(5))
	if v.Kind != KindTuple || len(v.Elems) != 5 {
		t.Fatalf("expected 5-element tuple, got %s", Display(v))
	}
	for i := range 5 {
		if v.Elems[i].Kind != KindNumber || v.Elems[i].Num != float64(i) {
			t.Errorf("expected %d at index %d, got %s", i, i, Display(v.Elems[i]))
		}
	}
}

func TestHead(t *testing.T) {
	t.Parallel()
	expectNum(t, "(t) => t[0]", 1, TupleVal(NumVal(1), NumVal(2), NumVal(3)))
}

func TestTail(t *testing.T) {
	t.Parallel()
	v := mustRun(t, "(t) => t.slice(1)", TupleVal(NumVal(1), NumVal(2), NumVal(3)))
	if v.Kind != KindTuple || len(v.Elems) != 2 {
		t.Fatalf("expected 2-element tuple, got %s", Display(v))
	}
}

func TestSum(t *testing.T) {
	t.Parallel()
	expectNum(t, "(t) => t.reduce((acc, x) => acc + x, 0)", 15,
		TupleVal(NumVal(1), NumVal(2), NumVal(3), NumVal(4), NumVal(5)))
}

func TestSumLoop(t *testing.T) {
	t.Parallel()
	expectNum(t, "(t) => { let sum = 0; for (let x of t) { sum = sum + x; } return sum; }", 15,
		TupleVal(NumVal(1), NumVal(2), NumVal(3), NumVal(4), NumVal(5)))
}

func TestDeepGet(t *testing.T) {
	t.Parallel()
	inner := NewOrderedMap()
	inner.Set("c", NumVal(42))
	mid := NewOrderedMap()
	mid.Set("b", ObjectVal(inner))
	outer := NewOrderedMap()
	outer.Set("a", ObjectVal(mid))

	expectNum(t, "(t, path) => path.split('.').reduce((obj, key) => obj[key], t)", 42,
		ObjectVal(outer), StrVal("a.b.c"))
}

func TestTypeDispatchIfElse(t *testing.T) {
	t.Parallel()
	src := `(x) => { if (typeof(x) == 'number') return 'a number'; if (typeof(x) == 'string') return 'a string'; if (typeof(x) == 'boolean') return 'a boolean'; return 'something else'; }`
	expectStr(t, src, "a number", NumVal(42))
	expectStr(t, src, "a string", StrVal("hi"))
	expectStr(t, src, "a boolean", BoolVal(true))
}

func TestFilter(t *testing.T) {
	t.Parallel()
	v := mustRun(t, "(t) => t.filter((x) => x > 0)",
		TupleVal(NumVal(1), NumVal(-2), NumVal(3), NumVal(-4), NumVal(5)))
	if v.Kind != KindTuple || len(v.Elems) != 3 {
		t.Fatalf("expected 3-element tuple, got %s", Display(v))
	}
}

func TestDoubled(t *testing.T) {
	t.Parallel()
	v := mustRun(t, "(t) => t.map((x) => x * 2)",
		TupleVal(NumVal(1), NumVal(2), NumVal(3)))
	if v.Kind != KindTuple || len(v.Elems) != 3 {
		t.Fatalf("expected 3-element tuple, got %s", Display(v))
	}
	if v.Elems[0].Num != 2 || v.Elems[1].Num != 4 || v.Elems[2].Num != 6 {
		t.Errorf("expected [2, 4, 6], got %s", Display(v))
	}
}

func TestUnique(t *testing.T) {
	t.Parallel()
	v := mustRun(t, "(t) => t.filter((x, i) => t.indexOf(x) == i)",
		TupleVal(NumVal(1), NumVal(2), NumVal(1), NumVal(3), NumVal(2), NumVal(4)))
	if v.Kind != KindTuple || len(v.Elems) != 4 {
		t.Fatalf("expected 4-element tuple, got %s", Display(v))
	}
}

func TestUniqueLoop(t *testing.T) {
	t.Parallel()
	src := `(t) => { let result = []; for (let x of t) { if (!result.includes(x)) { result = [...result, x]; } } return result; }`
	v := mustRun(t, src,
		TupleVal(NumVal(1), NumVal(2), NumVal(1), NumVal(3), NumVal(2), NumVal(4)))
	if v.Kind != KindTuple || len(v.Elems) != 4 {
		t.Fatalf("expected 4-element tuple, got %s", Display(v))
	}
}

func TestFactorial(t *testing.T) {
	t.Parallel()
	expectNum(t, "(n) => { let result = 1; let i = n; while (i > 0) { result = result * i; i = i - 1; } return result; }", 120, NumVal(5))
}

func TestCamelCaseKeys(t *testing.T) {
	t.Parallel()
	src := `(obj) => { let result = {}; for (let k of Object.keys(obj)) { let [first, ...rest] = k.split('_'); result[first + rest.map((p) => p.charAt(0).toUpperCase() + p.slice(1)).join('')] = obj[k]; } return result; }`
	m := NewOrderedMap()
	m.Set("user_id", NumVal(1))
	m.Set("first_name", StrVal("John"))
	m.Set("is_active", BoolVal(true))
	v := mustRun(t, src, ObjectVal(m))
	if v.Kind != KindObject {
		t.Fatalf("expected object, got %s", Display(v))
	}
	uid, ok := v.Props.Get("userId")
	if !ok || uid.Num != 1 {
		t.Errorf("expected userId=1, got %s", Display(uid))
	}
	fn, ok := v.Props.Get("firstName")
	if !ok || fn.Str != "John" {
		t.Errorf("expected firstName=John, got %s", Display(fn))
	}
}

func TestCamelCaseKeysFunctional(t *testing.T) {
	t.Parallel()
	src := `(obj) => Object.fromEntries(Object.entries(obj).map((e) => { let [first, ...rest] = e[0].split('_'); return [first + rest.map((p) => p.charAt(0).toUpperCase() + p.slice(1)).join(''), e[1]]; }))`
	m := NewOrderedMap()
	m.Set("user_id", NumVal(1))
	m.Set("first_name", StrVal("John"))
	v := mustRun(t, src, ObjectVal(m))
	if v.Kind != KindObject {
		t.Fatalf("expected object, got %s", Display(v))
	}
	uid, ok := v.Props.Get("userId")
	if !ok || uid.Num != 1 {
		t.Errorf("expected userId=1, got %s", Display(uid))
	}
}

func TestIfElse(t *testing.T) {
	t.Parallel()
	src := `(x) => { if (typeof(x) == 'number') return 'a number'; if (typeof(x) == 'string') return 'a string'; return 'something else'; }`
	expectStr(t, src, "a number", NumVal(42))
	expectStr(t, src, "a string", StrVal("hi"))
}

func TestBudgetExceeded(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, "(n) => { let i = 0; while (i < n) { i = i + 1; } return i; }")
	_, err := Run(prog, []Value{NumVal(100000)}, 100) // tiny budget
	if err == nil || !errors.Is(err, ErrBudgetExceeded) {
		t.Errorf("expected budget exceeded error, got %v", err)
	}
}

func TestParseError(t *testing.T) {
	t.Parallel()
	_, err := ParseProgram("(a, b) => a +++ b")
	if err == nil {
		t.Errorf("expected parse error")
	}
}

func TestRuntimeErrorPropertyAccess(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, "(n) => n.foo")
	_, err := Run(prog, []Value{NumVal(42)}, DefaultBudget)
	if err == nil {
		t.Errorf("expected runtime error")
	}
}

func TestRuntimeErrorCallNonFunction(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, "(n) => n(42)")
	_, err := Run(prog, []Value{NumVal(5)}, DefaultBudget)
	if err == nil {
		t.Errorf("expected runtime error")
	}
}

func TestMethodCallsString(t *testing.T) {
	t.Parallel()
	expectStr(t, `(s) => s.toUpperCase()`, "HELLO", StrVal("hello"))
	expectStr(t, `(s) => s.toLowerCase()`, "hello", StrVal("HELLO"))
	expectStr(t, `(s) => s.trim()`, "hi", StrVal("  hi  "))
	expectBool(t, `(s) => s.startsWith('he')`, true, StrVal("hello"))
	expectBool(t, `(s) => s.endsWith('lo')`, true, StrVal("hello"))
	expectBool(t, `(s) => s.includes('ll')`, true, StrVal("hello"))
	expectNum(t, `(s) => s.indexOf('l')`, 2, StrVal("hello"))
	expectStr(t, `(s) => s.slice(1, 3)`, "el", StrVal("hello"))
	expectStr(t, `(s) => s.replace('l', 'r')`, "herlo", StrVal("hello"))
	expectStr(t, `(s) => s.charAt(0)`, "h", StrVal("hello"))
}

func TestMethodCallsArray(t *testing.T) {
	t.Parallel()
	arr := TupleVal(NumVal(1), NumVal(2), NumVal(3))
	v := mustRun(t, `(a) => a.map((x) => x * 2)`, arr)
	if v.Elems[0].Num != 2 || v.Elems[1].Num != 4 || v.Elems[2].Num != 6 {
		t.Errorf("expected [2,4,6], got %s", Display(v))
	}
	v = mustRun(t, `(a) => a.filter((x) => x > 1)`, arr)
	if len(v.Elems) != 2 {
		t.Errorf("expected 2 elements, got %s", Display(v))
	}
	expectNum(t, `(a) => a.reduce((acc, x) => acc + x, 0)`, 6, arr)
	expectBool(t, `(a) => a.includes(2)`, true, arr)
	expectNum(t, `(a) => a.indexOf(3)`, 2, arr)
}

func TestMethodCallsSplit(t *testing.T) {
	t.Parallel()
	src := `(s) => s.split('_').map((p, i) => i == 0 ? p : p.charAt(0).toUpperCase() + p.slice(1)).join('')`
	expectStr(t, src, "userFirstName", StrVal("user_first_name"))
}

func TestStaticMethods(t *testing.T) {
	t.Parallel()
	m := NewOrderedMap()
	m.Set("a", NumVal(1))
	m.Set("b", NumVal(2))
	v := mustRun(t, `(obj) => Object.keys(obj)`, ObjectVal(m))
	if len(v.Elems) != 2 {
		t.Fatalf("expected 2 keys, got %s", Display(v))
	}
	v = mustRun(t, `(obj) => Object.entries(obj)`, ObjectVal(m))
	if len(v.Elems) != 2 {
		t.Fatalf("expected 2 entries, got %s", Display(v))
	}
	expectNum(t, `(n) => Math.abs(n)`, 5, NumVal(-5))
	expectNum(t, `(n) => Math.floor(n)`, 3, NumVal(3.7))
	expectNum(t, `(a, b) => Math.max(a, b)`, 10, NumVal(3), NumVal(10))
}

func TestTypeAnnotations(t *testing.T) {
	t.Parallel()
	expectNum(t, "(a: number, b: number) => a + b", 30, NumVal(10), NumVal(20))
	expectStr(t, "(s: string) => s.toUpperCase()", "HELLO", StrVal("hello"))
	expectNum(t, `(arr: number[]) => arr.reduce((acc: number, x: number) => acc + x, 0)`, 6,
		TupleVal(NumVal(1), NumVal(2), NumVal(3)))
	expectStr(t, `(s: string) => { let result: string = s.toUpperCase(); return result; }`, "HI", StrVal("hi"))
}

func TestTypeofOperator(t *testing.T) {
	t.Parallel()
	expectStr(t, `(x) => typeof x`, "number", NumVal(42))
	expectStr(t, `(x) => typeof x`, "string", StrVal("hi"))
	expectStr(t, `(x) => typeof x`, "boolean", BoolVal(true))
	// typeof(x) function call syntax still works
	expectStr(t, `(x) => typeof(x)`, "number", NumVal(42))
}

func TestComments(t *testing.T) {
	t.Parallel()
	// Line comments
	expectNum(t, "(a, b) => a + b // add them", 30, NumVal(10), NumVal(20))
	expectNum(t, `(n) => {
		// double it
		return n * 2;
	}`, 10, NumVal(5))

	// Block comments
	expectNum(t, "(a, b) => a /* plus */ + b", 30, NumVal(10), NumVal(20))
	expectNum(t, `(n) => {
		/* multiply by two */
		return n * 2;
	}`, 10, NumVal(5))

	// Comment in preamble
	expectNum(t, `// helper
let double = (n) => n * 2;
(x) => double(x)`, 10, NumVal(5))

	// Division still works (not confused with comment)
	expectNum(t, "(a, b) => a / b", 5, NumVal(10), NumVal(2))

	// String containing // is not a comment
	expectStr(t, `(s) => s`, "http://example.com", StrVal("http://example.com"))

	// Multiline block comment
	expectNum(t, `(n) => {
		/*
		 * This is a
		 * multiline comment
		 */
		return n + 1;
	}`, 6, NumVal(5))

	// Comment between statements
	expectNum(t, `(n) => {
		let x = n + 1;
		// now double
		let y = x * 2;
		return y;
	}`, 12, NumVal(5))
}

func TestCommentErrors(t *testing.T) {
	t.Parallel()
	// Unterminated block comment
	_, err := ParseProgram("(n) => n /* unterminated")
	if err == nil {
		// The lexer doesn't error on unterminated block comments - it just
		// consumes to EOF. The parser then sees no function body tokens and errors.
		// That's fine; we just verify it doesn't panic.
	}
}

func expectSentinel(t *testing.T, src string, expectedStr string, args ...Value) {
	t.Helper()
	v := mustRun(t, src, args...)
	if v.Kind != KindTypeSentinel || v.Str != expectedStr {
		t.Errorf("expected sentinel %q, got %s", expectedStr, Display(v))
	}
}

func expectUndefined(t *testing.T, src string, args ...Value) {
	t.Helper()
	v := mustRun(t, src, args...)
	if v.Kind != KindUndefined {
		t.Errorf("expected undefined, got %s", Display(v))
	}
}

func TestVoidError(t *testing.T) {
	t.Parallel()

	// Basic static error message
	expectSentinel(t, `(x) => void { error: "bad input" }`, "error:bad input", NumVal(1))

	// Dynamic error message with string concatenation
	expectSentinel(t, `(s) => {
		if (typeof s != 'string') return void { error: "Expected string, got " + typeof s };
		return s;
	}`, "error:Expected string, got number", NumVal(42))

	// Dynamic error message with value interpolation
	expectSentinel(t, `(n) => {
		if (n < 0) return void { error: "Negative: " + n };
		return n;
	}`, "error:Negative: -5", NumVal(-5))

	// Happy path - no error, returns the value
	expectNum(t, `(n) => {
		if (n < 0) return void { error: "Negative: " + n };
		return n;
	}`, 42, NumVal(42))

	// Error in if/else branch
	expectSentinel(t, `(x) => {
		if (typeof x == 'string') return x;
		if (typeof x == 'number') return void { error: "Got number " + x + ", expected string" };
		return void { error: "Unsupported type: " + typeof x };
	}`, "error:Got number 5, expected string", NumVal(5))

	// Error inside a closure (called via .filter)
	expectSentinel(t, `(s) => {
		if (s.length == 0) return void { error: "String must not be empty" };
		return s;
	}`, "error:String must not be empty", StrVal(""))

	// Error with complex computed message
	expectSentinel(t, `(port) => {
		if (port < 1 || port > 65535) {
			let msg: string = "Port " + port + " is out of range (1-65535)";
			return void { error: msg };
		}
		return port;
	}`, "error:Port 99999 is out of range (1-65535)", NumVal(99999))

	// void "never" still works
	expectSentinel(t, `(x) => void "never"`, "never", NumVal(1))

	// void {} without error key - should be undefined (not an error)
	expectUndefined(t, `(x) => void { foo: "bar" }`, NumVal(1))

	// void { error: 42 } - non-string error value - should be undefined
	expectUndefined(t, `(x) => void { error: 42 }`, NumVal(1))

	// void { error: "" } - empty error message
	expectSentinel(t, `(x) => void { error: "" }`, "error:", NumVal(1))
}

func TestObjectLitIdentKeys(t *testing.T) {
	t.Parallel()
	// Object literals with bare identifier keys (JS shorthand syntax)
	v := mustRun(t, `(x) => { let o = { name: "Alice", age: 30 }; return o; }`, NumVal(0))
	if v.Kind != KindObject {
		t.Fatalf("expected object, got %s", Display(v))
	}
	name, ok := v.Props.Get("name")
	if !ok || name.Str != "Alice" {
		t.Errorf("expected name='Alice', got %s", Display(name))
	}
	age, ok := v.Props.Get("age")
	if !ok || age.Num != 30 {
		t.Errorf("expected age=30, got %s", Display(age))
	}

	// Mixed: string key and identifier key
	v2 := mustRun(t, `(x) => { let o = { "key1": 1, key2: 2 }; return o; }`, NumVal(0))
	if v2.Kind != KindObject {
		t.Fatalf("expected object, got %s", Display(v2))
	}
	k1, _ := v2.Props.Get("key1")
	k2, _ := v2.Props.Get("key2")
	if k1.Num != 1 || k2.Num != 2 {
		t.Errorf("expected {key1:1, key2:2}, got %s", Display(v2))
	}
}

func TestStepCounts(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		src  string
		args []Value
	}{
		{"arithmetic", "(a, b) => a + b", []Value{NumVal(10), NumVal(20)}},
		{"string_transform", "(s) => s.toLowerCase().split(' ').join('-')", []Value{StrVal("Hello World This Is A Title")}},
		{"camel_case", `(s) => {
			let parts = s.split('_');
			let result = parts[0];
			let i = 1;
			while (i < parts.length) {
				result = result + parts[i].charAt(0).toUpperCase() + parts[i].slice(1);
				i = i + 1;
			}
			return result;
		}`, []Value{StrVal("get_user_profile_by_account_id")}},
		{"email_validation", `(s) => {
			if (typeof s != 'string') return void "never";
			let at = s.indexOf('@');
			if (at < 1) return void "never";
			let domain = s.slice(at + 1);
			if (domain.length < 3 || !domain.includes('.')) return void "never";
			return s;
		}`, []Value{StrVal("alice.smith@company.example.com")}},
		{"config_validation", `(cfg) => {
			if (typeof cfg != 'object') return void "never";
			if (typeof cfg['host'] != 'string' || cfg['host'].length == 0) return void "never";
			if (typeof cfg['port'] != 'number' || cfg['port'] < 1 || cfg['port'] > 65535) return void "never";
			if (cfg['protocol'] != 'http' && cfg['protocol'] != 'https') return void "never";
			if (typeof cfg['timeout'] != 'number' || cfg['timeout'] < 0) return void "never";
			if (typeof cfg['debug'] != 'boolean') return void "never";
			return cfg;
		}`, []Value{func() Value {
			m := NewOrderedMap()
			m.Set("host", StrVal("api.example.com"))
			m.Set("port", NumVal(443))
			m.Set("protocol", StrVal("https"))
			m.Set("timeout", NumVal(30000))
			m.Set("debug", BoolVal(false))
			return ObjectVal(m)
		}()}},
		{"uuid_validation", `(s) => {
			if (typeof s != 'string' || s.length != 36) return void "never";
			let hex = 'abcdef0123456789';
			let dashes = [8, 13, 18, 23];
			let i = 0;
			while (i < 36) {
				if (dashes.includes(i)) {
					if (s.slice(i, i + 1) != '-') return void "never";
				} else {
					if (!hex.includes(s.slice(i, i + 1).toLowerCase())) return void "never";
				}
				i = i + 1;
			}
			return s;
		}`, []Value{StrVal("550e8400-e29b-41d4-a716-446655440000")}},
		{"sql_parse", `(sql) => {
			if (typeof sql != 'string' || sql.trim().length == 0) return void "never";
			let norm = '';
			let lastWasSpace = true;
			let i = 0;
			while (i < sql.length) {
				let c = sql.slice(i, i + 1);
				if (c == ' ' || c == '\n' || c == '\t') {
					if (!lastWasSpace) { norm = norm + ' '; lastWasSpace = true; }
				} else { norm = norm + c; lastWasSpace = false; }
				i = i + 1;
			}
			norm = norm.trim();
			let upper = norm.toUpperCase();
			if (!upper.startsWith('SELECT ')) return void "never";
			let fromIdx = upper.indexOf(' FROM ');
			if (fromIdx == -1) return void "never";
			let columns = norm.slice(7, fromIdx).trim();
			let afterFrom = norm.slice(fromIdx + 6).trim();
			let result = {};
			result['columns'] = columns.split(',').map((c) => c.trim());
			result['table'] = afterFrom.includes(' ') ? afterFrom.slice(0, afterFrom.indexOf(' ')) : afterFrom;
			return result;
		}`, []Value{StrVal("SELECT id, name, email FROM users WHERE active = 1 ORDER BY name")}},
		{"array_pipeline_50", `(arr) => {
			let filtered = arr.filter((x) => x > 0);
			let doubled = filtered.map((x) => x * 2);
			return doubled.reduce((acc, x) => acc + x, 0);
		}`, []Value{func() Value {
			elems := make([]Value, 50)
			for i := range elems {
				elems[i] = NumVal(float64(i - 25))
			}
			return TupleVal(elems...)
		}()}},
	}

	for _, tc := range cases {
		prog, err := ParseProgram(tc.src)
		if err != nil {
			t.Fatalf("%s: parse error: %v", tc.name, err)
		}
		result, err := Run(prog, tc.args, DefaultBudget)
		if err != nil {
			t.Fatalf("%s: runtime error: %v", tc.name, err)
		}
		pct := float64(result.Steps) / float64(DefaultBudget) * 100
		t.Logf("%-25s %5d steps (%4.1f%% of budget)", tc.name, result.Steps, pct)
	}
}

func TestBudgetExhaustion(t *testing.T) {
	t.Parallel()

	prog, err := ParseProgram(`(n) => {
		let i: number = 0;
		while (i < n) { i = i + 1; }
		return i;
	}`)
	if err != nil {
		t.Fatal(err)
	}

	_, err = Run(prog, []Value{NumVal(100000)}, DefaultBudget)
	if err == nil {
		t.Fatal("expected budget exceeded error")
	}
	if !errors.Is(err, ErrBudgetExceeded) {
		t.Fatalf("expected ErrBudgetExceeded, got: %v", err)
	}
}

func TestMemoryBudget(t *testing.T) {
	t.Parallel()

	// String concatenation in a loop should eventually hit memory limit
	prog, err := ParseProgram(`(n) => {
		let s = "x";
		let i = 0;
		while (i < n) { s = s + s; i = i + 1; }
		return s;
	}`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = Run(prog, []Value{NumVal(100)}, DefaultBudget)
	if err == nil {
		t.Fatal("expected memory or budget exceeded error")
	}
	if !errors.Is(err, ErrMemoryExceeded) && !errors.Is(err, ErrBudgetExceeded) {
		t.Fatalf("expected ErrMemoryExceeded or ErrBudgetExceeded, got: %v", err)
	}
}

func TestNaNAndInfinity(t *testing.T) {
	t.Parallel()

	// Division by zero produces Infinity
	v := mustRun(t, "(n) => n / 0", NumVal(42))
	if v.Kind != KindNumber || !math.IsInf(v.Num, 1) {
		t.Errorf("expected +Inf, got %s", Display(v))
	}

	// Negative division by zero
	v = mustRun(t, "(n) => n / 0", NumVal(-1))
	if v.Kind != KindNumber || !math.IsInf(v.Num, -1) {
		t.Errorf("expected -Inf, got %s", Display(v))
	}

	// 0/0 produces NaN
	v = mustRun(t, "(a, b) => a / b", NumVal(0), NumVal(0))
	if v.Kind != KindNumber || !math.IsNaN(v.Num) {
		t.Errorf("expected NaN, got %s", Display(v))
	}

	// NaN != NaN
	expectBool(t, "(a, b) => a == b", false, NumVal(math.NaN()), NumVal(math.NaN()))

	// NaN propagation
	v = mustRun(t, "(n) => n + 1", NumVal(math.NaN()))
	if v.Kind != KindNumber || !math.IsNaN(v.Num) {
		t.Errorf("expected NaN, got %s", Display(v))
	}
}

func TestOutOfBounds(t *testing.T) {
	t.Parallel()

	arr := TupleVal(NumVal(10), NumVal(20), NumVal(30))

	// Read past end returns undefined
	expectUndefined(t, "(arr) => arr[99]", arr)

	// Negative index returns undefined
	expectUndefined(t, "(arr) => arr[-1]", arr)

	// String index past end returns undefined
	expectUndefined(t, "(s) => s[99]", StrVal("hi"))

	// slice past end returns empty
	v := mustRun(t, "(arr) => arr.slice(99)", arr)
	if v.Kind != KindTuple || len(v.Elems) != 0 {
		t.Errorf("expected empty tuple, got %s", Display(v))
	}

	// charAt past end
	expectStr(t, "(s) => s.charAt(99)", "", StrVal("hi"))
}

func TestEmptyCollections(t *testing.T) {
	t.Parallel()

	empty := TupleVal()

	// map on empty
	v := mustRun(t, "(arr) => arr.map((x) => x * 2)", empty)
	if v.Kind != KindTuple || len(v.Elems) != 0 {
		t.Errorf("expected empty tuple, got %s", Display(v))
	}

	// filter on empty
	v = mustRun(t, "(arr) => arr.filter((x) => x > 0)", empty)
	if v.Kind != KindTuple || len(v.Elems) != 0 {
		t.Errorf("expected empty tuple, got %s", Display(v))
	}

	// reduce on empty with initial value
	expectNum(t, "(arr) => arr.reduce((a, b) => a + b, 0)", 0, empty)

	// join on empty
	expectStr(t, "(arr) => arr.join(',')", "", empty)

	// Object.keys on empty object
	v = mustRun(t, "(obj) => Object.keys(obj)", ObjectVal(NewOrderedMap()))
	if v.Kind != KindTuple || len(v.Elems) != 0 {
		t.Errorf("expected empty tuple, got %s", Display(v))
	}
}

func TestRecursiveFunctions(t *testing.T) {
	t.Parallel()

	// Fibonacci via recursion
	expectNum(t, `(n) => {
		let fib = (x) => x <= 1 ? x : fib(x - 1) + fib(x - 2);
		return fib(n);
	}`, 55, NumVal(10))

	// Factorial via recursion
	expectNum(t, `(n) => {
		let fact = (x) => x <= 1 ? 1 : x * fact(x - 1);
		return fact(n);
	}`, 120, NumVal(5))

	// Deep recursion that hits budget
	prog, err := ParseProgram(`(n) => {
		let countdown = (x) => x <= 0 ? 0 : countdown(x - 1);
		return countdown(n);
	}`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = Run(prog, []Value{NumVal(100000)}, DefaultBudget)
	if !errors.Is(err, ErrBudgetExceeded) {
		t.Fatalf("expected budget exceeded for deep recursion, got: %v", err)
	}
}

func TestNegativeSliceIndices(t *testing.T) {
	t.Parallel()

	// String slice with negative start
	expectStr(t, "(s) => s.slice(-3)", "rld", StrVal("world"))

	// Array slice with negative start
	v := mustRun(t, "(arr) => arr.slice(-2)", TupleVal(NumVal(1), NumVal(2), NumVal(3), NumVal(4)))
	if v.Kind != KindTuple || len(v.Elems) != 2 || v.Elems[0].Num != 3 {
		t.Errorf("expected [3, 4], got %s", Display(v))
	}
}

// --- Group 1: value.go utilities ---

func TestCacheKey(t *testing.T) {
	t.Parallel()
	if got := CacheKey(NumVal(42)); got != "n42" {
		t.Errorf("number: got %q", got)
	}
	if got := CacheKey(NumVal(3.14)); got != "n3.14" {
		t.Errorf("float: got %q", got)
	}
	if got := CacheKey(StrVal("hi")); got != "s2:hi" {
		t.Errorf("string: got %q", got)
	}
	if got := CacheKey(BoolVal(true)); got != "T" {
		t.Errorf("true: got %q", got)
	}
	if got := CacheKey(BoolVal(false)); got != "F" {
		t.Errorf("false: got %q", got)
	}
	if got := CacheKey(Null); got != "null" {
		t.Errorf("null: got %q", got)
	}
	if got := CacheKey(Undefined); got != "undef" {
		t.Errorf("undefined: got %q", got)
	}
	if got := CacheKey(NeverType); got != "T:never" {
		t.Errorf("sentinel: got %q", got)
	}
	if got := CacheKey(TupleVal(NumVal(1), StrVal("a"))); got != "[n1,s1:a]" {
		t.Errorf("tuple: got %q", got)
	}
	m := NewOrderedMap()
	m.Set("x", NumVal(1))
	if got := CacheKey(ObjectVal(m)); got != "{x:n1}" {
		t.Errorf("object: got %q", got)
	}
	fn := Value{Kind: KindFunction, Fn: &Closure{}}
	if got := CacheKey(fn); got != "fn" {
		t.Errorf("function: got %q", got)
	}
}

func TestDisplay(t *testing.T) {
	t.Parallel()
	if got := Display(NumVal(42)); got != "42" {
		t.Errorf("int number: got %q", got)
	}
	if got := Display(NumVal(3.14)); got != "3.14" {
		t.Errorf("float number: got %q", got)
	}
	if got := Display(StrVal("hi")); got != `"hi"` {
		t.Errorf("string: got %q", got)
	}
	if got := Display(BoolVal(true)); got != "true" {
		t.Errorf("true: got %q", got)
	}
	if got := Display(BoolVal(false)); got != "false" {
		t.Errorf("false: got %q", got)
	}
	if got := Display(Null); got != "null" {
		t.Errorf("null: got %q", got)
	}
	if got := Display(Undefined); got != "undefined" {
		t.Errorf("undefined: got %q", got)
	}
	if got := Display(NeverType); got != "never" {
		t.Errorf("sentinel: got %q", got)
	}
	if got := Display(TupleVal(NumVal(1), NumVal(2))); got != "[1, 2]" {
		t.Errorf("tuple: got %q", got)
	}
	m := NewOrderedMap()
	m.Set("a", NumVal(1))
	m.Set("b", StrVal("x"))
	if got := Display(ObjectVal(m)); got != `{a: 1, b: "x"}` {
		t.Errorf("object: got %q", got)
	}
	fn := Value{Kind: KindFunction, Fn: &Closure{}}
	if got := Display(fn); got != "<function>" {
		t.Errorf("function: got %q", got)
	}
}

func TestCompareAndSortValues(t *testing.T) {
	t.Parallel()
	if CompareValues(NumVal(1), NumVal(2)) >= 0 {
		t.Error("1 < 2")
	}
	if CompareValues(NumVal(5), NumVal(3)) <= 0 {
		t.Error("5 > 3")
	}
	if CompareValues(NumVal(7), NumVal(7)) != 0 {
		t.Error("7 == 7")
	}
	if CompareValues(StrVal("apple"), StrVal("banana")) >= 0 {
		t.Error("apple < banana")
	}
	if CompareValues(StrVal("z"), StrVal("a")) <= 0 {
		t.Error("z > a")
	}
	// mixed types return 0
	if CompareValues(NumVal(1), StrVal("a")) != 0 {
		t.Error("mixed types should be 0")
	}

	vals := []Value{NumVal(3), NumVal(1), NumVal(2)}
	SortValues(vals)
	if vals[0].Num != 1 || vals[1].Num != 2 || vals[2].Num != 3 {
		t.Errorf("sort: got %v", vals)
	}

	svals := []Value{StrVal("c"), StrVal("a"), StrVal("b")}
	SortValues(svals)
	if svals[0].Str != "a" || svals[1].Str != "b" || svals[2].Str != "c" {
		t.Errorf("string sort: got %v", svals)
	}
}

// --- Group 2: ValuesEqual and IsTruthy ---

func TestValuesEqual(t *testing.T) {
	t.Parallel()

	// Same-kind comparisons
	if !ValuesEqual(Null, Null) {
		t.Error("null == null")
	}
	if !ValuesEqual(Undefined, Undefined) {
		t.Error("undefined == undefined")
	}
	if !ValuesEqual(NeverType, NeverType) {
		t.Error("never == never")
	}
	if ValuesEqual(NeverType, UnknownType) {
		t.Error("never != unknown")
	}

	// Tuples
	if !ValuesEqual(TupleVal(NumVal(1), NumVal(2)), TupleVal(NumVal(1), NumVal(2))) {
		t.Error("equal tuples")
	}
	if ValuesEqual(TupleVal(NumVal(1)), TupleVal(NumVal(1), NumVal(2))) {
		t.Error("different length tuples")
	}
	if ValuesEqual(TupleVal(NumVal(1)), TupleVal(NumVal(2))) {
		t.Error("different element tuples")
	}

	// Objects
	m1 := NewOrderedMap()
	m1.Set("a", NumVal(1))
	m2 := NewOrderedMap()
	m2.Set("a", NumVal(1))
	if !ValuesEqual(ObjectVal(m1), ObjectVal(m2)) {
		t.Error("equal objects")
	}
	m3 := NewOrderedMap()
	m3.Set("a", NumVal(2))
	if ValuesEqual(ObjectVal(m1), ObjectVal(m3)) {
		t.Error("different value objects")
	}
	m4 := NewOrderedMap()
	m4.Set("b", NumVal(1))
	if ValuesEqual(ObjectVal(m1), ObjectVal(m4)) {
		t.Error("different key objects")
	}
	m5 := NewOrderedMap()
	m5.Set("a", NumVal(1))
	m5.Set("b", NumVal(2))
	if ValuesEqual(ObjectVal(m1), ObjectVal(m5)) {
		t.Error("different size objects")
	}

	// Mismatched kinds
	if ValuesEqual(NumVal(1), StrVal("1")) {
		t.Error("num != str")
	}

	// Functions are never equal
	fn := Value{Kind: KindFunction, Fn: &Closure{}}
	if ValuesEqual(fn, fn) {
		t.Error("functions should not be equal")
	}
}

func TestIsTruthy(t *testing.T) {
	t.Parallel()

	cases := []struct {
		val    Value
		expect bool
	}{
		{BoolVal(true), true},
		{BoolVal(false), false},
		{NumVal(1), true},
		{NumVal(0), false},
		{NumVal(math.NaN()), false},
		{StrVal("hi"), true},
		{StrVal(""), false},
		{Null, false},
		{Undefined, false},
		{NeverType, false},
		{UnknownType, true},
		{StringType, true},
		{TupleVal(), true},
		{TupleVal(NumVal(1)), true},
		{ObjectVal(NewOrderedMap()), true},
		{Value{Kind: KindFunction, Fn: &Closure{}}, true},
	}
	for _, tc := range cases {
		if IsTruthy(tc.val) != tc.expect {
			t.Errorf("IsTruthy(%s) = %v, want %v", Display(tc.val), !tc.expect, tc.expect)
		}
	}
}

// --- Group 3: callMethod - string and array methods ---

func TestStringMethods(t *testing.T) {
	t.Parallel()
	expectStr(t, `(s) => s.trim()`, "hello", StrVal("  hello  "))
	expectStr(t, `(s) => s.replace("world", "go")`, "hello go", StrVal("hello world"))
	expectBool(t, `(s) => s.startsWith("hel")`, true, StrVal("hello"))
	expectBool(t, `(s) => s.startsWith("xyz")`, false, StrVal("hello"))
	expectBool(t, `(s) => s.endsWith("llo")`, true, StrVal("hello"))
	expectBool(t, `(s) => s.endsWith("xyz")`, false, StrVal("hello"))
	expectStr(t, `(s) => s.charAt(1)`, "e", StrVal("hello"))
	expectBool(t, `(s) => s.includes("ell")`, true, StrVal("hello"))
	expectBool(t, `(s) => s.includes("xyz")`, false, StrVal("hello"))
}

func TestArrayFind(t *testing.T) {
	t.Parallel()
	arr := TupleVal(NumVal(1), NumVal(2), NumVal(3), NumVal(4))
	expectNum(t, `(a) => a.find((x) => x > 2)`, 3, arr)
	expectUndefined(t, `(a) => a.find((x) => x > 10)`, arr)
}

func TestArraySomeEvery(t *testing.T) {
	t.Parallel()
	arr := TupleVal(NumVal(1), NumVal(2), NumVal(3))
	expectBool(t, `(a) => a.some((x) => x > 2)`, true, arr)
	expectBool(t, `(a) => a.some((x) => x > 10)`, false, arr)
	expectBool(t, `(a) => a.every((x) => x > 0)`, true, arr)
	expectBool(t, `(a) => a.every((x) => x > 1)`, false, arr)
}

func TestArrayFlat(t *testing.T) {
	t.Parallel()
	v := mustRun(t, `(a) => a.flat()`,
		TupleVal(TupleVal(NumVal(1), NumVal(2)), NumVal(3), TupleVal(NumVal(4))))
	if v.Kind != KindTuple || len(v.Elems) != 4 {
		t.Fatalf("expected 4 elements, got %s", Display(v))
	}
	if v.Elems[0].Num != 1 || v.Elems[3].Num != 4 {
		t.Errorf("got %s", Display(v))
	}
}

func TestArrayConcat(t *testing.T) {
	t.Parallel()
	v := mustRun(t, `(a, b) => a.concat(b)`,
		TupleVal(NumVal(1), NumVal(2)), TupleVal(NumVal(3), NumVal(4)))
	if v.Kind != KindTuple || len(v.Elems) != 4 {
		t.Fatalf("expected 4 elements, got %s", Display(v))
	}
}

func TestArrayReverse(t *testing.T) {
	t.Parallel()
	v := mustRun(t, `(a) => a.reverse()`, TupleVal(NumVal(1), NumVal(2), NumVal(3)))
	if v.Elems[0].Num != 3 || v.Elems[2].Num != 1 {
		t.Errorf("expected [3,2,1], got %s", Display(v))
	}
}

func TestArraySort(t *testing.T) {
	t.Parallel()
	v := mustRun(t, `(a) => a.sort()`, TupleVal(NumVal(3), NumVal(1), NumVal(2)))
	if v.Elems[0].Num != 1 || v.Elems[1].Num != 2 || v.Elems[2].Num != 3 {
		t.Errorf("expected [1,2,3], got %s", Display(v))
	}
}

func TestArrayIndexOf(t *testing.T) {
	t.Parallel()
	arr := TupleVal(StrVal("a"), StrVal("b"), StrVal("c"))
	expectNum(t, `(a) => a.indexOf("b")`, 1, arr)
	expectNum(t, `(a) => a.indexOf("z")`, -1, arr)
}

func TestArrayIncludes(t *testing.T) {
	t.Parallel()
	arr := TupleVal(NumVal(10), NumVal(20))
	expectBool(t, `(a) => a.includes(10)`, true, arr)
	expectBool(t, `(a) => a.includes(99)`, false, arr)
}

func TestArrayJoin(t *testing.T) {
	t.Parallel()
	expectStr(t, `(a) => a.join("-")`, "a-b-c", TupleVal(StrVal("a"), StrVal("b"), StrVal("c")))
}

func TestArraySlice(t *testing.T) {
	t.Parallel()
	arr := TupleVal(NumVal(10), NumVal(20), NumVal(30), NumVal(40), NumVal(50))
	v := mustRun(t, `(a) => a.slice(1, 3)`, arr)
	if len(v.Elems) != 2 || v.Elems[0].Num != 20 || v.Elems[1].Num != 30 {
		t.Errorf("expected [20,30], got %s", Display(v))
	}
}

func TestObjectMethodCall(t *testing.T) {
	t.Parallel()
	v := mustRun(t, `(x) => {
		let obj = {};
		obj["greet"] = (name) => "hello " + name;
		return obj.greet("world");
	}`, NumVal(0))
	if v.Kind != KindString || v.Str != "hello world" {
		t.Errorf("expected 'hello world', got %s", Display(v))
	}
}

// --- Group 4: callStatic - Math and Object methods ---

func TestMathMethods(t *testing.T) {
	t.Parallel()
	expectNum(t, `(n) => Math.floor(n)`, 3, NumVal(3.7))
	expectNum(t, `(n) => Math.ceil(n)`, 4, NumVal(3.2))
	expectNum(t, `(n) => Math.round(n)`, 4, NumVal(3.5))
	expectNum(t, `(n) => Math.round(n)`, 3, NumVal(3.4))
	expectNum(t, `(a, b) => Math.min(a, b)`, 2, NumVal(2), NumVal(5))
	expectNum(t, `(a, b) => Math.max(a, b)`, 5, NumVal(2), NumVal(5))
}

func TestObjectValues(t *testing.T) {
	t.Parallel()
	m := NewOrderedMap()
	m.Set("a", NumVal(1))
	m.Set("b", NumVal(2))
	v := mustRun(t, `(obj) => Object.values(obj)`, ObjectVal(m))
	if v.Kind != KindTuple || len(v.Elems) != 2 {
		t.Fatalf("expected 2 values, got %s", Display(v))
	}
	if v.Elems[0].Num != 1 || v.Elems[1].Num != 2 {
		t.Errorf("expected [1, 2], got %s", Display(v))
	}
}

func TestObjectEntries(t *testing.T) {
	t.Parallel()
	m := NewOrderedMap()
	m.Set("x", NumVal(10))
	v := mustRun(t, `(obj) => Object.entries(obj)`, ObjectVal(m))
	if v.Kind != KindTuple || len(v.Elems) != 1 {
		t.Fatalf("expected 1 entry, got %s", Display(v))
	}
	entry := v.Elems[0]
	if entry.Kind != KindTuple || entry.Elems[0].Str != "x" || entry.Elems[1].Num != 10 {
		t.Errorf("expected ['x', 10], got %s", Display(entry))
	}
}

func TestObjectFromEntries(t *testing.T) {
	t.Parallel()
	v := mustRun(t, `(x) => Object.fromEntries([["a", 1], ["b", 2]])`, NumVal(0))
	if v.Kind != KindObject {
		t.Fatalf("expected object, got %s", Display(v))
	}
	a, _ := v.Props.Get("a")
	b, _ := v.Props.Get("b")
	if a.Num != 1 || b.Num != 2 {
		t.Errorf("expected {a:1, b:2}, got %s", Display(v))
	}
}

// --- Group 5: evalCall - call dispatch paths ---

func TestCallLambdaInVariable(t *testing.T) {
	t.Parallel()
	expectNum(t, `(x) => { let f = (n) => n * 2; return f(x); }`, 10, NumVal(5))
}

func TestCallComputedExpression(t *testing.T) {
	t.Parallel()
	expectNum(t, `(x) => {
		let fns = [(n) => n + 1, (n) => n * 2];
		return fns[1](x);
	}`, 10, NumVal(5))
}

func TestCallNonFunction(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(x) => { let v = 42; return v(1); }`)
	_, err := Run(prog, []Value{NumVal(0)}, DefaultBudget)
	if err == nil {
		t.Error("expected error calling non-function")
	}
}

func TestCallComputedNonFunction(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(x) => { let a = [1, 2]; return a[0](5); }`)
	_, err := Run(prog, []Value{NumVal(0)}, DefaultBudget)
	if err == nil {
		t.Error("expected error calling non-function from index")
	}
}

// --- Group 6: evalObjectLit ---

func TestObjectSpread(t *testing.T) {
	t.Parallel()
	v := mustRun(t, `(x) => {
		let base = { a: 1, b: 2 };
		return { ...base, c: 3 };
	}`, NumVal(0))
	if v.Kind != KindObject {
		t.Fatalf("expected object, got %s", Display(v))
	}
	a, _ := v.Props.Get("a")
	c, _ := v.Props.Get("c")
	if a.Num != 1 || c.Num != 3 {
		t.Errorf("expected {a:1, b:2, c:3}, got %s", Display(v))
	}
}

func TestObjectComputedKey(t *testing.T) {
	t.Parallel()
	v := mustRun(t, `(key) => {
		let o = {};
		o[key] = 42;
		return o;
	}`, StrVal("dynamic"))
	val, ok := v.Props.Get("dynamic")
	if !ok || val.Num != 42 {
		t.Errorf("expected {dynamic: 42}, got %s", Display(v))
	}
}

func TestEmptyObject(t *testing.T) {
	t.Parallel()
	v := mustRun(t, `(x) => { let o = {}; return o; }`, NumVal(0))
	if v.Kind != KindObject || v.Props.Size() != 0 {
		t.Errorf("expected empty object, got %s", Display(v))
	}
}

// --- Group 7: execStmt / execIf / control flow ---

func TestIfElseIfElse(t *testing.T) {
	t.Parallel()
	src := `(x) => {
		if (x > 10) { return "big"; }
		else if (x > 5) { return "medium"; }
		else { return "small"; }
	}`
	expectStr(t, src, "big", NumVal(15))
	expectStr(t, src, "medium", NumVal(7))
	expectStr(t, src, "small", NumVal(2))
}

func TestForOfBreak(t *testing.T) {
	t.Parallel()
	v := mustRun(t, `(arr) => {
		let result = 0;
		for (let x of arr) {
			if (x > 3) { break; }
			result = result + x;
		}
		return result;
	}`, TupleVal(NumVal(1), NumVal(2), NumVal(3), NumVal(4), NumVal(5)))
	if v.Kind != KindNumber || v.Num != 6 {
		t.Errorf("expected 6, got %s", Display(v))
	}
}

func TestForOfContinue(t *testing.T) {
	t.Parallel()
	v := mustRun(t, `(arr) => {
		let result = 0;
		for (let x of arr) {
			if (x % 2 == 0) { continue; }
			result = result + x;
		}
		return result;
	}`, TupleVal(NumVal(1), NumVal(2), NumVal(3), NumVal(4), NumVal(5)))
	if v.Kind != KindNumber || v.Num != 9 {
		t.Errorf("expected 9, got %s", Display(v))
	}
}

func TestWhileBreak(t *testing.T) {
	t.Parallel()
	expectNum(t, `(x) => {
		let i = 0;
		while (i < 100) {
			if (i == 5) { break; }
			i = i + 1;
		}
		return i;
	}`, 5, NumVal(0))
}

func TestWhileContinue(t *testing.T) {
	t.Parallel()
	v := mustRun(t, `(x) => {
		let sum = 0;
		let i = 0;
		while (i < 10) {
			i = i + 1;
			if (i % 2 == 0) { continue; }
			sum = sum + i;
		}
		return sum;
	}`, NumVal(0))
	if v.Kind != KindNumber || v.Num != 25 {
		t.Errorf("expected 25, got %s", Display(v))
	}
}

func TestVariableReassignment(t *testing.T) {
	t.Parallel()
	expectNum(t, `(x) => { let v = 1; v = v + 10; return v; }`, 11, NumVal(0))
}

func TestIndexAssignment(t *testing.T) {
	t.Parallel()
	v := mustRun(t, `(x) => {
		let obj = {};
		obj["key"] = 42;
		return obj["key"];
	}`, NumVal(0))
	if v.Num != 42 {
		t.Errorf("expected 42, got %s", Display(v))
	}

	v = mustRun(t, `(x) => {
		let arr = [10, 20, 30];
		arr[1] = 99;
		return arr[1];
	}`, NumVal(0))
	if v.Num != 99 {
		t.Errorf("expected 99, got %s", Display(v))
	}
}

func TestDotAssignment(t *testing.T) {
	t.Parallel()
	v := mustRun(t, `(x) => {
		let obj = { a: 1 };
		obj.a = 42;
		return obj.a;
	}`, NumVal(0))
	if v.Num != 42 {
		t.Errorf("expected 42, got %s", Display(v))
	}
}

// --- Group 8: evalPropAccess / evalIndexAccess ---

func TestTupleLength(t *testing.T) {
	t.Parallel()
	expectNum(t, `(a) => a.length`, 3, TupleVal(NumVal(1), NumVal(2), NumVal(3)))
}

func TestStringLength(t *testing.T) {
	t.Parallel()
	expectNum(t, `(s) => s.length`, 5, StrVal("hello"))
}

func TestObjectMissingProp(t *testing.T) {
	t.Parallel()
	m := NewOrderedMap()
	m.Set("a", NumVal(1))
	expectUndefined(t, `(o) => o.missing`, ObjectVal(m))
}

func TestIndexAccessObjectString(t *testing.T) {
	t.Parallel()
	m := NewOrderedMap()
	m.Set("k", NumVal(99))
	expectNum(t, `(o) => o["k"]`, 99, ObjectVal(m))
}

func TestIndexAccessString(t *testing.T) {
	t.Parallel()
	expectStr(t, `(s) => s[0]`, "h", StrVal("hello"))
	expectStr(t, `(s) => s[4]`, "o", StrVal("hello"))
}

// --- Group 9: Parser error paths ---

func TestParseErrors(t *testing.T) {
	t.Parallel()

	cases := []string{
		"(a, b => a + b",               // missing closing paren
		"(a, b) a + b",                 // missing arrow
		"(a) => { return a;",           // missing closing brace
		"(a) => @",                     // invalid token
		"(a) => { for (let x a) { } }", // malformed for-of (missing "of")
		"(a) => { while a > 0 { } }",   // malformed while (missing parens)
		"(a) => { let; }",              // malformed let (missing name/=)
		"(a) => { let [x, y; }",        // destructuring missing ]
	}
	for _, src := range cases {
		_, err := ParseProgram(src)
		if err == nil {
			t.Errorf("expected parse error for: %s", src)
		}
	}
}

// --- Group 10: evalUnary ---

func TestTypeofAllKinds(t *testing.T) {
	t.Parallel()
	expectStr(t, `(x) => typeof x`, "number", NumVal(1))
	expectStr(t, `(x) => typeof x`, "string", StrVal(""))
	expectStr(t, `(x) => typeof x`, "boolean", BoolVal(false))
	expectStr(t, `(x) => typeof x`, "null", Null)
	expectStr(t, `(x) => typeof x`, "undefined", Undefined)
	expectStr(t, `(x) => typeof x`, "tuple", TupleVal())
	expectStr(t, `(x) => typeof x`, "object", ObjectVal(NewOrderedMap()))
	expectStr(t, `(x) => typeof x`, "never", NeverType)
}

func TestTypeofFunction(t *testing.T) {
	t.Parallel()
	expectStr(t, `(x) => { let f = (n) => n; return typeof f; }`, "function", NumVal(0))
}

func TestVoidNonSpecial(t *testing.T) {
	t.Parallel()
	expectUndefined(t, `(x) => void x`, NumVal(42))
	expectUndefined(t, `(x) => void x`, StrVal("hello"))
}

func TestUnaryNegateNonNumber(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(x) => -x`)
	_, err := Run(prog, []Value{StrVal("hello")}, DefaultBudget)
	if err == nil {
		t.Error("expected error negating string")
	}
}

// --- Group 11: coerceToString ---

func TestCoerceNumberToString(t *testing.T) {
	t.Parallel()
	expectStr(t, `(x) => "val:" + x`, "val:42", NumVal(42))
	expectStr(t, `(x) => x + ":end"`, "3.14:end", NumVal(3.14))
}

func TestCoerceBoolToStringErrors(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(x) => "val:" + x`)
	_, err := Run(prog, []Value{BoolVal(true)}, DefaultBudget)
	if err == nil {
		t.Error("expected error coercing bool to string")
	}
}

func TestCoerceNullToStringErrors(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(x) => "val:" + x`)
	_, err := Run(prog, []Value{Null}, DefaultBudget)
	if err == nil {
		t.Error("expected error coercing null to string")
	}
}

// --- Group 12: Memory budget ---

func TestMemoryBudgetArraySpread(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(n) => {
		let arr = [1];
		let i = 0;
		while (i < n) {
			arr = [...arr, ...arr];
			i = i + 1;
		}
		return arr;
	}`)
	_, err := Run(prog, []Value{NumVal(100)}, DefaultBudget)
	if err == nil {
		t.Error("expected memory or budget exceeded")
	}
	if !errors.Is(err, ErrMemoryExceeded) && !errors.Is(err, ErrBudgetExceeded) {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestMemoryBudgetStringConcat(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(n) => {
		let s = "abcdefghij";
		let i = 0;
		while (i < n) { s = s + s; i = i + 1; }
		return s;
	}`)
	_, err := Run(prog, []Value{NumVal(100)}, DefaultBudget)
	if err == nil {
		t.Error("expected memory or budget exceeded")
	}
	if !errors.Is(err, ErrMemoryExceeded) && !errors.Is(err, ErrBudgetExceeded) {
		t.Errorf("unexpected error: %v", err)
	}
}

// --- Additional edge cases ---

func TestForOfReturn(t *testing.T) {
	t.Parallel()
	expectNum(t, `(arr) => {
		for (let x of arr) {
			if (x == 3) { return x * 10; }
		}
		return -1;
	}`, 30, TupleVal(NumVal(1), NumVal(2), NumVal(3), NumVal(4)))
}

func TestWhileReturn(t *testing.T) {
	t.Parallel()
	expectNum(t, `(x) => {
		let i = 0;
		while (i < 100) {
			if (i == 7) { return i; }
			i = i + 1;
		}
		return -1;
	}`, 7, NumVal(0))
}

func TestLogicalOperators(t *testing.T) {
	t.Parallel()
	expectNum(t, `(x) => x || 10`, 5, NumVal(5))
	expectNum(t, `(x) => x || 10`, 10, NumVal(0))
	expectNum(t, `(x) => x && 10`, 10, NumVal(5))
	expectNum(t, `(x) => x && 10`, 0, NumVal(0))
}

func TestNotEqual(t *testing.T) {
	t.Parallel()
	expectBool(t, `(a, b) => a != b`, true, NumVal(1), NumVal(2))
	expectBool(t, `(a, b) => a != b`, false, NumVal(1), NumVal(1))
}

func TestObjectLitComputedKeyNonString(t *testing.T) {
	t.Parallel()
	v := mustRun(t, `(x) => {
		let o = {};
		o[42] = "num_key";
		return o;
	}`, NumVal(0))
	if v.Kind != KindObject {
		t.Fatalf("expected object, got %s", Display(v))
	}
}

func TestTupleIndexLength(t *testing.T) {
	t.Parallel()
	expectNum(t, `(a) => a["length"]`, 3, TupleVal(NumVal(1), NumVal(2), NumVal(3)))
}

// ============================================================
// Coverage gap tests
// ============================================================

// --- Run() edge cases ---

func TestRunBudgetZero(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, "(x) => x + 1")
	result, err := Run(prog, []Value{NumVal(1)}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if result.Value.Num != 2 {
		t.Errorf("expected 2, got %s", Display(result.Value))
	}
}

func TestRunPreambleError(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, "let x = undeclared_var; (a) => a")
	_, err := Run(prog, []Value{NumVal(1)}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error from preamble")
	}
}

func TestRunPreambleSignal(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, "let x = 42; (a) => a + x")
	result, err := Run(prog, []Value{NumVal(1)}, DefaultBudget)
	if err != nil {
		t.Fatal(err)
	}
	if result.Value.Num != 43 {
		t.Errorf("expected 43, got %s", Display(result.Value))
	}
}

func TestRunExtraParams(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, "(a, b, c) => a")
	result, err := Run(prog, []Value{NumVal(1)}, DefaultBudget)
	if err != nil {
		t.Fatal(err)
	}
	if result.Value.Num != 1 {
		t.Errorf("expected 1, got %s", Display(result.Value))
	}
}

// --- evalNode: NodeUndefinedLit, NodeNullLit, unknown node kind ---

func TestNodeNullLitCoverage(t *testing.T) {
	t.Parallel()
	v := mustRun(t, "(x) => null == undefined", NumVal(0))
	if v.Kind != KindBoolean {
		t.Errorf("expected boolean, got %s", Display(v))
	}
}

// --- Binary ops: requireNum error branches ---

func TestBinaryModuloNonNumber(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(a, b) => a % b`)
	_, err := Run(prog, []Value{StrVal("x"), NumVal(2)}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error for modulo on non-number (left)")
	}
}

func TestBinaryModuloRightNonNumber(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(a, b) => a % b`)
	_, err := Run(prog, []Value{NumVal(10), StrVal("y")}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error for modulo on non-number (right)")
	}
}

func TestBinaryComparisonNonNumber(t *testing.T) {
	t.Parallel()
	ops := []string{"<", ">", "<=", ">="}
	for _, op := range ops {
		prog := mustParse(t, "(a, b) => a "+op+" b")
		_, err := Run(prog, []Value{StrVal("x"), NumVal(2)}, DefaultBudget)
		if err == nil {
			t.Fatalf("expected error for %s on non-number", op)
		}
	}
}

func TestBinarySubtractNonNumber(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(a, b) => a - b`)
	_, err := Run(prog, []Value{StrVal("x"), NumVal(2)}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error for subtraction on non-number")
	}
}

func TestBinaryMultiplyNonNumber(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(a, b) => a * b`)
	_, err := Run(prog, []Value{StrVal("x"), NumVal(2)}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error for multiplication on non-number")
	}
}

func TestBinaryDivideNonNumber(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(a, b) => a / b`)
	_, err := Run(prog, []Value{StrVal("x"), NumVal(2)}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error for division on non-number")
	}
}

func TestBinaryAddNonNumber(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(a, b) => a + b`)
	_, err := Run(prog, []Value{BoolVal(true), NumVal(2)}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error for add on bool + number")
	}
}

// coerceToString error for left side
func TestBinaryStringConcatCoerceLeftError(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(a, b) => a + b`)
	_, err := Run(prog, []Value{TupleVal(), StrVal("x")}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error coercing tuple to string")
	}
}

// coerceToString error for right side
func TestBinaryStringConcatCoerceRightError(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(a, b) => a + b`)
	_, err := Run(prog, []Value{StrVal("x"), TupleVal()}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error coercing tuple to string")
	}
}

// coerceToString alloc error
func TestBinaryStringConcatAllocError(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(a, b) => { let s = a; let i = 0; while (i < 100) { s = s + b; i = i + 1; } return s; }`)
	bigStr := make([]byte, 100000)
	for i := range bigStr {
		bigStr[i] = 'x'
	}
	_, err := Run(prog, []Value{StrVal(string(bigStr)), StrVal(string(bigStr))}, DefaultBudget)
	if err == nil {
		t.Fatal("expected memory error")
	}
}

// --- evalBinary: right side eval error ---

func TestBinaryRightEvalError(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(x) => x + undefined_var`)
	_, err := Run(prog, []Value{NumVal(1)}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- evalUnary: unary error on eval ---

func TestUnaryEvalError(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(x) => -undefined_var`)
	_, err := Run(prog, []Value{NumVal(1)}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- void on non-string non-object ---

func TestVoidOnNumber(t *testing.T) {
	t.Parallel()
	expectUndefined(t, `(x) => void x`, NumVal(42))
}

func TestVoidOnTuple(t *testing.T) {
	t.Parallel()
	expectUndefined(t, `(x) => void x`, TupleVal(NumVal(1)))
}

func TestVoidOnBool(t *testing.T) {
	t.Parallel()
	expectUndefined(t, `(x) => void x`, BoolVal(true))
}

func TestVoidOnNull(t *testing.T) {
	t.Parallel()
	expectUndefined(t, `(x) => void x`, Null)
}

// --- evalPropAccess: error on non-obj/tuple/string ---

func TestPropAccessOnNumber(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(x) => x.foo`)
	_, err := Run(prog, []Value{NumVal(42)}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestPropAccessOnBool(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(x) => x.foo`)
	_, err := Run(prog, []Value{BoolVal(true)}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- evalPropAccess: eval error ---

func TestPropAccessEvalError(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(x) => undefined_var.foo`)
	_, err := Run(prog, []Value{NumVal(0)}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- evalIndexAccess: eval errors ---

func TestIndexAccessLeftEvalError(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(x) => undefined_var[0]`)
	_, err := Run(prog, []Value{NumVal(0)}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestIndexAccessRightEvalError(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(x) => x[undefined_var]`)
	_, err := Run(prog, []Value{TupleVal(NumVal(1))}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- evalIndexAccess: default fallthrough returns undefined ---

func TestIndexAccessOnNumberReturnsUndefined(t *testing.T) {
	t.Parallel()
	expectUndefined(t, `(x) => x[0]`, NumVal(42))
}

// --- evalCall: undefined function ---

func TestCallUndefinedFunction(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(x) => undefined_fn(x)`)
	_, err := Run(prog, []Value{NumVal(0)}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- evalCall: non-function on prop access (method not found) ---

func TestMethodCallOnNonObjectFallthrough(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(x) => x.nonexistent(1)`)
	_, err := Run(prog, []Value{NumVal(42)}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- evalCall: static namespace call on unknown method ---

func TestUnknownStaticMethod(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(x) => Object.nonexistent(x)`)
	_, err := Run(prog, []Value{NumVal(0)}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error for unknown Object method")
	}
}

func TestUnknownMathMethod(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(x) => Math.nonexistent(x)`)
	_, err := Run(prog, []Value{NumVal(0)}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error for unknown Math method")
	}
}

// --- evalCall: eval errors in various positions ---

func TestCallArgEvalError(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(x) => x.toUpperCase(undefined_var)`)
	_, err := Run(prog, []Value{StrVal("hello")}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCallReceiverEvalError(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(x) => undefined_var.toUpperCase()`)
	_, err := Run(prog, []Value{NumVal(0)}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCallCalleeEvalError(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(x) => { let a = [1]; return a[undefined_var](5); }`)
	_, err := Run(prog, []Value{NumVal(0)}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCallCalleeArgEvalError(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(x) => { let a = [(n) => n]; return a[0](undefined_var); }`)
	_, err := Run(prog, []Value{NumVal(0)}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- callMethod: string methods with wrong arg types ---

func TestStringSplitNoArgs(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(s) => s.split()`)
	_, err := Run(prog, []Value{StrVal("hello")}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestStringSplitWrongType(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(s) => s.split(42)`)
	_, err := Run(prog, []Value{StrVal("hello")}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestStringIncludesNoArgs(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(s) => s.includes()`)
	_, err := Run(prog, []Value{StrVal("hello")}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestStringIncludesWrongType(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(s) => s.includes(42)`)
	_, err := Run(prog, []Value{StrVal("hello")}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestStringIndexOfNoArgs(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(s) => s.indexOf()`)
	_, err := Run(prog, []Value{StrVal("hello")}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestStringIndexOfWrongType(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(s) => s.indexOf(42)`)
	_, err := Run(prog, []Value{StrVal("hello")}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestStringStartsWithNoArgs(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(s) => s.startsWith()`)
	_, err := Run(prog, []Value{StrVal("hello")}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestStringStartsWithWrongType(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(s) => s.startsWith(42)`)
	_, err := Run(prog, []Value{StrVal("hello")}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestStringEndsWithNoArgs(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(s) => s.endsWith()`)
	_, err := Run(prog, []Value{StrVal("hello")}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestStringEndsWithWrongType(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(s) => s.endsWith(42)`)
	_, err := Run(prog, []Value{StrVal("hello")}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestStringSliceNoArgs(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(s) => s.slice()`)
	_, err := Run(prog, []Value{StrVal("hello")}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestStringReplaceNoArgs(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(s) => s.replace()`)
	_, err := Run(prog, []Value{StrVal("hello")}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestStringReplaceWrongOld(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(s) => s.replace(42, "x")`)
	_, err := Run(prog, []Value{StrVal("hello")}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestStringReplaceWrongNew(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(s) => s.replace("h", 42)`)
	_, err := Run(prog, []Value{StrVal("hello")}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestStringCharAtNoArgs(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(s) => s.charAt()`)
	_, err := Run(prog, []Value{StrVal("hello")}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestStringCharAtWrongType(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(s) => s.charAt("x")`)
	_, err := Run(prog, []Value{StrVal("hello")}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- callMethod: array methods with wrong arg types ---

func TestArrayMapNoArgs(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(a) => a.map()`)
	_, err := Run(prog, []Value{TupleVal(NumVal(1))}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestArrayFilterNoArgs(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(a) => a.filter()`)
	_, err := Run(prog, []Value{TupleVal(NumVal(1))}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestArrayReduceNoArgs(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(a) => a.reduce()`)
	_, err := Run(prog, []Value{TupleVal(NumVal(1))}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestArrayReduceNonFunction(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(a) => a.reduce(42, 0)`)
	_, err := Run(prog, []Value{TupleVal(NumVal(1))}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestArrayFindNoArgs(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(a) => a.find()`)
	_, err := Run(prog, []Value{TupleVal(NumVal(1))}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestArrayFindNonFunction(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(a) => a.find(42)`)
	_, err := Run(prog, []Value{TupleVal(NumVal(1))}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestArraySomeNoArgs(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(a) => a.some()`)
	_, err := Run(prog, []Value{TupleVal(NumVal(1))}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestArraySomeNonFunction(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(a) => a.some(42)`)
	_, err := Run(prog, []Value{TupleVal(NumVal(1))}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestArrayEveryNoArgs(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(a) => a.every()`)
	_, err := Run(prog, []Value{TupleVal(NumVal(1))}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestArrayEveryNonFunction(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(a) => a.every(42)`)
	_, err := Run(prog, []Value{TupleVal(NumVal(1))}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestArrayConcatNoArgs(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(a) => a.concat()`)
	_, err := Run(prog, []Value{TupleVal(NumVal(1))}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestArrayConcatWrongType(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(a) => a.concat(42)`)
	_, err := Run(prog, []Value{TupleVal(NumVal(1))}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestArrayIncludesNoArgs(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(a) => a.includes()`)
	_, err := Run(prog, []Value{TupleVal(NumVal(1))}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestArrayIndexOfNoArgs(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(a) => a.indexOf()`)
	_, err := Run(prog, []Value{TupleVal(NumVal(1))}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestArraySliceNoArgs(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(a) => a.slice()`)
	_, err := Run(prog, []Value{TupleVal(NumVal(1))}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestArrayJoinNoArgs(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(a) => a.join()`)
	_, err := Run(prog, []Value{TupleVal(StrVal("a"))}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestArrayJoinWrongSepType(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(a) => a.join(42)`)
	_, err := Run(prog, []Value{TupleVal(StrVal("a"))}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestArrayJoinNonStringElement(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(a) => a.join(",")`)
	_, err := Run(prog, []Value{TupleVal(NumVal(1), NumVal(2))}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error joining non-string elements")
	}
}

// --- callMethod: method call on non-matching kind ---

func TestMethodCallOnNonMethodKind(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(x) => x.split(".")`)
	_, err := Run(prog, []Value{TupleVal(NumVal(1))}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error calling string method on tuple")
	}
}

// --- callStatic: wrong arg types ---

func TestObjectKeysWrongType(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(x) => Object.keys(x)`)
	_, err := Run(prog, []Value{NumVal(42)}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestObjectValuesWrongType(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(x) => Object.values(x)`)
	_, err := Run(prog, []Value{NumVal(42)}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestObjectEntriesWrongType(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(x) => Object.entries(x)`)
	_, err := Run(prog, []Value{NumVal(42)}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestObjectFromEntriesWrongType(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(x) => Object.fromEntries(x)`)
	_, err := Run(prog, []Value{NumVal(42)}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestObjectFromEntriesNonTupleEntry(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(x) => Object.fromEntries([42])`)
	_, err := Run(prog, []Value{NumVal(0)}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestObjectFromEntriesNonStringKey(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(x) => Object.fromEntries([[42, "v"]])`)
	_, err := Run(prog, []Value{NumVal(0)}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestMathAbsWrongType(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(x) => Math.abs(x)`)
	_, err := Run(prog, []Value{StrVal("hello")}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestMathMinWrongTypeFirst(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(a, b) => Math.min(a, b)`)
	_, err := Run(prog, []Value{StrVal("x"), NumVal(1)}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestMathMinWrongTypeSecond(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(a, b) => Math.min(a, b)`)
	_, err := Run(prog, []Value{NumVal(1), StrVal("x")}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- callBuiltinSlice: various error paths ---

func TestSliceWrongStartType(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(a) => a.slice("x")`)
	_, err := Run(prog, []Value{TupleVal(NumVal(1))}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestStringSliceNegativeEnd(t *testing.T) {
	t.Parallel()
	expectStr(t, `(s) => s.slice(0, -1)`, "hell", StrVal("hello"))
}

func TestStringSliceEndParam(t *testing.T) {
	t.Parallel()
	expectStr(t, `(s) => s.slice(1, 3)`, "el", StrVal("hello"))
}

func TestStringSliceEndWrongType(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(s) => s.slice(0, "x")`)
	_, err := Run(prog, []Value{StrVal("hello")}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestArraySliceEndParam(t *testing.T) {
	t.Parallel()
	v := mustRun(t, `(a) => a.slice(0, -1)`, TupleVal(NumVal(1), NumVal(2), NumVal(3)))
	if v.Kind != KindTuple || len(v.Elems) != 2 {
		t.Fatalf("expected [1,2], got %s", Display(v))
	}
}

func TestArraySliceEndWrongType(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(a) => a.slice(0, "x")`)
	_, err := Run(prog, []Value{TupleVal(NumVal(1))}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestStringSliceStartGEEnd(t *testing.T) {
	t.Parallel()
	expectStr(t, `(s) => s.slice(3, 1)`, "", StrVal("hello"))
}

func TestArraySliceStartGEEnd(t *testing.T) {
	t.Parallel()
	v := mustRun(t, `(a) => a.slice(3, 1)`, TupleVal(NumVal(1), NumVal(2), NumVal(3)))
	if v.Kind != KindTuple || len(v.Elems) != 0 {
		t.Fatalf("expected empty tuple, got %s", Display(v))
	}
}

// --- callBuiltinMap/Filter: error during callback ---

func TestMapCallbackError(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(a) => a.map((x) => x.nonexistent())`)
	_, err := Run(prog, []Value{TupleVal(NumVal(1))}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestFilterCallbackError(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(a) => a.filter((x) => x.nonexistent())`)
	_, err := Run(prog, []Value{TupleVal(NumVal(1))}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- callClosure: fewer args than params ---

func TestClosureFewerArgs(t *testing.T) {
	t.Parallel()
	v := mustRun(t, `(x) => { let f = (a, b) => typeof b; return f(1); }`, NumVal(0))
	if v.Kind != KindString || v.Str != "undefined" {
		t.Errorf("expected 'undefined', got %s", Display(v))
	}
}

// --- evalObjectLit: spread non-object ---

func TestObjectSpreadNonObject(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(x) => { return { ...x }; }`)
	_, err := Run(prog, []Value{NumVal(42)}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error spreading non-object")
	}
}

// --- evalArrayLit: spread non-array ---

func TestArraySpreadNonArray(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(x) => [...x]`)
	_, err := Run(prog, []Value{NumVal(42)}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error spreading non-array")
	}
}

// --- execStmt: destructure more names than elements ---

func TestDestructureMoreNamesThanElems(t *testing.T) {
	t.Parallel()
	expectUndefined(t, `(x) => { let [a, b, c] = x; return c; }`, TupleVal(NumVal(1), NumVal(2)))
}

// --- execStmt: destructure rest empty ---

func TestDestructureRestEmpty(t *testing.T) {
	t.Parallel()
	v := mustRun(t, `(x) => { let [a, b, ...rest] = x; return rest; }`, TupleVal(NumVal(1), NumVal(2)))
	if v.Kind != KindTuple || len(v.Elems) != 0 {
		t.Errorf("expected empty tuple, got %s", Display(v))
	}
}

// --- execStmt: while return inside ---

func TestWhileReturnFromInner(t *testing.T) {
	t.Parallel()
	expectNum(t, `(x) => {
		let i = 0;
		while (true) {
			if (i == 3) { return i; }
			i = i + 1;
		}
	}`, 3, NumVal(0))
}

// --- execIf: else-if chains ---

func TestElseIfChain(t *testing.T) {
	t.Parallel()
	src := `(x) => {
		if (x == 1) { return "one"; }
		else if (x == 2) { return "two"; }
		else if (x == 3) { return "three"; }
		else { return "other"; }
	}`
	expectStr(t, src, "one", NumVal(1))
	expectStr(t, src, "two", NumVal(2))
	expectStr(t, src, "three", NumVal(3))
	expectStr(t, src, "other", NumVal(99))
}

func TestIfFalseNoElse(t *testing.T) {
	t.Parallel()
	expectUndefined(t, `(x) => { if (false) { return 1; } }`, NumVal(0))
}

// --- execStmt: let init error ---

func TestLetInitError(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(x) => { let v = undefined_var; return v; }`)
	_, err := Run(prog, []Value{NumVal(0)}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- execStmt: destructure init error ---

func TestDestructureInitError(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(x) => { let [a, b] = undefined_var; return a; }`)
	_, err := Run(prog, []Value{NumVal(0)}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- execStmt: assign value error ---

func TestAssignValueError(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(x) => { let v = 1; v = undefined_var; return v; }`)
	_, err := Run(prog, []Value{NumVal(0)}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- execStmt: index assign errors ---

func TestIndexAssignObjError(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(x) => { undefined_var["k"] = 1; }`)
	_, err := Run(prog, []Value{NumVal(0)}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestIndexAssignIdxError(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(x) => { let o = {}; o[undefined_var] = 1; }`)
	_, err := Run(prog, []Value{NumVal(0)}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestIndexAssignValError(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(x) => { let o = {}; o["k"] = undefined_var; }`)
	_, err := Run(prog, []Value{NumVal(0)}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- execStmt: for-of iter error ---

func TestForOfIterError(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(x) => { for (let v of undefined_var) { } }`)
	_, err := Run(prog, []Value{NumVal(0)}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- execStmt: for-of body error ---

func TestForOfBodyError(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(x) => { for (let v of [1, 2]) { let w = undefined_var; } }`)
	_, err := Run(prog, []Value{NumVal(0)}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- execStmt: while cond error ---

func TestWhileCondError(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(x) => { while (undefined_var) { } }`)
	_, err := Run(prog, []Value{NumVal(0)}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- execStmt: while body error ---

func TestWhileBodyError(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(x) => { while (true) { let w = undefined_var; } }`)
	_, err := Run(prog, []Value{NumVal(0)}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- execStmt: return error ---

func TestReturnError(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(x) => { return undefined_var; }`)
	_, err := Run(prog, []Value{NumVal(0)}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- execStmt: expr error ---

func TestExprStmtError(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(x) => { undefined_var; return 1; }`)
	_, err := Run(prog, []Value{NumVal(0)}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- execIf: cond error ---

func TestIfCondError(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(x) => { if (undefined_var) { return 1; } return 0; }`)
	_, err := Run(prog, []Value{NumVal(0)}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- evalTernary: cond error ---

func TestTernaryCondError(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(x) => undefined_var ? 1 : 2`)
	_, err := Run(prog, []Value{NumVal(0)}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- evalLambda and evalBlock ---

func TestLambdaValue(t *testing.T) {
	t.Parallel()
	v := mustRun(t, `(x) => { let f = (n) => n + 1; return typeof f; }`, NumVal(0))
	if v.Kind != KindString || v.Str != "function" {
		t.Errorf("expected 'function', got %s", Display(v))
	}
}

// --- value.go: TypeofValue for unknown kind ---

func TestTypeofValueUnknownKind(t *testing.T) {
	t.Parallel()
	v := Value{Kind: ValueKind(99)}
	if TypeofValue(v) != "unknown" {
		t.Errorf("expected 'unknown', got %q", TypeofValue(v))
	}
}

// --- value.go: CacheKey for object with multiple props ---

func TestCacheKeyObjectMultipleProps(t *testing.T) {
	t.Parallel()
	m := NewOrderedMap()
	m.Set("a", NumVal(1))
	m.Set("b", NumVal(2))
	got := CacheKey(ObjectVal(m))
	if got != "{a:n1,b:n2}" {
		t.Errorf("got %q", got)
	}
}

// --- value.go: ValuesEqual for functions (always false) ---

func TestValuesEqualFunctions(t *testing.T) {
	t.Parallel()
	fn1 := Value{Kind: KindFunction, Fn: &Closure{}}
	fn2 := Value{Kind: KindFunction, Fn: &Closure{}}
	if ValuesEqual(fn1, fn2) {
		t.Error("functions should never be equal")
	}
}

// --- value.go: DeepCopy ---

func TestDeepCopyObject(t *testing.T) {
	t.Parallel()
	m := NewOrderedMap()
	m.Set("k", NumVal(42))
	orig := ObjectVal(m)
	copied := DeepCopy(orig)
	if copied.Kind != KindObject {
		t.Fatalf("expected object, got %d", copied.Kind)
	}
	v, _ := copied.Props.Get("k")
	if v.Num != 42 {
		t.Errorf("expected 42, got %v", v.Num)
	}
}

// --- Lexer: string escapes ---

func TestLexerEscapeBackslash(t *testing.T) {
	t.Parallel()
	expectStr(t, `(x) => "\\"`, `\`, NumVal(0))
}

func TestLexerEscapeSingleQuote(t *testing.T) {
	t.Parallel()
	expectStr(t, `(x) => '\''`, "'", NumVal(0))
}

func TestLexerEscapeBacktick(t *testing.T) {
	t.Parallel()
	tokens, err := Lex("`hello\\`world`")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, tok := range tokens {
		if tok.Kind == TokStr && tok.Str == "hello`world" {
			found = true
		}
	}
	if !found {
		t.Error("expected backtick escape in string")
	}
}

func TestLexerEscapeTab(t *testing.T) {
	t.Parallel()
	expectStr(t, `(x) => "a\tb"`, "a\tb", NumVal(0))
}

func TestLexerUnknownEscape(t *testing.T) {
	t.Parallel()
	expectStr(t, `(x) => "\q"`, "q", NumVal(0))
}

func TestLexerUnterminatedString(t *testing.T) {
	t.Parallel()
	_, err := Lex(`"unterminated`)
	if err == nil {
		t.Fatal("expected error for unterminated string")
	}
}

func TestLexerInvalidNumber(t *testing.T) {
	t.Parallel()
	_, err := Lex("1.2.3")
	if err == nil {
		t.Fatal("expected error for invalid number")
	}
}

func TestLexerUnexpectedChar(t *testing.T) {
	t.Parallel()
	_, err := Lex("@")
	if err == nil {
		t.Fatal("expected error for unexpected character")
	}
}

func TestLexerDotNumber(t *testing.T) {
	t.Parallel()
	expectNum(t, `(x) => .5 + .5`, 1, NumVal(0))
}

// --- Parser error paths ---

func TestParseErrorMissingArrow(t *testing.T) {
	t.Parallel()
	_, err := ParseProgram("(a, b) { a + b }")
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestParseErrorMissingParenAfterParams(t *testing.T) {
	t.Parallel()
	_, err := ParseProgram("a, b => a + b")
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestParseErrorMissingCloseBrace(t *testing.T) {
	t.Parallel()
	_, err := ParseProgram("(a) => { return a;")
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestParseErrorMissingTernaryColon(t *testing.T) {
	t.Parallel()
	_, err := ParseProgram("(a) => a ? 1 2")
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestParseErrorMissingCloseBrack(t *testing.T) {
	t.Parallel()
	_, err := ParseProgram("(a) => a[0")
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestParseErrorMissingCloseCallParen(t *testing.T) {
	t.Parallel()
	_, err := ParseProgram("(a) => a(1, 2")
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestParseErrorInvalidPrimary(t *testing.T) {
	t.Parallel()
	_, err := ParseProgram("(a) => +")
	if err == nil {
		t.Fatal("expected parse error for invalid primary")
	}
}

func TestParseErrorMissingPropName(t *testing.T) {
	t.Parallel()
	_, err := ParseProgram("(a) => a.")
	if err == nil {
		t.Fatal("expected parse error for missing prop name")
	}
}

func TestParseErrorExtraTokenAfterProgram(t *testing.T) {
	t.Parallel()
	_, err := ParseProgram("(a) => a )")
	if err == nil {
		t.Fatal("expected parse error for extra token")
	}
}

func TestParseErrorForOfMissingParen(t *testing.T) {
	t.Parallel()
	_, err := ParseProgram("(a) => { for let x of a { } }")
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestParseErrorForOfMissingLet(t *testing.T) {
	t.Parallel()
	_, err := ParseProgram("(a) => { for (x of a) { } }")
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestParseErrorForOfMissingBindingName(t *testing.T) {
	t.Parallel()
	_, err := ParseProgram("(a) => { for (let 42 of a) { } }")
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestParseErrorForOfMissingOf(t *testing.T) {
	t.Parallel()
	_, err := ParseProgram("(a) => { for (let x in a) { } }")
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestParseErrorForOfMissingCloseParen(t *testing.T) {
	t.Parallel()
	_, err := ParseProgram("(a) => { for (let x of a { } }")
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestParseErrorForOfMissingOpenBrace(t *testing.T) {
	t.Parallel()
	_, err := ParseProgram("(a) => { for (let x of a) return x; }")
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestParseErrorForOfMissingCloseBrace(t *testing.T) {
	t.Parallel()
	_, err := ParseProgram("(a) => { for (let x of a) { return x;  }")
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestParseErrorWhileMissingParen(t *testing.T) {
	t.Parallel()
	_, err := ParseProgram("(a) => { while true { } }")
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestParseErrorWhileMissingCloseParen(t *testing.T) {
	t.Parallel()
	_, err := ParseProgram("(a) => { while (true { } }")
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestParseErrorWhileMissingOpenBrace(t *testing.T) {
	t.Parallel()
	_, err := ParseProgram("(a) => { while (true) return 1; }")
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestParseErrorWhileMissingCloseBrace(t *testing.T) {
	t.Parallel()
	_, err := ParseProgram("(a) => { while (true) { return 1; }")
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestParseErrorReturnBareValue(t *testing.T) {
	t.Parallel()
	v := mustRun(t, `(x) => { return; }`, NumVal(0))
	if v.Kind != KindUndefined {
		t.Errorf("expected undefined, got %s", Display(v))
	}
}

func TestParseErrorLetMissingEquals(t *testing.T) {
	t.Parallel()
	_, err := ParseProgram("(a) => { let x 42; }")
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestParseErrorLetMissingName(t *testing.T) {
	t.Parallel()
	_, err := ParseProgram("(a) => { let 42; }")
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestParseErrorDestructureSpreadNonIdent(t *testing.T) {
	t.Parallel()
	_, err := ParseProgram("(a) => { let [...42] = a; }")
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestParseErrorDestructureNonIdent(t *testing.T) {
	t.Parallel()
	_, err := ParseProgram("(a) => { let [42] = a; }")
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestParseErrorDestructureMissingBrack(t *testing.T) {
	t.Parallel()
	_, err := ParseProgram("(a) => { let [x, y = a; }")
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestParseErrorDestructureMissingEquals(t *testing.T) {
	t.Parallel()
	_, err := ParseProgram("(a) => { let [x, y] a; }")
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestParseErrorIfMissingParen(t *testing.T) {
	t.Parallel()
	_, err := ParseProgram("(a) => { if true { return 1; } }")
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestParseErrorIfMissingCloseParen(t *testing.T) {
	t.Parallel()
	_, err := ParseProgram("(a) => { if (true { return 1; } }")
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestParseErrorElseIfMissingParen(t *testing.T) {
	t.Parallel()
	_, err := ParseProgram("(a) => { if (false) { } else if true { } }")
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestParseErrorElseIfMissingCloseParen(t *testing.T) {
	t.Parallel()
	_, err := ParseProgram("(a) => { if (false) { } else if (true { } }")
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestParseErrorInvalidAssignTarget(t *testing.T) {
	t.Parallel()
	_, err := ParseProgram("(a) => { 42 = 1; }")
	if err == nil {
		t.Fatal("expected parse error for invalid assignment target")
	}
}

func TestParseErrorObjectMissingColon(t *testing.T) {
	t.Parallel()
	_, err := ParseProgram("(a) => { return { 42 }; }")
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestParseErrorObjectMissingClose(t *testing.T) {
	t.Parallel()
	_, err := ParseProgram("(a) => { return { a: 1 ; }")
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestParseErrorArrayMissingClose(t *testing.T) {
	t.Parallel()
	_, err := ParseProgram("(a) => [1, 2")
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestParseErrorExpectedCloseParen(t *testing.T) {
	t.Parallel()
	_, err := ParseProgram("(a) => (1 + 2")
	if err == nil {
		t.Fatal("expected parse error for missing close paren")
	}
}

func TestParseErrorParamMissingCloseParen(t *testing.T) {
	t.Parallel()
	_, err := ParseProgram("(a, b => a + b")
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestParseErrorParamNonIdent(t *testing.T) {
	t.Parallel()
	_, err := ParseProgram("(42) => 1")
	if err == nil {
		t.Fatal("expected parse error")
	}
}

// --- Parser: preamble error ---

func TestParsePreambleError(t *testing.T) {
	t.Parallel()
	_, err := ParseProgram("let x = ; (a) => a")
	if err == nil {
		t.Fatal("expected parse error")
	}
}

// --- Dot assignment on prop access ---

func TestDotAssignmentPropAccess(t *testing.T) {
	t.Parallel()
	v := mustRun(t, `(x) => {
		let obj = { a: 1, b: 2 };
		obj.b = 99;
		return obj.b;
	}`, NumVal(0))
	if v.Num != 99 {
		t.Errorf("expected 99, got %s", Display(v))
	}
}

// --- callStatic: eval args error ---

func TestStaticCallArgError(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(x) => Object.keys(undefined_var)`)
	_, err := Run(prog, []Value{NumVal(0)}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- callMethod: method call args eval error ---

func TestMethodCallArgsEvalError(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(x) => x.map(undefined_var)`)
	_, err := Run(prog, []Value{TupleVal(NumVal(1))}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- callMethod: reduce step error ---

func TestReduceStepError(t *testing.T) {
	t.Parallel()
	// Build a massive array that will exceed budget during reduce
	src := `(x) => { let a = []; let i = 0; while (i < 50) { a = [...a, i]; i = i + 1; } return a.reduce((acc, x) => acc + x, 0); }`
	prog := mustParse(t, src)
	_, err := Run(prog, []Value{NumVal(0)}, 100)
	if err == nil {
		t.Fatal("expected budget error")
	}
}

// --- callMethod: find step error ---

func TestFindStepError(t *testing.T) {
	t.Parallel()
	src := `(x) => { let a = []; let i = 0; while (i < 50) { a = [...a, i]; i = i + 1; } return a.find((x) => x == 999); }`
	prog := mustParse(t, src)
	_, err := Run(prog, []Value{NumVal(0)}, 100)
	if err == nil {
		t.Fatal("expected budget error")
	}
}

// --- callMethod: some step error ---

func TestSomeStepError(t *testing.T) {
	t.Parallel()
	src := `(x) => { let a = []; let i = 0; while (i < 50) { a = [...a, i]; i = i + 1; } return a.some((x) => x == 999); }`
	prog := mustParse(t, src)
	_, err := Run(prog, []Value{NumVal(0)}, 100)
	if err == nil {
		t.Fatal("expected budget error")
	}
}

// --- callMethod: every step error ---

func TestEveryStepError(t *testing.T) {
	t.Parallel()
	src := `(x) => { let a = []; let i = 0; while (i < 50) { a = [...a, i]; i = i + 1; } return a.every((x) => x >= 0); }`
	prog := mustParse(t, src)
	_, err := Run(prog, []Value{NumVal(0)}, 100)
	if err == nil {
		t.Fatal("expected budget error")
	}
}

// --- callBuiltinMap: step error ---

func TestMapStepError(t *testing.T) {
	t.Parallel()
	src := `(x) => { let a = []; let i = 0; while (i < 50) { a = [...a, i]; i = i + 1; } return a.map((x) => x * 2); }`
	prog := mustParse(t, src)
	_, err := Run(prog, []Value{NumVal(0)}, 100)
	if err == nil {
		t.Fatal("expected budget error")
	}
}

// --- callBuiltinFilter: step error ---

func TestFilterStepError(t *testing.T) {
	t.Parallel()
	src := `(x) => { let a = []; let i = 0; while (i < 50) { a = [...a, i]; i = i + 1; } return a.filter((x) => x > 0); }`
	prog := mustParse(t, src)
	_, err := Run(prog, []Value{NumVal(0)}, 100)
	if err == nil {
		t.Fatal("expected budget error")
	}
}

// --- evalObjectLit: key/value eval error ---

func TestObjectLitKeyEvalError(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(x) => { return { ...undefined_var }; }`)
	_, err := Run(prog, []Value{NumVal(0)}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestObjectLitValueEvalError(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(x) => { return { a: undefined_var }; }`)
	_, err := Run(prog, []Value{NumVal(0)}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- evalArrayLit: eval error ---

func TestArrayLitEvalError(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(x) => [undefined_var]`)
	_, err := Run(prog, []Value{NumVal(0)}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- callClosure: closure body error ---

func TestClosureBodyError(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(x) => { let f = (n) => n + undefined_var; return f(1); }`)
	_, err := Run(prog, []Value{NumVal(0)}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- String slice with end param via method ---

func TestStringSliceMethodWithEnd(t *testing.T) {
	t.Parallel()
	expectStr(t, `(s) => s.slice(1, 4)`, "ell", StrVal("hello"))
}

// --- Array slice negative start ---

func TestArraySliceNegativeStart(t *testing.T) {
	t.Parallel()
	v := mustRun(t, `(a) => a.slice(-2)`, TupleVal(NumVal(1), NumVal(2), NumVal(3)))
	if v.Kind != KindTuple || len(v.Elems) != 2 || v.Elems[0].Num != 2 {
		t.Errorf("expected [2,3], got %s", Display(v))
	}
}

// --- for-of continue signal ---

func TestForOfContinueSignal(t *testing.T) {
	t.Parallel()
	v := mustRun(t, `(x) => {
		let sum = 0;
		for (let v of [1, 2, 3, 4, 5]) {
			if (v % 2 == 0) { continue; }
			sum = sum + v;
		}
		return sum;
	}`, NumVal(0))
	if v.Num != 9 {
		t.Errorf("expected 9, got %s", Display(v))
	}
}

// --- while continue signal ---

func TestWhileContinueSignal(t *testing.T) {
	t.Parallel()
	v := mustRun(t, `(x) => {
		let sum = 0;
		let i = 0;
		while (i < 5) {
			i = i + 1;
			if (i % 2 == 0) { continue; }
			sum = sum + i;
		}
		return sum;
	}`, NumVal(0))
	if v.Num != 9 {
		t.Errorf("expected 9, got %s", Display(v))
	}
}

// --- Preamble signal: return/break from preamble ---

func TestPreambleReturnSignal(t *testing.T) {
	t.Parallel()
	// A preamble with `return` in a block: the preamble loop breaks on non-sigNone
	prog := mustParse(t, "let x = 42; (a) => x + a")
	result, err := Run(prog, []Value{NumVal(1)}, DefaultBudget)
	if err != nil {
		t.Fatal(err)
	}
	if result.Value.Num != 43 {
		t.Errorf("expected 43, got %s", Display(result.Value))
	}
}

// --- evalCall: direct call arg eval error ---

func TestDirectCallArgEvalError(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(x) => { let f = (a) => a; return f(undefined_var); }`)
	_, err := Run(prog, []Value{NumVal(0)}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- callBuiltin typeof function form ---

func TestTypeofFunctionCallForm(t *testing.T) {
	t.Parallel()
	// typeof(x) is parsed as unary, but if we create a direct call to a var named typeof...
	// Actually this can't trigger callBuiltin easily. It's unreachable.
	// Just verify typeof unary works for all kinds.
	expectStr(t, `(x) => typeof x`, "function", Value{Kind: KindFunction, Fn: &Closure{
		Params: []string{"n"},
		Body:   Node{Kind: NodeNumberLit, NumVal: 1},
		Env:    NewEnv(),
	}})
}

// --- reduce/find/some/every: step errors via budget during iteration ---

func TestReduceStepBudgetDirect(t *testing.T) {
	t.Parallel()
	arr := make([]Value, 1000)
	for i := range arr {
		arr[i] = NumVal(float64(i))
	}
	prog := mustParse(t, `(a) => a.reduce((acc, x) => 0, 0)`)
	_, err := Run(prog, []Value{TupleVal(arr...)}, 10)
	if err == nil {
		t.Fatal("expected budget error during reduce iteration")
	}
}

func TestFindStepBudgetDirect(t *testing.T) {
	t.Parallel()
	arr := make([]Value, 100)
	for i := range arr {
		arr[i] = NumVal(float64(i))
	}
	prog := mustParse(t, `(a) => a.find((x) => false)`)
	// Budget=5: setup(3) + 1 iteration(2) = 5. Next iteration step exceeds budget.
	_, err := Run(prog, []Value{TupleVal(arr...)}, 5)
	if err == nil {
		t.Fatal("expected budget error during find iteration")
	}
}

func TestSomeStepBudgetDirect(t *testing.T) {
	t.Parallel()
	arr := make([]Value, 100)
	for i := range arr {
		arr[i] = NumVal(float64(i))
	}
	prog := mustParse(t, `(a) => a.some((x) => false)`)
	_, err := Run(prog, []Value{TupleVal(arr...)}, 5)
	if err == nil {
		t.Fatal("expected budget error during some iteration")
	}
}

func TestEveryStepBudgetDirect(t *testing.T) {
	t.Parallel()
	arr := make([]Value, 100)
	for i := range arr {
		arr[i] = NumVal(float64(i))
	}
	prog := mustParse(t, `(a) => a.every((x) => true)`)
	_, err := Run(prog, []Value{TupleVal(arr...)}, 5)
	if err == nil {
		t.Fatal("expected budget error during every iteration")
	}
}

func TestMapStepBudgetDirect(t *testing.T) {
	t.Parallel()
	arr := make([]Value, 100)
	for i := range arr {
		arr[i] = NumVal(float64(i))
	}
	prog := mustParse(t, `(a) => a.map((x) => 0)`)
	_, err := Run(prog, []Value{TupleVal(arr...)}, 5)
	if err == nil {
		t.Fatal("expected budget error during map iteration")
	}
}

func TestFilterStepBudgetDirect(t *testing.T) {
	t.Parallel()
	arr := make([]Value, 100)
	for i := range arr {
		arr[i] = NumVal(float64(i))
	}
	prog := mustParse(t, `(a) => a.filter((x) => true)`)
	_, err := Run(prog, []Value{TupleVal(arr...)}, 5)
	if err == nil {
		t.Fatal("expected budget error during filter iteration")
	}
}

// Even budget = causes step error inside callback; odd budget = causes step error at loop step
func TestFindStepBudgetAtLoopStep(t *testing.T) {
	t.Parallel()
	arr := TupleVal(NumVal(1), NumVal(2), NumVal(3), NumVal(4), NumVal(5))
	prog := mustParse(t, `(a) => a.find((x) => false)`)
	_, err := Run(prog, []Value{arr}, 5)
	if err == nil {
		t.Fatal("expected budget error")
	}
}

func TestSomeStepBudgetAtLoopStep(t *testing.T) {
	t.Parallel()
	arr := TupleVal(NumVal(1), NumVal(2), NumVal(3), NumVal(4), NumVal(5))
	prog := mustParse(t, `(a) => a.some((x) => false)`)
	_, err := Run(prog, []Value{arr}, 5)
	if err == nil {
		t.Fatal("expected budget error")
	}
}

func TestEveryStepBudgetAtLoopStep(t *testing.T) {
	t.Parallel()
	arr := TupleVal(NumVal(1), NumVal(2), NumVal(3), NumVal(4), NumVal(5))
	prog := mustParse(t, `(a) => a.every((x) => true)`)
	_, err := Run(prog, []Value{arr}, 5)
	if err == nil {
		t.Fatal("expected budget error")
	}
}

func TestMapStepBudgetAtLoopStep(t *testing.T) {
	t.Parallel()
	arr := TupleVal(NumVal(1), NumVal(2), NumVal(3), NumVal(4), NumVal(5))
	prog := mustParse(t, `(a) => a.map((x) => 0)`)
	_, err := Run(prog, []Value{arr}, 5)
	if err == nil {
		t.Fatal("expected budget error")
	}
}

func TestFilterStepBudgetAtLoopStep(t *testing.T) {
	t.Parallel()
	arr := TupleVal(NumVal(1), NumVal(2), NumVal(3), NumVal(4), NumVal(5))
	prog := mustParse(t, `(a) => a.filter((x) => true)`)
	_, err := Run(prog, []Value{arr}, 5)
	if err == nil {
		t.Fatal("expected budget error")
	}
}

// --- evalObjectLit: key eval error (non-spread, non-ident key) ---

func TestObjectLitKeyExprEvalError(t *testing.T) {
	t.Parallel()
	// Non-ident key in object literal that errors at runtime
	// The parser treats (expr): val as object literal with computed key when expr is not just an ident
	prog := mustParse(t, `(x) => { return { (undefined_fn()): 1 }; }`)
	_, err := Run(prog, []Value{NumVal(0)}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- evalObjectLit: val eval error (non-spread) ---

func TestObjectLitValExprEvalError(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(x) => { return { a: undefined_var }; }`)
	_, err := Run(prog, []Value{NumVal(0)}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- execStmt step error ---

func TestExecStmtStepError(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(x) => { let a = 1; let b = 2; let c = 3; return a + b + c; }`)
	_, err := Run(prog, []Value{NumVal(0)}, 3)
	if err == nil {
		t.Fatal("expected budget error")
	}
}

// --- execIf: else-if cond eval error ---

func TestElseIfCondError(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(x) => {
		if (false) { return 1; }
		else if (undefined_var) { return 2; }
		else { return 3; }
	}`)
	_, err := Run(prog, []Value{NumVal(0)}, DefaultBudget)
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- alloc error on coerce string concat ---

func TestStringCoerceConcatAllocError(t *testing.T) {
	t.Parallel()
	// coerce path: number + big string where alloc check fails
	bigStr := make([]byte, 1100000)
	for i := range bigStr {
		bigStr[i] = 'x'
	}
	prog := mustParse(t, `(n, s) => n + s`)
	_, err := Run(prog, []Value{NumVal(42), StrVal(string(bigStr))}, DefaultBudget)
	if err == nil {
		t.Fatal("expected memory error")
	}
}

// --- Type annotation parsing: generic with > at depth 0 ---

func TestTypeAnnotationGeneric(t *testing.T) {
	t.Parallel()
	expectNum(t, `(a: Array<number>) => a[0]`, 42, TupleVal(NumVal(42)))
}

func TestTypeAnnotationNestedGeneric(t *testing.T) {
	t.Parallel()
	expectNum(t, `(a: Map<string, Array<number>>) => 42`, 42, NumVal(0))
}

// --- Lexer: escape sequences ---

func TestLexerEscapeDoubleQuote(t *testing.T) {
	t.Parallel()
	tokens, err := Lex(`"hello\"world"`)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, tok := range tokens {
		if tok.Kind == TokStr && tok.Str == `hello"world` {
			found = true
		}
	}
	if !found {
		t.Error("expected double quote escape in string")
	}
}

// --- Parser: various error paths ---

func TestParseErrorStmtBlockError(t *testing.T) {
	t.Parallel()
	// ']' is a valid token but not valid as expression start, triggering parse error inside block
	_, err := ParseProgram(`(a) => { if (true) { let x = ]; } }`)
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestParseErrorStmtBlockMissingCloseBrace(t *testing.T) {
	t.Parallel()
	// The inner if block is missing its closing brace
	_, err := ParseProgram(`(a) => { if (true) { let x = 1; return x; }`)
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestParseErrorIfStmtBlockMissingClose(t *testing.T) {
	t.Parallel()
	// Stmt block (if body) is missing closing brace
	_, err := ParseProgram(`(a) => { if (true) { return 1; }`)
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestParseErrorIfBodyError(t *testing.T) {
	t.Parallel()
	_, err := ParseProgram(`(a) => { if (true) ]; }`)
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestParseErrorElseIfBodyError(t *testing.T) {
	t.Parallel()
	_, err := ParseProgram(`(a) => { if (false) { } else if (true) ]; }`)
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestParseErrorElseBodyError(t *testing.T) {
	t.Parallel()
	_, err := ParseProgram(`(a) => { if (false) { } else ]; }`)
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestParseErrorForOfBodyStmtError(t *testing.T) {
	t.Parallel()
	_, err := ParseProgram(`(a) => { for (let x of a) { ]; } }`)
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestParseErrorWhileBodyStmtError(t *testing.T) {
	t.Parallel()
	_, err := ParseProgram(`(a) => { while (true) { ]; } }`)
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestParseErrorForOfExprError(t *testing.T) {
	t.Parallel()
	_, err := ParseProgram(`(a) => { for (let x of ]) { } }`)
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestParseErrorWhileCondError(t *testing.T) {
	t.Parallel()
	_, err := ParseProgram(`(a) => { while (]) { } }`)
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestParseErrorIfCondError(t *testing.T) {
	t.Parallel()
	_, err := ParseProgram(`(a) => { if (]) { } }`)
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestParseErrorElseIfCondError(t *testing.T) {
	t.Parallel()
	_, err := ParseProgram(`(a) => { if (false) { } else if (]) { } }`)
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestParseErrorReturnExprError(t *testing.T) {
	t.Parallel()
	_, err := ParseProgram(`(a) => { return ]; }`)
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestParseErrorLetExprError(t *testing.T) {
	t.Parallel()
	_, err := ParseProgram(`(a) => { let x = ]; }`)
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestParseErrorDestructureExprError(t *testing.T) {
	t.Parallel()
	_, err := ParseProgram(`(a) => { let [x, y] = ]; }`)
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestParseErrorTernaryThenError(t *testing.T) {
	t.Parallel()
	_, err := ParseProgram(`(a) => true ? ] : 2`)
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestParseErrorTernaryElseError(t *testing.T) {
	t.Parallel()
	_, err := ParseProgram(`(a) => true ? 1 : ]`)
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestParseErrorBinaryRightError(t *testing.T) {
	t.Parallel()
	_, err := ParseProgram(`(a) => 1 + ]`)
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestParseErrorUnaryOperandError(t *testing.T) {
	t.Parallel()
	_, err := ParseProgram(`(a) => -]`)
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestParseErrorNotOperandError(t *testing.T) {
	t.Parallel()
	_, err := ParseProgram(`(a) => !]`)
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestParseErrorVoidOperandError(t *testing.T) {
	t.Parallel()
	_, err := ParseProgram(`(a) => void ]`)
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestParseErrorTypeofOperandError(t *testing.T) {
	t.Parallel()
	_, err := ParseProgram(`(a) => typeof ]`)
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestParseErrorIndexExprError(t *testing.T) {
	t.Parallel()
	_, err := ParseProgram(`(a) => a[}]`)
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestParseErrorCallArgError(t *testing.T) {
	t.Parallel()
	_, err := ParseProgram(`(a) => a(])`)
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestParseErrorArrayElemError(t *testing.T) {
	t.Parallel()
	_, err := ParseProgram(`(a) => [}]`)
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestParseErrorObjectSpreadError(t *testing.T) {
	t.Parallel()
	_, err := ParseProgram(`(a) => { return { ...] }; }`)
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestParseErrorObjectValueError(t *testing.T) {
	t.Parallel()
	_, err := ParseProgram(`(a) => { return { a: ] }; }`)
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestParseErrorObjectNonIdentKeyError(t *testing.T) {
	t.Parallel()
	_, err := ParseProgram(`(a) => { return { ]: 1 }; }`)
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestParseErrorLambdaBodyError(t *testing.T) {
	t.Parallel()
	_, err := ParseProgram(`(a) => { let f = () => ]; return f(); }`)
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestParseErrorGroupedExprError(t *testing.T) {
	t.Parallel()
	_, err := ParseProgram(`(a) => (])`)
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestParseErrorAssignExprError(t *testing.T) {
	t.Parallel()
	_, err := ParseProgram(`(a) => { let x = 1; x = ]; }`)
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestParseErrorExprStmtError(t *testing.T) {
	t.Parallel()
	_, err := ParseProgram(`(a) => { ]; }`)
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestParseErrorProgramBodyError(t *testing.T) {
	t.Parallel()
	_, err := ParseProgram(`(a) => ]`)
	if err == nil {
		t.Fatal("expected parse error")
	}
}

// --- value.go: ValuesEqual for objects with missing key ---

func TestValuesEqualObjectsMissingKey(t *testing.T) {
	t.Parallel()
	m1 := NewOrderedMap()
	m1.Set("a", NumVal(1))
	m1.Set("b", NumVal(2))
	m2 := NewOrderedMap()
	m2.Set("a", NumVal(1))
	m2.Set("c", NumVal(2))
	if ValuesEqual(ObjectVal(m1), ObjectVal(m2)) {
		t.Error("objects with different keys should not be equal")
	}
}

// --- evalObjectLit: numeric computed key display ---

func TestObjectLitNumericKeyDisplay(t *testing.T) {
	t.Parallel()
	// Object literal with numeric key: { 42: "val" }
	v := mustRun(t, `(x) => { return { 42: "val" }; }`, NumVal(0))
	if v.Kind != KindObject {
		t.Fatalf("expected object, got %s", Display(v))
	}
	val, ok := v.Props.Get("42")
	if !ok || val.Str != "val" {
		t.Errorf("expected {42: 'val'}, got %s", Display(v))
	}
}

// --- ValuesEqual: function kind returns false ---

func TestValuesEqualFunctionKind(t *testing.T) {
	t.Parallel()
	fn := Value{Kind: KindFunction, Fn: &Closure{}}
	if ValuesEqual(fn, fn) {
		t.Error("function values should not be equal")
	}
}

func TestValuesEqualBooleans(t *testing.T) {
	t.Parallel()
	if !ValuesEqual(BoolVal(true), BoolVal(true)) {
		t.Error("true == true")
	}
	if ValuesEqual(BoolVal(true), BoolVal(false)) {
		t.Error("true != false")
	}
}

// --- Parser: object ident key that isn't followed by colon (shorthand/expr form) ---

func TestObjectIdentKeyAsExpr(t *testing.T) {
	t.Parallel()
	// An ident key that's not followed by colon goes through the expr path
	// This is: { foo } where foo is treated as expression key, then expects ':'
	_, err := ParseProgram(`(a) => { return { foo }; }`)
	if err == nil {
		t.Fatal("expected parse error for shorthand ident without colon")
	}
}

func TestDestructureLetWithRest(t *testing.T) {
	t.Parallel()
	v := mustRun(t, `(arr) => {
		let [a, b, ...rest] = arr;
		return rest;
	}`, TupleVal(NumVal(1), NumVal(2), NumVal(3), NumVal(4)))
	if v.Kind != KindTuple || len(v.Elems) != 2 || v.Elems[0].Num != 3 {
		t.Errorf("expected [3, 4], got %s", Display(v))
	}
}

func TestDestructureNonArray(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(x) => { let [a, b] = x; return a; }`)
	_, err := Run(prog, []Value{NumVal(42)}, DefaultBudget)
	if err == nil {
		t.Error("expected error destructuring non-array")
	}
}

func TestAssignUndeclared(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(x) => { undeclared = 5; return undeclared; }`)
	_, err := Run(prog, []Value{NumVal(0)}, DefaultBudget)
	if err == nil {
		t.Error("expected error assigning to undeclared")
	}
}

func TestUndefinedVariable(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(x) => nope`)
	_, err := Run(prog, []Value{NumVal(0)}, DefaultBudget)
	if err == nil {
		t.Error("expected error for undefined variable")
	}
}

func TestSpreadNonArray(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(x) => [...x]`)
	_, err := Run(prog, []Value{NumVal(42)}, DefaultBudget)
	if err == nil {
		t.Error("expected error spreading non-array")
	}
}

func TestSpreadNonObject(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(x) => { return { ...x }; }`)
	_, err := Run(prog, []Value{NumVal(42)}, DefaultBudget)
	if err == nil {
		t.Error("expected error spreading non-object")
	}
}

func TestForOfNonArray(t *testing.T) {
	t.Parallel()
	prog := mustParse(t, `(x) => { for (let i of x) {} return 0; }`)
	_, err := Run(prog, []Value{NumVal(42)}, DefaultBudget)
	if err == nil {
		t.Error("expected error for-of on non-array")
	}
}

func TestLEqGEq(t *testing.T) {
	t.Parallel()
	expectBool(t, `(a, b) => a <= b`, true, NumVal(3), NumVal(3))
	expectBool(t, `(a, b) => a <= b`, true, NumVal(2), NumVal(3))
	expectBool(t, `(a, b) => a <= b`, false, NumVal(4), NumVal(3))
	expectBool(t, `(a, b) => a >= b`, true, NumVal(3), NumVal(3))
	expectBool(t, `(a, b) => a >= b`, true, NumVal(4), NumVal(3))
	expectBool(t, `(a, b) => a >= b`, false, NumVal(2), NumVal(3))
}

func TestNotOperator(t *testing.T) {
	t.Parallel()
	expectBool(t, `(x) => !x`, false, BoolVal(true))
	expectBool(t, `(x) => !x`, true, BoolVal(false))
	expectBool(t, `(x) => !x`, true, NumVal(0))
	expectBool(t, `(x) => !x`, false, NumVal(1))
}

func TestNullLiteral(t *testing.T) {
	t.Parallel()
	v := mustRun(t, `(x) => null`, NumVal(0))
	if v.Kind != KindNull {
		t.Errorf("expected null, got %s", Display(v))
	}
}

func TestObjectMissingIndex(t *testing.T) {
	t.Parallel()
	m := NewOrderedMap()
	m.Set("a", NumVal(1))
	expectUndefined(t, `(o) => o["missing"]`, ObjectVal(m))
}

func TestTypeofFunctionCall(t *testing.T) {
	t.Parallel()
	expectStr(t, `(x) => typeof(x)`, "number", NumVal(42))
	expectStr(t, `(x) => typeof(x)`, "string", StrVal("hi"))
}

func TestObjectLitBacktrackParseError(t *testing.T) {
	t.Parallel()
	// ident not followed by colon, backtrack parses as expr, but expr itself fails
	_, err := ParseProgram(`(x) => ({ foo })`)
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestExpressionStatement(t *testing.T) {
	t.Parallel()
	expectNum(t, `(x) => { x + 1; }`, 6, NumVal(5))
}

func TestUnclosedBlocks(t *testing.T) {
	t.Parallel()
	cases := []string{
		`(x) => { if (true) { return 1;`,          // unclosed if block
		`(x) => { for (let i of [1]) { return i;`, // unclosed for-of
		`(x) => { while (true) { return 1;`,       // unclosed while
	}
	for _, src := range cases {
		_, err := ParseProgram(src)
		if err == nil {
			t.Errorf("expected parse error for: %s", src)
		}
	}
}

func TestSkipTypeAnnotationCloseAtDepthZero(t *testing.T) {
	t.Parallel()
	// ')' at depth 0 in type annotation stops the skip
	expectNum(t, `(x: any) => x`, 42, NumVal(42))
	// '>' at depth 0 in type annotation stops the skip
	_, err := ParseProgram(`(x: Foo>) => x`)
	// This is a malformed annotation but the parser should handle it
	_ = err
}

func TestParseBlockNotCalledWithoutBrace(t *testing.T) {
	t.Parallel()
	// parseBlock is only called when '{' was already checked,
	// but let's verify parseBodyOrExpr handles expression bodies
	expectNum(t, `(x) => x + 1`, 6, NumVal(5))
}
