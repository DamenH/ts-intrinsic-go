package intrinsicdsl

import "testing"

func BenchmarkArithmetic(b *testing.B) {
	prog := mustParseB(b, "(a, b) => a + b")
	args := []Value{NumVal(10), NumVal(20)}
	b.ResetTimer()
	for range b.N {
		Run(prog, args, DefaultBudget)
	}
}

func BenchmarkStringTransform(b *testing.B) {
	prog := mustParseB(b, "(s) => s.toLowerCase().split(' ').join('-')")
	args := []Value{StrVal("Hello World This Is A Title")}
	b.ResetTimer()
	for range b.N {
		Run(prog, args, DefaultBudget)
	}
}

// snake_case to camelCase on a realistic identifier
func BenchmarkCamelCase(b *testing.B) {
	prog := mustParseB(b, `(s) => {
		let parts = s.split('_');
		let result = parts[0];
		let i = 1;
		while (i < parts.length) {
			result = result + parts[i].charAt(0).toUpperCase() + parts[i].slice(1);
			i = i + 1;
		}
		return result;
	}`)
	args := []Value{StrVal("get_user_profile_by_account_id")}
	b.ResetTimer()
	for range b.N {
		Run(prog, args, DefaultBudget)
	}
}

// Full email validation: indexOf, slice, includes on domain
func BenchmarkEmailValidation(b *testing.B) {
	prog := mustParseB(b, `(s) => {
		if (typeof s != 'string') return NeverType;
		let at = s.indexOf('@');
		if (at < 1) return NeverType;
		let domain = s.slice(at + 1);
		if (domain.length < 3 || !domain.includes('.')) return NeverType;
		let dot = domain.indexOf('.');
		if (dot < 1 || dot == domain.length - 1) return NeverType;
		return s;
	}`)
	args := []Value{StrVal("alice.smith@company.example.com")}
	b.ResetTimer()
	for range b.N {
		Run(prog, args, DefaultBudget)
	}
}

// CamelCase all keys of an 8-property object
func BenchmarkObjectKeyTransform(b *testing.B) {
	m := NewOrderedMap()
	m.Set("user_id", NumVal(1))
	m.Set("first_name", StrVal("Alice"))
	m.Set("last_name", StrVal("Smith"))
	m.Set("email_address", StrVal("alice@test.com"))
	m.Set("is_active", BoolVal(true))
	m.Set("created_at", StrVal("2024-01-01"))
	m.Set("updated_at", StrVal("2024-03-15"))
	m.Set("account_type", StrVal("premium"))
	prog := mustParseB(b, `(obj) => {
		let result = {};
		for (let k of Object.keys(obj)) {
			let parts = k.split('_');
			let newKey = parts[0];
			let i = 1;
			while (i < parts.length) {
				newKey = newKey + parts[i].charAt(0).toUpperCase() + parts[i].slice(1);
				i = i + 1;
			}
			result[newKey] = obj[k];
		}
		return result;
	}`)
	args := []Value{ObjectVal(m)}
	b.ResetTimer()
	for range b.N {
		Run(prog, args, DefaultBudget)
	}
}

// Filter + map + reduce on a 50-element array
func BenchmarkArrayPipeline(b *testing.B) {
	elems := make([]Value, 50)
	for i := range elems {
		elems[i] = NumVal(float64(i - 25))
	}
	prog := mustParseB(b, `(arr) => {
		let filtered = arr.filter((x) => x > 0);
		let doubled = filtered.map((x) => x * 2);
		return doubled.reduce((acc, x) => acc + x, 0);
	}`)
	args := []Value{TupleVal(elems...)}
	b.ResetTimer()
	for range b.N {
		Run(prog, args, DefaultBudget)
	}
}

// SQL-like validation: normalize whitespace, check keyword positions
func BenchmarkSqlValidation(b *testing.B) {
	prog := mustParseB(b, `(sql) => {
		if (typeof sql != 'string' || sql.trim().length == 0) return NeverType;
		let norm = '';
		let lastWasSpace = true;
		let ci = 0;
		while (ci < sql.length) {
			let c = sql.slice(ci, ci + 1);
			if (c == ' ' || c == '\n' || c == '\t') {
				if (!lastWasSpace) { norm = norm + ' '; lastWasSpace = true; }
			} else { norm = norm + c; lastWasSpace = false; }
			ci = ci + 1;
		}
		norm = norm.trim();
		let upper = norm.toUpperCase();
		if (!upper.startsWith('SELECT ')) return NeverType;
		let fromIdx = upper.indexOf(' FROM ');
		if (fromIdx == -1) return NeverType;
		let columns = norm.slice(7, fromIdx).trim();
		if (columns.length == 0) return NeverType;
		let afterFrom = norm.slice(fromIdx + 6).trim();
		if (afterFrom.length == 0) return NeverType;
		let result = {};
		result['columns'] = columns.split(',').map((c) => c.trim());
		result['table'] = afterFrom.includes(' ') ? afterFrom.slice(0, afterFrom.indexOf(' ')) : afterFrom;
		return result;
	}`)
	args := []Value{StrVal("SELECT id, name, email, age, status FROM users WHERE active = 1 ORDER BY name ASC LIMIT 50")}
	b.ResetTimer()
	for range b.N {
		Run(prog, args, DefaultBudget)
	}
}

