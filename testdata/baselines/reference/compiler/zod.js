//// [tests/cases/compiler/intrinsic/zod.ts] ////

//// [zod.ts]
// Zod-like validators as plain functions.
// Each works at both compile time (via Intrinsic<typeof fn, [T]>) and runtime.

const capitalize = (s: string) => s.charAt(0).toUpperCase() + s.slice(1);
function range(startOrEnd: number, end?: number): number[] {
    const s = end === undefined ? 0 : startOrEnd;
    const e = end === undefined ? startOrEnd : end;
    return Array.from({ length: Math.max(e - s, 0) }, (_, i) => s + i);
}

// ---- Primitives ----

const _string = (x: any) => {
    if (typeof x != 'string') return void { error: "Expected string, got " + typeof x };
    return x;
};
const _number = (x: any) => {
    if (typeof x != 'number') return void { error: "Expected number, got " + typeof x };
    return x;
};
const _boolean = (x: any) => {
    if (typeof x != 'boolean') return void { error: "Expected boolean, got " + typeof x };
    return x;
};

// ---- String validators ----

const _email = (s: any) => {
    if (typeof s != 'string') return void { error: "Expected string for email, got " + typeof s };
    let at: number = s.indexOf('@');
    if (at < 1) return void { error: "Invalid email '" + s + "': missing @ sign" };
    let domain: string = s.slice(at + 1);
    if (domain.length < 3 || !domain.includes('.')) return void { error: "Invalid email '" + s + "': invalid domain" };
    let dot: number = domain.indexOf('.');
    if (dot < 1 || dot == domain.length - 1) return void { error: "Invalid email '" + s + "': invalid domain" };
    return s;
};

const _url = (s: any) => {
    if (typeof s != 'string') return void { error: "Expected string for URL, got " + typeof s };
    if (!s.startsWith('http://') && !s.startsWith('https://')) return void { error: "URL must start with http:// or https://, got '" + s + "'" };
    if (s.length < 10) return void { error: "URL too short: '" + s + "'" };
    return s;
};

const _uuid = (s: any) => {
    if (typeof s != 'string') return void { error: "Expected string for UUID, got " + typeof s };
    if (s.length != 36) return void { error: "UUID must be 36 characters, got " + s.length };
    let hex: string = 'abcdef0123456789';
    let dashes = [8, 13, 18, 23];
    let i = 0;
    while (i < 36) {
        if (dashes.includes(i)) {
            if (s.slice(i, i + 1) != '-') return void { error: "UUID has invalid format: '" + s + "'" };
        } else {
            if (!hex.includes(s.slice(i, i + 1).toLowerCase())) return void { error: "UUID has invalid format: '" + s + "'" };
        }
        i = i + 1;
    }
    return s;
};

const _nonEmpty = (s: any) => {
    if (typeof s != 'string') return void { error: "Expected string, got " + typeof s };
    if (s.length == 0) return void { error: "String must not be empty" };
    return s;
};

const _trimmed = (s: any) => {
    if (typeof s != 'string') return void "never";
    return s.trim();
};

// ---- Number validators ----

const _int = (n: any) => {
    if (typeof n != 'number') return void { error: "Expected number, got " + typeof n };
    if (Math.floor(n) != n) return void { error: n + " is not an integer" };
    return n;
};

const _positive = (n: any) => {
    if (typeof n != 'number') return void { error: "Expected number, got " + typeof n };
    if (n <= 0) return void { error: "Expected positive number, got " + n };
    return n;
};

const _nonnegative = (n: any) => {
    if (typeof n != 'number') return void { error: "Expected number, got " + typeof n };
    if (n < 0) return void { error: "Expected non-negative number, got " + n };
    return n;
};

const _port = (n: any) => {
    if (typeof n != 'number') return void { error: "Port must be a number, got " + typeof n };
    if (n < 1 || n > 65535) return void { error: "Port " + n + " out of range (1-65535)" };
    if (Math.floor(n) != n) return void { error: "Port must be an integer, got " + n };
    return n;
};

// ---- Coercions ----

const _coerceNumber = (s: any) => {
    if (typeof s == 'number') return s;
    if (typeof s != 'string' || s.length == 0) return void "never";
    let digits: string = '0123456789';
    let start: number = 0;
    let neg: boolean = false;
    if (s.slice(0, 1) == '-') { neg = true; start = 1; }
    if (start == s.length) return void "never";
    let result: number = 0;
    let i = start;
    while (i < s.length) {
        let d: number = digits.indexOf(s.slice(i, i + 1));
        if (d == -1) return void "never";
        result = result * 10 + d;
        i = i + 1;
    }
    return neg ? -result : result;
};

const _coerceBoolean = (x: any) => {
    if (typeof x == 'boolean') return x;
    if (x == 'true' || x == '1') return true;
    if (x == 'false' || x == '0') return false;
    return void "never";
};

// ---- Transforms ----

const _toLowerCase = (s: any) => {
    if (typeof s != 'string') return void "never";
    return s.toLowerCase();
};

const _toUpperCase = (s: any) => {
    if (typeof s != 'string') return void "never";
    return s.toUpperCase();
};

const _toSlug = (s: string) => {
    return s.toLowerCase().split(' ').join('-');
};

const _toCamelCase = (s: string) => {
    let [first, ...rest] = s.split('_');
    return first + rest.map((p: string) => capitalize(p)).join('');
};

// ---- Object validators ----

const _serverConfig = (cfg: any) => {
    if (typeof cfg != 'object') return void { error: "ServerConfig: expected an object" };
    if (typeof cfg['host'] != 'string' || cfg['host'].length == 0) return void { error: "ServerConfig.host: must be a non-empty string" };
    if (typeof cfg['port'] != 'number') return void { error: "ServerConfig.port: must be a number" };
    if (cfg['port'] < 1 || cfg['port'] > 65535) return void { error: "ServerConfig.port: " + cfg['port'] + " out of range (1-65535)" };
    if (cfg['protocol'] != 'http' && cfg['protocol'] != 'https') return void { error: "ServerConfig.protocol: must be 'http' or 'https', got '" + cfg['protocol'] + "'" };
    return cfg;
};

const _user = (u: any) => {
    if (typeof u != 'object') return void { error: "User: expected an object" };
    if (typeof u['name'] != 'string' || u['name'].length == 0) return void { error: "User.name: must be a non-empty string" };
    if (typeof u['email'] != 'string') return void { error: "User.email: must be a string" };
    let at: number = u['email'].indexOf('@');
    if (at < 1 || !u['email'].slice(at + 1).includes('.')) return void { error: "User.email: '" + u['email'] + "' is not a valid email" };
    if (typeof u['age'] != 'number') return void { error: "User.age: must be a number" };
    if (u['age'] < 0 || u['age'] > 150) return void { error: "User.age: " + u['age'] + " out of range (0-150)" };
    if (Math.floor(u['age']) != u['age']) return void { error: "User.age: must be an integer, got " + u['age'] };
    return u;
};

const _dbConfig = (cfg: any) => {
    if (typeof cfg != 'object') return void { error: "DbConfig: expected an object" };
    let driver = cfg['driver'];
    if (driver != 'postgres' && driver != 'mysql' && driver != 'sqlite') return void { error: "DbConfig.driver: must be 'postgres', 'mysql', or 'sqlite', got '" + driver + "'" };
    if (typeof cfg['host'] != 'string' || cfg['host'].length == 0) return void { error: "DbConfig.host: must be a non-empty string" };
    if (typeof cfg['port'] != 'number') return void { error: "DbConfig.port: must be a number" };
    if (cfg['port'] < 1 || cfg['port'] > 65535) return void { error: "DbConfig.port: " + cfg['port'] + " out of range (1-65535)" };
    if (typeof cfg['database'] != 'string' || cfg['database'].length == 0) return void { error: "DbConfig.database: must be a non-empty string" };
    return cfg;
};

const _envConfig = (env: any) => {
    if (typeof env != 'object') return void { error: "EnvConfig: expected an object" };
    let nodeEnv = env['NODE_ENV'];
    if (nodeEnv != 'development' && nodeEnv != 'production' && nodeEnv != 'test') return void { error: "EnvConfig.NODE_ENV: must be 'development', 'production', or 'test', got '" + nodeEnv + "'" };
    if (typeof env['PORT'] != 'number') return void { error: "EnvConfig.PORT: must be a number" };
    if (typeof env['DATABASE_URL'] != 'string' || env['DATABASE_URL'].length == 0) return void { error: "EnvConfig.DATABASE_URL: must be a non-empty string" };
    if (typeof env['DEBUG'] != 'boolean') return void { error: "EnvConfig.DEBUG: must be a boolean" };
    return env;
};

// ---- Object transforms ----

const _camelCaseKeys = (obj: any) => {
    let result: Record<string, any> = {};
    for (let k of Object.keys(obj)) {
        let [first, ...rest] = k.split('_');
        let newKey: string = first + rest.map((p: string) => capitalize(p)).join('');
        result[newKey] = obj[k];
    }
    return result;
};

const _buildUrl = (cfg: any) => {
    if (typeof cfg != 'object') return void "never";
    let url: string = cfg['protocol'] + '://' + cfg['host'];
    if (typeof cfg['port'] == 'number') url = url + ':' + cfg['port'];
    if (typeof cfg['path'] == 'string') url = url + cfg['path'];
    return url;
};

const _parseDate = (s: any) => {
    if (typeof s != 'string' || s.length != 10) return void "never";
    if (s.slice(4, 5) != '-' || s.slice(7, 8) != '-') return void "never";
    let digits: string = '0123456789';
    for (let p of [0, 1, 2, 3, 5, 6, 8, 9]) {
        if (!digits.includes(s.slice(p, p + 1))) return void "never";
    }
    let year: number = digits.indexOf(s.slice(0, 1)) * 1000 + digits.indexOf(s.slice(1, 2)) * 100 + digits.indexOf(s.slice(2, 3)) * 10 + digits.indexOf(s.slice(3, 4));
    let month: number = digits.indexOf(s.slice(5, 6)) * 10 + digits.indexOf(s.slice(6, 7));
    let day: number = digits.indexOf(s.slice(8, 9)) * 10 + digits.indexOf(s.slice(9, 10));
    if (month < 1 || month > 12 || day < 1 || day > 31) return void "never";
    let result: Record<string, any> = {};
    result['year'] = year;
    result['month'] = month;
    result['day'] = day;
    return result;
};