// UUID validation: char-by-char hex check with dash positions
func BenchmarkUuidValidation(b *testing.B) {
	prog := mustParseB(b, `(s) => {
		if (typeof s != 'string' || s.length != 36) return NeverType;
		let hex = 'abcdef0123456789';
		let dashes = [8, 13, 18, 23];
		let i = 0;
		while (i < 36) {
			if (dashes.includes(i)) {
				if (s.slice(i, i + 1) != '-') return NeverType;
			} else {
				if (!hex.includes(s.slice(i, i + 1).toLowerCase())) return NeverType;
			}
			i = i + 1;
		}
		return s;
	}`)
	args := []Value{StrVal("550e8400-e29b-41d4-a716-446655440000")}
	b.ResetTimer()
	for range b.N {
		Run(prog, args, DefaultBudget)
	}
}

// Multi-field object validation: types, ranges, enums
func BenchmarkConfigValidation(b *testing.B) {
	m := NewOrderedMap()
	m.Set("host", StrVal("api.production.example.com"))
	m.Set("port", NumVal(443))
	m.Set("protocol", StrVal("https"))
	m.Set("timeout", NumVal(30000))
	m.Set("retries", NumVal(3))
	m.Set("debug", BoolVal(false))
	prog := mustParseB(b, `(cfg) => {
		if (typeof cfg != 'object') return NeverType;
		if (typeof cfg['host'] != 'string' || cfg['host'].length == 0) return NeverType;
		if (typeof cfg['port'] != 'number' || cfg['port'] < 1 || cfg['port'] > 65535) return NeverType;
		if (cfg['protocol'] != 'http' && cfg['protocol'] != 'https') return NeverType;
		if (typeof cfg['timeout'] != 'number' || cfg['timeout'] < 0) return NeverType;
		if (typeof cfg['retries'] != 'number' || cfg['retries'] < 0 || cfg['retries'] > 10) return NeverType;
		if (typeof cfg['debug'] != 'boolean') return NeverType;
		return cfg;
	}`)
	args := []Value{ObjectVal(m)}
	b.ResetTimer()
	for range b.N {
		Run(prog, args, DefaultBudget)
	}
}

// Semver parsing: split, loop, digit-by-digit parseInt
func BenchmarkSemverParsing(b *testing.B) {
	prog := mustParseB(b, `(s) => {
		if (typeof s != 'string') return NeverType;
		let parts = s.split('.');
		if (parts.length != 3) return NeverType;
		let digits = '0123456789';
		let nums = [];
		for (let part of parts) {
			if (part.length == 0) return NeverType;
			let n = 0;
			let pi = 0;
			while (pi < part.length) {
				let d = digits.indexOf(part.slice(pi, pi + 1));
				if (d == -1) return NeverType;
				n = n * 10 + d;
				pi = pi + 1;
			}
			nums = [...nums, n];
		}
		let result = {};
		result['major'] = nums[0];
		result['minor'] = nums[1];
		result['patch'] = nums[2];
		return result;
	}`)
	args := []Value{StrVal("16.4.2")}
	b.ResetTimer()
	for range b.N {
		Run(prog, args, DefaultBudget)
	}
}

// Connection string parsing: protocol, host, port, database extraction
func BenchmarkConnectionStringParsing(b *testing.B) {
	prog := mustParseB(b, `(s) => {
		if (typeof s != 'string') return NeverType;
		let protocolEnd = s.indexOf('://');
		if (protocolEnd == -1) return NeverType;
		let protocol = s.slice(0, protocolEnd);
		let rest = s.slice(protocolEnd + 3);
		let pathStart = rest.indexOf('/');
		let hostPort = pathStart == -1 ? rest : rest.slice(0, pathStart);
		let database = pathStart == -1 ? '' : rest.slice(pathStart + 1);
		let colonIdx = hostPort.indexOf(':');
		let host = colonIdx == -1 ? hostPort : hostPort.slice(0, colonIdx);
		let result = {};
		result['protocol'] = protocol;
		result['host'] = host;
		if (database.length > 0) result['database'] = database;
		return result;
	}`)
	args := []Value{StrVal("postgres://db.production.internal:5432/myapp_production")}
	b.ResetTimer()
	for range b.N {
		Run(prog, args, DefaultBudget)
	}
}

// Parser throughput: measures parsing a complex function body
func BenchmarkParseComplex(b *testing.B) {
	src := `(cfg) => {
		if (typeof cfg != 'object') return NeverType;
		if (typeof cfg['host'] != 'string' || cfg['host'].length == 0) return NeverType;
		if (typeof cfg['port'] != 'number' || cfg['port'] < 1 || cfg['port'] > 65535) return NeverType;
		if (cfg['protocol'] != 'http' && cfg['protocol'] != 'https') return NeverType;
		let result = {};
		result['url'] = cfg['protocol'] + '://' + cfg['host'] + ':' + cfg['port'];
		return result;
	}`
	b.ResetTimer()
	for range b.N {
		ParseProgram(src)
	}
}

func mustParseB(b *testing.B, src string) *Node {
	b.Helper()
	node, err := ParseProgram(src)
	if err != nil {
		b.Fatalf("parse error: %v", err)
	}
	return node
}