// ---- Parsers ----

const _parseRoute = (path: any) => {
    if (typeof path != 'string' || !path.startsWith('/api/')) return void "never";
    let parts: string[] = path.slice(5).split('/');
    if (parts.length < 2) return void "never";
    let result: Record<string, any> = {};
    result['version'] = parts[0];
    result['resource'] = parts[1];
    if (parts.length > 2) result['id'] = parts[2];
    return result;
};

const _hexToRgb = (s: any) => {
    if (typeof s != 'string' || s.length != 7 || s.slice(0, 1) != '#') return void "never";
    let hex: string = '0123456789abcdef';
    let result: Record<string, any> = {};
    let r1: number = hex.indexOf(s.slice(1, 2).toLowerCase());
    let r2: number = hex.indexOf(s.slice(2, 3).toLowerCase());
    let g1: number = hex.indexOf(s.slice(3, 4).toLowerCase());
    let g2: number = hex.indexOf(s.slice(4, 5).toLowerCase());
    let b1: number = hex.indexOf(s.slice(5, 6).toLowerCase());
    let b2: number = hex.indexOf(s.slice(6, 7).toLowerCase());
    if (r1 == -1 || r2 == -1 || g1 == -1 || g2 == -1 || b1 == -1 || b2 == -1) return void "never";
    result['r'] = r1 * 16 + r2;
    result['g'] = g1 * 16 + g2;
    result['b'] = b1 * 16 + b2;
    return result;
};

const _parseSemver = (s: any) => {
    if (typeof s != 'string') return void "never";
    let parts: string[] = s.split('.');
    if (parts.length != 3) return void "never";
    let digits: string = '0123456789';
    let nums: number[] = [];
    for (let part of parts) {
        if (part.length == 0) return void "never";
        let n: number = 0;
        let i = 0;
        while (i < part.length) {
            let d: number = digits.indexOf(part.slice(i, i + 1));
            if (d == -1) return void "never";
            n = n * 10 + d;
            i = i + 1;
        }
        nums = [...nums, n];
    }
    let result: Record<string, any> = {};
    result['major'] = nums[0];
    result['minor'] = nums[1];
    result['patch'] = nums[2];
    return result;
};

const _parseConnectionString = (s: any) => {
    if (typeof s != 'string') return void "never";
    let protocolEnd: number = s.indexOf('://');
    if (protocolEnd == -1) return void "never";
    let protocol: string = s.slice(0, protocolEnd);
    let rest: string = s.slice(protocolEnd + 3);
    let pathStart: number = rest.indexOf('/');
    let hostPort: string = pathStart == -1 ? rest : rest.slice(0, pathStart);
    let database: string = pathStart == -1 ? '' : rest.slice(pathStart + 1);
    let colonIdx: number = hostPort.indexOf(':');
    let host: string = colonIdx == -1 ? hostPort : hostPort.slice(0, colonIdx);
    let result: Record<string, any> = {};
    result['protocol'] = protocol;
    result['host'] = host;
    if (database.length > 0) result['database'] = database;
    return result;
};

// ---- Pick / Omit ----

const _pick = (obj: any, fields: string[]) => {
    if (typeof obj != 'object') return void "never";
    let result: Record<string, any> = {};
    for (let f of fields) {
        if (obj[f] != undefined) result[f] = obj[f];
    }
    return result;
};

const _omit = (obj: any, fields: string[]) => {
    if (typeof obj != 'object') return void "never";
    let result: Record<string, any> = {};
    for (let k of Object.keys(obj)) {
        if (!fields.includes(k)) result[k] = obj[k];
    }
    return result;
};

const _enrichUser = (u: any) => {
    if (typeof u != 'object') return void "never";
    if (typeof u['firstName'] != 'string' || typeof u['lastName'] != 'string') return void "never";
    let result: Record<string, any> = {};
    result['firstName'] = u['firstName'];
    result['lastName'] = u['lastName'];
    result['email'] = u['email'];
    result['fullName'] = u['firstName'] + ' ' + u['lastName'];
    result['handle'] = u['firstName'].toLowerCase() + u['lastName'].toLowerCase();
    return result;
};

// ---- Validators with custom error messages ----
// Use void "error:message" to produce compile-time diagnostics.

const _strictEmail = (s: any) => {
    if (typeof s != 'string') return void { error: "Expected a string, got " + typeof s };
    let at: number = s.indexOf('@');
    if (at < 1) return void { error: "Invalid email '" + s + "': missing @ sign" };
    let domain: string = s.slice(at + 1);
    if (!domain.includes('.')) return void { error: "Invalid email '" + s + "': domain has no dot" };
    return s;
};

const _strictPort = (n: any) => {
    if (typeof n != 'number') return void { error: "Port must be a number, got " + typeof n };
    if (n < 1 || n > 65535) return void { error: "Port " + n + " is out of range (1-65535)" };
    if (Math.floor(n) != n) return void { error: "Port must be an integer, got " + n };
    return n;
};

const _strictPositive = (n: any) => {
    if (typeof n != 'number') return void { error: "Expected a number, got " + typeof n };
    if (n <= 0) return void { error: "Expected a positive number, got " + n };
    return n;
};

// ---- Exported type aliases (the public API) ----

export type ZString<T> = Intrinsic<typeof _string, [T]>;
export type ZNumber<T> = Intrinsic<typeof _number, [T]>;
export type ZBoolean<T> = Intrinsic<typeof _boolean, [T]>;
export type ZEmail<T> = Intrinsic<typeof _email, [T]>;
export type ZUrl<T> = Intrinsic<typeof _url, [T]>;
export type ZUuid<T> = Intrinsic<typeof _uuid, [T]>;
export type ZNonEmpty<T> = Intrinsic<typeof _nonEmpty, [T]>;
export type ZTrimmed<T> = Intrinsic<typeof _trimmed, [T]>;
export type ZInt<T> = Intrinsic<typeof _int, [T]>;
export type ZPositive<T> = Intrinsic<typeof _positive, [T]>;
export type ZNonnegative<T> = Intrinsic<typeof _nonnegative, [T]>;
export type ZPort<T> = Intrinsic<typeof _port, [T]>;
export type ZCoerceNumber<T> = Intrinsic<typeof _coerceNumber, [T]>;
export type ZCoerceBoolean<T> = Intrinsic<typeof _coerceBoolean, [T]>;
export type ZToLowerCase<T> = Intrinsic<typeof _toLowerCase, [T]>;
export type ZToUpperCase<T> = Intrinsic<typeof _toUpperCase, [T]>;
export type ZToSlug<T> = Intrinsic<typeof _toSlug, [T]>;
export type ZToCamelCase<T> = Intrinsic<typeof _toCamelCase, [T]>;
export type ZServerConfig<T> = Intrinsic<typeof _serverConfig, [T]>;
export type ZUser<T> = Intrinsic<typeof _user, [T]>;
export type ZDbConfig<T> = Intrinsic<typeof _dbConfig, [T]>;
export type ZEnvConfig<T> = Intrinsic<typeof _envConfig, [T]>;
export type ZCamelCaseKeys<T> = Intrinsic<typeof _camelCaseKeys, [T]>;
export type ZBuildUrl<T> = Intrinsic<typeof _buildUrl, [T]>;
export type ZParseDate<T> = Intrinsic<typeof _parseDate, [T]>;
export type ZParseRoute<T> = Intrinsic<typeof _parseRoute, [T]>;
export type ZHexToRgb<T> = Intrinsic<typeof _hexToRgb, [T]>;
export type ZParseSemver<T> = Intrinsic<typeof _parseSemver, [T]>;
export type ZParseConnectionString<T> = Intrinsic<typeof _parseConnectionString, [T]>;
export type Pick_<T, F extends string[]> = Intrinsic<typeof _pick, [T, F]>;
export type Omit_<T, F extends string[]> = Intrinsic<typeof _omit, [T, F]>;
export type ZEnrichUser<T> = Intrinsic<typeof _enrichUser, [T]>;
export type ZStrictEmail<T> = Intrinsic<typeof _strictEmail, [T]>;
export type ZStrictPort<T> = Intrinsic<typeof _strictPort, [T]>;
export type ZStrictPositive<T> = Intrinsic<typeof _strictPositive, [T]>;


//// [zod_test.ts]
// Consumer file - imports type aliases and uses them.

import type {
    ZString, ZNumber, ZBoolean,
    ZEmail, ZUrl, ZUuid, ZNonEmpty, ZTrimmed,
    ZInt, ZPositive, ZNonnegative, ZPort,
    ZCoerceNumber, ZCoerceBoolean,
    ZToLowerCase, ZToUpperCase, ZToSlug, ZToCamelCase,
    ZServerConfig, ZUser, ZDbConfig, ZEnvConfig,
    ZCamelCaseKeys, ZBuildUrl, ZParseDate,
    ZParseRoute, ZHexToRgb, ZParseSemver, ZParseConnectionString,
    Pick_, Omit_, ZEnrichUser,
    ZStrictEmail, ZStrictPort, ZStrictPositive,
} from './zod';

// ==== Primitive validation ====

type T01 = ZString<"hello">;       // "hello"
type T02 = ZString<42>;            // never
type T03 = ZNumber<42>;            // 42
type T04 = ZNumber<"nope">;        // never
type T05 = ZBoolean<true>;         // true
type T06 = ZBoolean<"yes">;        // never

// ==== String format validation ====

type T10 = ZEmail<"user@example.com">;     // "user@example.com"
type T11 = ZEmail<"bad">;                   // never
type T12 = ZUrl<"https://example.com">;     // "https://example.com"
type T13 = ZUrl<"not-a-url">;               // never
type T14 = ZUuid<"550e8400-e29b-41d4-a716-446655440000">;  // the uuid
type T15 = ZUuid<"not-a-uuid">;             // never
type T16 = ZNonEmpty<"hi">;                 // "hi"
type T17 = ZNonEmpty<"">;                   // never

// ==== Number validation ====

type T20 = ZInt<42>;           // 42
type T21 = ZInt<3.14>;         // never
type T22 = ZPositive<5>;       // 5
type T23 = ZPositive<0>;       // never
type T24 = ZPositive<-1>;      // never
type T25 = ZNonnegative<0>;    // 0
type T26 = ZNonnegative<-1>;   // never
type T27 = ZPort<443>;         // 443
type T28 = ZPort<0>;           // never
type T29 = ZPort<70000>;       // never

// ==== Coercions ====

type T30 = ZCoerceNumber<"42">;      // 42
type T31 = ZCoerceNumber<"-7">;      // -7
type T32 = ZCoerceNumber<"abc">;     // never
type T33 = ZCoerceNumber<100>;       // 100 (passthrough)
type T34 = ZCoerceBoolean<"true">;   // true
type T35 = ZCoerceBoolean<"false">;  // false
type T36 = ZCoerceBoolean<true>;     // true (passthrough)

// ==== Transforms ====

type T40 = ZToLowerCase<"HELLO">;                // "hello"
type T41 = ZToUpperCase<"hello">;                // "HELLO"
type T42 = ZToSlug<"Hello World">;               // "hello-world"
type T43 = ZToCamelCase<"user_first_name">;      // "userFirstName"
type T44 = ZTrimmed<"  hello  ">;                // "hello"

// ==== Object validation ====

type T50 = ZServerConfig<{ host: "localhost", port: 3000, protocol: "http" }>;
type T51 = ZServerConfig<{ host: "api.example.com", port: 443, protocol: "https" }>;
type T52 = ZServerConfig<{ host: "", port: 3000, protocol: "http" }>;          // never
type T53 = ZServerConfig<{ host: "localhost", port: 0, protocol: "http" }>;     // never

type T60 = ZUser<{ name: "Alice", email: "alice@example.com", age: 30 }>;
type T61 = ZUser<{ name: "", email: "alice@example.com", age: 30 }>;           // never
type T62 = ZUser<{ name: "Bob", email: "invalid", age: 25 }>;                  // never
type T63 = ZUser<{ name: "Eve", email: "eve@test.com", age: -1 }>;             // never

type T70 = ZDbConfig<{ driver: "postgres", host: "localhost", port: 5432, database: "myapp" }>;
type T71 = ZDbConfig<{ driver: "mysql", host: "db.internal", port: 3306, database: "users" }>;
type T72 = ZDbConfig<{ driver: "redis", host: "localhost", port: 6379, database: "cache" }>;  // never

type T80 = ZEnvConfig<{ NODE_ENV: "production", PORT: 3000, DATABASE_URL: "postgres://localhost/db", DEBUG: false }>;
type T81 = ZEnvConfig<{ NODE_ENV: "invalid", PORT: 3000, DATABASE_URL: "postgres://localhost/db", DEBUG: false }>;  // never

// ==== Object transforms ====

type T90 = ZCamelCaseKeys<{ user_id: 1, first_name: "Alice", is_active: true }>;
type T91 = ZCamelCaseKeys<{ created_at: "2024-01-01", updated_at: "2024-03-15" }>;

type T92 = ZBuildUrl<{ protocol: "https", host: "api.example.com", port: 443, path: "/v1/users" }>;
type T93 = ZBuildUrl<{ protocol: "http", host: "localhost", port: 8080, path: "/health" }>;

type T94 = ZParseDate<"2024-03-15">;
type T95 = ZParseDate<"1999-12-31">;
type T96 = ZParseDate<"2024-13-01">;   // never
type T97 = ZParseDate<"not-a-date">;   // never

// ==== Route / color / semver parsing ====

type P1 = ZParseRoute<"/api/v1/users">;        // { version: "v1", resource: "users" }
type P2 = ZParseRoute<"/api/v2/posts/42">;     // { version: "v2", resource: "posts", id: "42" }
type P3 = ZParseRoute<"/not/api">;              // never

type C1 = ZHexToRgb<"#ff0000">;   // { r: 255, g: 0, b: 0 }
type C2 = ZHexToRgb<"#00ff00">;   // { r: 0, g: 255, b: 0 }
type C3 = ZHexToRgb<"#1a2b3c">;   // { r: 26, g: 43, b: 60 }
type C4 = ZHexToRgb<"red">;        // never

type V1 = ZParseSemver<"1.0.0">;    // { major: 1, minor: 0, patch: 0 }
type V2 = ZParseSemver<"16.4.2">;   // { major: 16, minor: 4, patch: 2 }
type V3 = ZParseSemver<"1.0">;       // never
type V4 = ZParseSemver<"a.b.c">;     // never

// ==== Connection string ====

type CS1 = ZParseConnectionString<"postgres://localhost/mydb">;
type CS2 = ZParseConnectionString<"redis://cache.internal">;
type CS3 = ZParseConnectionString<"mysql://db.prod:3306/users">;
type CS4 = ZParseConnectionString<"not-a-connection-string">;  // never

// ==== Pick / Omit / Enrich ====

type FullUser = { id: 1, name: "Alice", email: "alice@test.com", password: "secret" };
type PublicUser = Omit_<FullUser, ["password"]>;   // { id: 1, name: "Alice", email: "alice@test.com" }
type UserRef = Pick_<FullUser, ["id", "name"]>;    // { id: 1, name: "Alice" }

type D1 = ZEnrichUser<{ firstName: "Alice", lastName: "Smith", email: "a@b.c" }>;
// { firstName: "Alice", lastName: "Smith", email: "a@b.c", fullName: "Alice Smith", handle: "alicesmith" }

// ==== Custom error messages (void "error:...") ====

// Valid inputs - no errors, types resolve normally
type SE1 = ZStrictEmail<"user@example.com">;       // "user@example.com"
type SP1 = ZStrictPort<443>;                        // 443
type SN1 = ZStrictPositive<42>;                     // 42

// Invalid inputs - should produce compile errors with custom messages
type SE2 = ZStrictEmail<"no-at-sign">;              // error: Invalid email 'no-at-sign': missing @ sign
type SE3 = ZStrictEmail<"user@nodot">;              // error: Invalid email 'user@nodot': domain has no dot
type SE4 = ZStrictEmail<42>;                        // error: Expected a string, got number
type SP2 = ZStrictPort<0>;                          // error: Port 0 is out of range (1-65535)
type SP3 = ZStrictPort<99999>;                      // error: Port 99999 is out of range (1-65535)
type SN2 = ZStrictPositive<-5>;                     // error: Expected a positive number, got -5
type SN3 = ZStrictPositive<0>;                      // error: Expected a positive number, got 0


//// [zod.js]
// Zod-like validators as plain functions.
// Each works at both compile time (via Intrinsic<typeof fn, [T]>) and runtime.
const capitalize = (s) => s.charAt(0).toUpperCase() + s.slice(1);
function range(startOrEnd, end) {
    const s = end === undefined ? 0 : startOrEnd;
    const e = end === undefined ? startOrEnd : end;
    return Array.from({ length: Math.max(e - s, 0) }, (_, i) => s + i);
}
// ---- Primitives ----
const _string = (x) => {
    if (typeof x != 'string')
        return void { error: "Expected string, got " + typeof x };
    return x;
};
const _number = (x) => {
    if (typeof x != 'number')
        return void { error: "Expected number, got " + typeof x };
    return x;
};
const _boolean = (x) => {
    if (typeof x != 'boolean')
        return void { error: "Expected boolean, got " + typeof x };
    return x;
};
// ---- String validators ----
const _email = (s) => {
    if (typeof s != 'string')
        return void { error: "Expected string for email, got " + typeof s };
    let at = s.indexOf('@');
    if (at < 1)
        return void { error: "Invalid email '" + s + "': missing @ sign" };
    let domain = s.slice(at + 1);
    if (domain.length < 3 || !domain.includes('.'))
        return void { error: "Invalid email '" + s + "': invalid domain" };
    let dot = domain.indexOf('.');
    if (dot < 1 || dot == domain.length - 1)
        return void { error: "Invalid email '" + s + "': invalid domain" };
    return s;
};
const _url = (s) => {
    if (typeof s != 'string')
        return void { error: "Expected string for URL, got " + typeof s };
    if (!s.startsWith('http://') && !s.startsWith('https://'))
        return void { error: "URL must start with http:// or https://, got '" + s + "'" };
    if (s.length < 10)
        return void { error: "URL too short: '" + s + "'" };
    return s;
};
const _uuid = (s) => {
    if (typeof s != 'string')
        return void { error: "Expected string for UUID, got " + typeof s };
    if (s.length != 36)
        return void { error: "UUID must be 36 characters, got " + s.length };
    let hex = 'abcdef0123456789';
    let dashes = [8, 13, 18, 23];
    let i = 0;
    while (i < 36) {
        if (dashes.includes(i)) {
            if (s.slice(i, i + 1) != '-')
                return void { error: "UUID has invalid format: '" + s + "'" };
        }
        else {
            if (!hex.includes(s.slice(i, i + 1).toLowerCase()))
                return void { error: "UUID has invalid format: '" + s + "'" };
        }
        i = i + 1;
    }
    return s;
};
const _nonEmpty = (s) => {
    if (typeof s != 'string')
        return void { error: "Expected string, got " + typeof s };
    if (s.length == 0)
        return void { error: "String must not be empty" };
    return s;
};
const _trimmed = (s) => {
    if (typeof s != 'string')
        return void "never";
    return s.trim();
};
// ---- Number validators ----
const _int = (n) => {
    if (typeof n != 'number')
        return void { error: "Expected number, got " + typeof n };
    if (Math.floor(n) != n)
        return void { error: n + " is not an integer" };
    return n;
};
const _positive = (n) => {
    if (typeof n != 'number')
        return void { error: "Expected number, got " + typeof n };
    if (n <= 0)
        return void { error: "Expected positive number, got " + n };
    return n;
};
const _nonnegative = (n) => {
    if (typeof n != 'number')
        return void { error: "Expected number, got " + typeof n };
    if (n < 0)
        return void { error: "Expected non-negative number, got " + n };
    return n;
};
const _port = (n) => {
    if (typeof n != 'number')
        return void { error: "Port must be a number, got " + typeof n };
    if (n < 1 || n > 65535)
        return void { error: "Port " + n + " out of range (1-65535)" };
    if (Math.floor(n) != n)
        return void { error: "Port must be an integer, got " + n };
    return n;
};
// ---- Coercions ----
const _coerceNumber = (s) => {
    if (typeof s == 'number')
        return s;
    if (typeof s != 'string' || s.length == 0)
        return void "never";
    let digits = '0123456789';
    let start = 0;
    let neg = false;
    if (s.slice(0, 1) == '-') {
        neg = true;
        start = 1;
    }
    if (start == s.length)
        return void "never";
    let result = 0;
    let i = start;
    while (i < s.length) {
        let d = digits.indexOf(s.slice(i, i + 1));
        if (d == -1)
            return void "never";
        result = result * 10 + d;
        i = i + 1;
    }
    return neg ? -result : result;
};
const _coerceBoolean = (x) => {
    if (typeof x == 'boolean')
        return x;
    if (x == 'true' || x == '1')
        return true;
    if (x == 'false' || x == '0')
        return false;
    return void "never";
};
// ---- Transforms ----
const _toLowerCase = (s) => {
    if (typeof s != 'string')
        return void "never";
    return s.toLowerCase();
};
const _toUpperCase = (s) => {
    if (typeof s != 'string')
        return void "never";
    return s.toUpperCase();
};
const _toSlug = (s) => {
    return s.toLowerCase().split(' ').join('-');
};
const _toCamelCase = (s) => {
    let [first, ...rest] = s.split('_');
    return first + rest.map((p) => capitalize(p)).join('');
};
// ---- Object validators ----
const _serverConfig = (cfg) => {
    if (typeof cfg != 'object')
        return void { error: "ServerConfig: expected an object" };
    if (typeof cfg['host'] != 'string' || cfg['host'].length == 0)
        return void { error: "ServerConfig.host: must be a non-empty string" };
    if (typeof cfg['port'] != 'number')
        return void { error: "ServerConfig.port: must be a number" };
    if (cfg['port'] < 1 || cfg['port'] > 65535)
        return void { error: "ServerConfig.port: " + cfg['port'] + " out of range (1-65535)" };
    if (cfg['protocol'] != 'http' && cfg['protocol'] != 'https')
        return void { error: "ServerConfig.protocol: must be 'http' or 'https', got '" + cfg['protocol'] + "'" };
    return cfg;
};
const _user = (u) => {
    if (typeof u != 'object')
        return void { error: "User: expected an object" };
    if (typeof u['name'] != 'string' || u['name'].length == 0)
        return void { error: "User.name: must be a non-empty string" };
    if (typeof u['email'] != 'string')
        return void { error: "User.email: must be a string" };
    let at = u['email'].indexOf('@');
    if (at < 1 || !u['email'].slice(at + 1).includes('.'))
        return void { error: "User.email: '" + u['email'] + "' is not a valid email" };
    if (typeof u['age'] != 'number')
        return void { error: "User.age: must be a number" };
    if (u['age'] < 0 || u['age'] > 150)
        return void { error: "User.age: " + u['age'] + " out of range (0-150)" };
    if (Math.floor(u['age']) != u['age'])
        return void { error: "User.age: must be an integer, got " + u['age'] };
    return u;
};
const _dbConfig = (cfg) => {
    if (typeof cfg != 'object')
        return void { error: "DbConfig: expected an object" };
    let driver = cfg['driver'];
    if (driver != 'postgres' && driver != 'mysql' && driver != 'sqlite')
        return void { error: "DbConfig.driver: must be 'postgres', 'mysql', or 'sqlite', got '" + driver + "'" };
    if (typeof cfg['host'] != 'string' || cfg['host'].length == 0)
        return void { error: "DbConfig.host: must be a non-empty string" };
    if (typeof cfg['port'] != 'number')
        return void { error: "DbConfig.port: must be a number" };
    if (cfg['port'] < 1 || cfg['port'] > 65535)
        return void { error: "DbConfig.port: " + cfg['port'] + " out of range (1-65535)" };
    if (typeof cfg['database'] != 'string' || cfg['database'].length == 0)
        return void { error: "DbConfig.database: must be a non-empty string" };
    return cfg;
};
const _envConfig = (env) => {
    if (typeof env != 'object')
        return void { error: "EnvConfig: expected an object" };
    let nodeEnv = env['NODE_ENV'];
    if (nodeEnv != 'development' && nodeEnv != 'production' && nodeEnv != 'test')
        return void { error: "EnvConfig.NODE_ENV: must be 'development', 'production', or 'test', got '" + nodeEnv + "'" };
    if (typeof env['PORT'] != 'number')
        return void { error: "EnvConfig.PORT: must be a number" };
    if (typeof env['DATABASE_URL'] != 'string' || env['DATABASE_URL'].length == 0)
        return void { error: "EnvConfig.DATABASE_URL: must be a non-empty string" };
    if (typeof env['DEBUG'] != 'boolean')
        return void { error: "EnvConfig.DEBUG: must be a boolean" };
    return env;
};
// ---- Object transforms ----
const _camelCaseKeys = (obj) => {
    let result = {};
    for (let k of Object.keys(obj)) {
        let [first, ...rest] = k.split('_');
        let newKey = first + rest.map((p) => capitalize(p)).join('');
        result[newKey] = obj[k];
    }
    return result;
};
const _buildUrl = (cfg) => {
    if (typeof cfg != 'object')
        return void "never";
    let url = cfg['protocol'] + '://' + cfg['host'];
    if (typeof cfg['port'] == 'number')
        url = url + ':' + cfg['port'];
    if (typeof cfg['path'] == 'string')
        url = url + cfg['path'];
    return url;
};
const _parseDate = (s) => {
    if (typeof s != 'string' || s.length != 10)
        return void "never";
    if (s.slice(4, 5) != '-' || s.slice(7, 8) != '-')
        return void "never";
    let digits = '0123456789';
    for (let p of [0, 1, 2, 3, 5, 6, 8, 9]) {
        if (!digits.includes(s.slice(p, p + 1)))
            return void "never";
    }
    let year = digits.indexOf(s.slice(0, 1)) * 1000 + digits.indexOf(s.slice(1, 2)) * 100 + digits.indexOf(s.slice(2, 3)) * 10 + digits.indexOf(s.slice(3, 4));
    let month = digits.indexOf(s.slice(5, 6)) * 10 + digits.indexOf(s.slice(6, 7));
    let day = digits.indexOf(s.slice(8, 9)) * 10 + digits.indexOf(s.slice(9, 10));
    if (month < 1 || month > 12 || day < 1 || day > 31)
        return void "never";
    let result = {};
    result['year'] = year;
    result['month'] = month;
    result['day'] = day;
    return result;
};
// ---- Parsers ----
const _parseRoute = (path) => {
    if (typeof path != 'string' || !path.startsWith('/api/'))
        return void "never";
    let parts = path.slice(5).split('/');
    if (parts.length < 2)
        return void "never";
    let result = {};
    result['version'] = parts[0];
    result['resource'] = parts[1];
    if (parts.length > 2)
        result['id'] = parts[2];
    return result;
};
const _hexToRgb = (s) => {
    if (typeof s != 'string' || s.length != 7 || s.slice(0, 1) != '#')
        return void "never";
    let hex = '0123456789abcdef';
    let result = {};
    let r1 = hex.indexOf(s.slice(1, 2).toLowerCase());
    let r2 = hex.indexOf(s.slice(2, 3).toLowerCase());
    let g1 = hex.indexOf(s.slice(3, 4).toLowerCase());
    let g2 = hex.indexOf(s.slice(4, 5).toLowerCase());
    let b1 = hex.indexOf(s.slice(5, 6).toLowerCase());
    let b2 = hex.indexOf(s.slice(6, 7).toLowerCase());
    if (r1 == -1 || r2 == -1 || g1 == -1 || g2 == -1 || b1 == -1 || b2 == -1)
        return void "never";
    result['r'] = r1 * 16 + r2;
    result['g'] = g1 * 16 + g2;
    result['b'] = b1 * 16 + b2;
    return result;
};
const _parseSemver = (s) => {
    if (typeof s != 'string')
        return void "never";
    let parts = s.split('.');
    if (parts.length != 3)
        return void "never";
    let digits = '0123456789';
    let nums = [];
    for (let part of parts) {
        if (part.length == 0)
            return void "never";
        let n = 0;
        let i = 0;
        while (i < part.length) {
            let d = digits.indexOf(part.slice(i, i + 1));
            if (d == -1)
                return void "never";
            n = n * 10 + d;
            i = i + 1;
        }
        nums = [...nums, n];
    }
    let result = {};
    result['major'] = nums[0];
    result['minor'] = nums[1];
    result['patch'] = nums[2];
    return result;
};
const _parseConnectionString = (s) => {
    if (typeof s != 'string')
        return void "never";
    let protocolEnd = s.indexOf('://');
    if (protocolEnd == -1)
        return void "never";
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
    if (database.length > 0)
        result['database'] = database;
    return result;
};
// ---- Pick / Omit ----
const _pick = (obj, fields) => {
    if (typeof obj != 'object')
        return void "never";
    let result = {};
    for (let f of fields) {
        if (obj[f] != undefined)
            result[f] = obj[f];
    }
    return result;
};
const _omit = (obj, fields) => {
    if (typeof obj != 'object')
        return void "never";
    let result = {};
    for (let k of Object.keys(obj)) {
        if (!fields.includes(k))
            result[k] = obj[k];
    }
    return result;
};
const _enrichUser = (u) => {
    if (typeof u != 'object')
        return void "never";
    if (typeof u['firstName'] != 'string' || typeof u['lastName'] != 'string')
        return void "never";
    let result = {};
    result['firstName'] = u['firstName'];
    result['lastName'] = u['lastName'];
    result['email'] = u['email'];
    result['fullName'] = u['firstName'] + ' ' + u['lastName'];
    result['handle'] = u['firstName'].toLowerCase() + u['lastName'].toLowerCase();
    return result;
};
// ---- Validators with custom error messages ----
// Use void "error:message" to produce compile-time diagnostics.
const _strictEmail = (s) => {
    if (typeof s != 'string')
        return void { error: "Expected a string, got " + typeof s };
    let at = s.indexOf('@');
    if (at < 1)
        return void { error: "Invalid email '" + s + "': missing @ sign" };
    let domain = s.slice(at + 1);
    if (!domain.includes('.'))
        return void { error: "Invalid email '" + s + "': domain has no dot" };
    return s;
};
const _strictPort = (n) => {
    if (typeof n != 'number')
        return void { error: "Port must be a number, got " + typeof n };
    if (n < 1 || n > 65535)
        return void { error: "Port " + n + " is out of range (1-65535)" };
    if (Math.floor(n) != n)
        return void { error: "Port must be an integer, got " + n };
    return n;
};
const _strictPositive = (n) => {
    if (typeof n != 'number')
        return void { error: "Expected a number, got " + typeof n };
    if (n <= 0)
        return void { error: "Expected a positive number, got " + n };
    return n;
};
export {};
//// [zod_test.js]
// Consumer file - imports type aliases and uses them.
export {};
