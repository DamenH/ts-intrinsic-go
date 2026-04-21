// @module: esnext
// @moduleResolution: bundler
// @filename: sql_helpers.ts
// Shared helpers for compile-time SQL validation and parsing via Intrinsic<>.

export const ne = { $never$: {} } ;

export function range(startOrEnd: number, end?: number): number[] {
    const s = end === undefined ? 0 : startOrEnd;
    const e = end === undefined ? startOrEnd : end;
    return Array.from({ length: Math.max(e - s, 0) }, (_, i) => s + i);
}

// Collapse runs of whitespace to a single space, preserving single-quoted strings.
export const normalizeSql = (sql: string) => {
    let norm: string = '';
    let lastWasSpace: boolean = true;
    let inStr: boolean = false;
    let i = 0;
    while (i < sql.length) {
        let c: string = sql.slice(i, i + 1);
        if (c == "'") { inStr = !inStr; norm = norm + c; lastWasSpace = false; }
        else if (inStr) { norm = norm + c; lastWasSpace = false; }
        else if (c == ' ' || c == '\n' || c == '\t') {
            if (!lastWasSpace) { norm = norm + ' '; lastWasSpace = true; }
        } else { norm = norm + c; lastWasSpace = false; }
        i = i + 1;
    }
    return norm.trim();
};

// Find a SQL keyword at a word boundary in an uppercased string. Returns index or -1.
export const findKw = (upper: string, kw: string) => {
    let idx: number = upper.indexOf(kw);
    while (idx != -1) {
        let before: boolean = idx == 0 || upper.slice(idx - 1, idx) == ' ' || upper.slice(idx - 1, idx) == '(';
        let end: number = idx + kw.length;
        let after: boolean = end == upper.length || upper.slice(end, end + 1) == ' ' || upper.slice(end, end + 1) == '(' || upper.slice(end, end + 1) == ')';
        if (before && after) return idx;
        idx = upper.indexOf(kw, idx + 1);
    }
    return -1;
};

// Extract $1, $2, ... parameter placeholders from a SQL string.
export const findParams = (s: string) => {
    let params: string[] = [];
    let i = 0;
    while (i < s.length - 1) {
        if (s.slice(i, i + 1) == '$') {
            let j: number = i + 1;
            while (j < s.length && '0123456789'.includes(s.slice(j, j + 1))) { j = j + 1; }
            if (j > i + 1) {
                let p: string = s.slice(i, j);
                if (!params.includes(p)) params = [...params, p];
            }
        }
        i = i + 1;
    }
    return params;
};

// Extract the first word (table name) from a string. Returns [table, rest].
export const extractTable = (s: string) => {
    let spaceIdx: number = s.indexOf(' ');
    if (spaceIdx == -1) return [s, ''];
    return [s.slice(0, spaceIdx), s.slice(spaceIdx + 1)];
};

// Extract WHERE clause, trimming anything after ORDER BY / LIMIT / GROUP BY.
export const extractWhere = (norm: string, upper: string) => {
    let whereIdx: number = findKw(upper, 'WHERE');
    if (whereIdx == -1) return '';
    let afterWhere: string = norm.slice(whereIdx + 6).trim();
    let endIdx: number = afterWhere.length;
    let orderCheck: number = afterWhere.toUpperCase().indexOf(' ORDER BY ');
    let limitCheck: number = afterWhere.toUpperCase().indexOf(' LIMIT ');
    let groupCheck: number = afterWhere.toUpperCase().indexOf(' GROUP BY ');
    if (orderCheck != -1 && orderCheck < endIdx) endIdx = orderCheck;
    if (limitCheck != -1 && limitCheck < endIdx) endIdx = limitCheck;
    if (groupCheck != -1 && groupCheck < endIdx) endIdx = groupCheck;
    return afterWhere.slice(0, endIdx).trim();
};

// Extract ORDER BY clause value.
export const extractOrderBy = (norm: string, upper: string) => {
    let orderIdx: number = findKw(upper, 'ORDER BY');
    if (orderIdx == -1) return '';
    let afterOrder: string = norm.slice(orderIdx + 9).trim();
    let limitCheck: number = afterOrder.toUpperCase().indexOf(' LIMIT ');
    return limitCheck == -1 ? afterOrder : afterOrder.slice(0, limitCheck).trim();
};

// Extract LIMIT clause value.
export const extractLimit = (norm: string, upper: string) => {
    let limitIdx: number = findKw(upper, 'LIMIT');
    if (limitIdx == -1) return '';
    let afterLimit: string = norm.slice(limitIdx + 6).trim();
    let offsetCheck: number = afterLimit.toUpperCase().indexOf(' OFFSET ');
    return offsetCheck == -1 ? afterLimit : afterLimit.slice(0, offsetCheck).trim();
};

// Split comma-separated items and trim each.
export const splitTrim = (s: string) => s.split(',').map((c: string) => c.trim());

// Parse CREATE TABLE statements from a schema string into { tableName: [col, ...] }.
export const parseSchema = (schemaSql: string) => {
    let tables: Record<string, any> = {};
    let stmts: string[] = schemaSql.split(';');
    for (let stmt of stmts) {
        let s: string = stmt.trim();
        if (s.length == 0) { }
        else if (s.toUpperCase().startsWith('CREATE TABLE ')) {
            let afterCT: string = s.slice(13).trim();
            if (afterCT.toUpperCase().startsWith('IF NOT EXISTS ')) {
                afterCT = afterCT.slice(14).trim();
            }
            let parenIdx: number = afterCT.indexOf('(');
            if (parenIdx == -1) return 'Error: invalid schema: missing ( in CREATE TABLE';
            let tableName: string = afterCT.slice(0, parenIdx).trim();
            let endParen: number = -1;
            let ei = 0;
            while (ei < afterCT.length) {
                if (afterCT.slice(ei, ei + 1) == ')') endParen = ei;
                ei = ei + 1;
            }
            if (endParen == -1) return 'Error: invalid schema: missing ) in CREATE TABLE';
            let colDefs: string = afterCT.slice(parenIdx + 1, endParen);
            let colNames: string[] = [];
            let parts: string[] = colDefs.split(',');
            for (let part of parts) {
                let p: string = part.trim();
                if (p.length > 0) {
                    let firstWord: string = p.includes(' ') ? p.slice(0, p.indexOf(' ')) : p;
                    let upper: string = firstWord.toUpperCase();
                    if (upper != 'PRIMARY' && upper != 'FOREIGN' && upper != 'UNIQUE'
                        && upper != 'CHECK' && upper != 'CONSTRAINT') {
                        colNames = [...colNames, firstWord];
                    }
                }
            }
            tables[tableName] = colNames;
        }
    }
    return tables;
};

// Resolve selected columns against a schema, returning [cols] or an error string.
export const resolveColumns = (columnsStr: string, validColumns: any, tableName: string) => {
    if (columnsStr == '*') return validColumns;
    let selectedCols: string[] = [];
    let rawCols: string[] = splitTrim(columnsStr);
    for (let col of rawCols) {
        let colUpper: string = col.toUpperCase();
        if (colUpper.startsWith('COUNT(') || colUpper.startsWith('SUM(')
            || colUpper.startsWith('AVG(') || colUpper.startsWith('MIN(')
            || colUpper.startsWith('MAX(')) {
            let inner: string = col.slice(col.indexOf('(') + 1, col.indexOf(')'));
            if (inner != '*' && !validColumns.includes(inner)) {
                return 'Error: column \'' + inner + '\' not found in table \'' + tableName + '\'';
            }
            selectedCols = [...selectedCols, col];
        } else {
            let colName: string = col.includes(' ') ? col.slice(0, col.indexOf(' ')) : col;
            if (!validColumns.includes(colName)) {
                return 'Error: column \'' + colName + '\' not found in table \'' + tableName + '\'';
            }
            selectedCols = [...selectedCols, colName];
        }
    }
    return selectedCols;
};

// Validate a SELECT's WHERE clause columns against the schema.
export const validateWhereColumns = (norm: string, upper: string, validColumns: any, tableName: string) => {
    let w: string = extractWhere(norm, upper);
    if (w.length == 0) return '';
    let firstWord: string = w.includes(' ') ? w.slice(0, w.indexOf(' ')) : w;
    if (firstWord.length > 0 && !firstWord.startsWith('$') && !firstWord.startsWith('(')
        && !firstWord.startsWith("'") && !'0123456789'.includes(firstWord.slice(0, 1))
        && !validColumns.includes(firstWord)) {
        return 'Error: column \'' + firstWord + '\' not found in table \'' + tableName + '\'';
    }
    return '';
};

// @filename: sql.ts
// Compile-time SQL validation and parsing via Intrinsic<>.
// Returns a structured breakdown on success, or a descriptive error string on failure.

import { normalizeSql, findKw, findParams, extractTable, extractWhere, extractOrderBy, extractLimit, splitTrim, parseSchema, resolveColumns, validateWhereColumns } from "./sql_helpers";

// ---- Statement parsers ----

const parseSelect = (norm: string, upper: string) => {
    let fromIdx: number = findKw(upper, 'FROM');
    if (fromIdx == -1) return 'Error: SELECT requires FROM clause';
    let columnsStr: string = norm.slice(7, fromIdx).trim();
    if (columnsStr.length == 0) return 'Error: SELECT requires column list';
    let afterFrom: string = norm.slice(fromIdx + 5).trim();
    if (afterFrom.length == 0) return 'Error: FROM requires table name';
    let [table] = extractTable(afterFrom);
    let result: Record<string, any> = {};
    result['statement'] = 'SELECT';
    result['table'] = table;
    result['columns'] = splitTrim(columnsStr);
    result['params'] = findParams(norm);
    let w: string = extractWhere(norm, upper);
    if (w.length > 0) result['where'] = w;
    let o: string = extractOrderBy(norm, upper);
    if (o.length > 0) result['orderBy'] = o;
    let l: string = extractLimit(norm, upper);
    if (l.length > 0) result['limit'] = l;
    return result;
};

const parseInsert = (norm: string, upper: string) => {
    let afterInsert: string = norm.slice(12).trim();
    let parenIdx: number = afterInsert.indexOf('(');
    let valIdx: number = findKw(upper, 'VALUES');
    if (valIdx == -1) return 'Error: INSERT requires VALUES clause';
    let table: string = '';
    let insertCols: string[] = [];
    if (parenIdx != -1 && parenIdx + 12 < valIdx) {
        table = afterInsert.slice(0, parenIdx).trim();
        let closeParen: number = afterInsert.indexOf(')');
        if (closeParen != -1) insertCols = splitTrim(afterInsert.slice(parenIdx + 1, closeParen));
    } else {
        let [t] = extractTable(afterInsert);
        table = t;
    }
    if (table.length == 0) return 'Error: INSERT requires table name';
    let result: Record<string, any> = {};
    result['statement'] = 'INSERT';
    result['table'] = table;
    if (insertCols.length > 0) result['columns'] = insertCols;
    result['params'] = findParams(norm);
    return result;
};

const parseUpdate = (norm: string, upper: string) => {
    let setIdx: number = findKw(upper, 'SET');
    if (setIdx == -1) return 'Error: UPDATE requires SET clause';
    let table: string = norm.slice(7, setIdx).trim();
    if (table.length == 0) return 'Error: UPDATE requires table name';
    let afterSet: string = norm.slice(setIdx + 4).trim();
    if (afterSet.length == 0) return 'Error: SET requires assignments';
    let whereIdx: number = findKw(upper, 'WHERE');
    let assignments: string = whereIdx == -1 ? afterSet : afterSet.slice(0, whereIdx - setIdx - 4).trim();
    let result: Record<string, any> = {};
    result['statement'] = 'UPDATE';
    result['table'] = table;
    result['set'] = splitTrim(assignments);
    result['params'] = findParams(norm);
    if (whereIdx != -1) result['where'] = norm.slice(whereIdx + 6).trim();
    return result;
};

const parseDelete = (norm: string, upper: string) => {
    let afterFrom: string = norm.slice(12).trim();
    if (afterFrom.length == 0) return 'Error: DELETE requires table name';
    let whereIdx: number = findKw(upper, 'WHERE');
    let table: string = whereIdx == -1 ? afterFrom : afterFrom.slice(0, whereIdx - 12).trim();
    let result: Record<string, any> = {};
    result['statement'] = 'DELETE';
    result['table'] = table;
    result['params'] = findParams(norm);
    if (whereIdx != -1) result['where'] = norm.slice(whereIdx + 6).trim();
    return result;
};

const parseCreateTable = (norm: string) => {
    let parenIdx: number = norm.indexOf('(');
    if (parenIdx == -1) return 'Error: CREATE TABLE requires column definitions';
    let table: string = norm.slice(13, parenIdx).trim();
    if (table.length == 0) return 'Error: CREATE TABLE requires table name';
    if (!norm.endsWith(')')) return 'Error: CREATE TABLE missing closing parenthesis';
    let result: Record<string, any> = {};
    result['statement'] = 'CREATE TABLE';
    result['table'] = table;
    result['definition'] = norm.slice(parenIdx + 1, norm.length - 1).trim();
    return result;
};

// Parse a SELECT and validate its columns/WHERE against a schema.
const parseTypedSelect = (norm: string, upper: string, schema: any) => {
    let fromIdx: number = upper.indexOf(' FROM ');
    if (fromIdx == -1) return 'Error: SELECT requires FROM clause';
    let columnsStr: string = norm.slice(7, fromIdx).trim();
    if (columnsStr.length == 0) return 'Error: SELECT requires column list';
    let afterFrom: string = norm.slice(fromIdx + 6).trim();
    if (afterFrom.length == 0) return 'Error: FROM requires table name';
    let [tableName] = extractTable(afterFrom);
    let validColumns = schema[tableName];
    if (validColumns == undefined) return 'Error: table \'' + tableName + '\' not found in schema';
    let selectedCols = resolveColumns(columnsStr, validColumns, tableName);
    if (typeof selectedCols == 'string') return selectedCols;
    let whereErr: string = validateWhereColumns(norm, upper, validColumns, tableName);
    if (whereErr.length > 0) return whereErr;
    let result: Record<string, any> = {};
    result['statement'] = 'SELECT';
    result['table'] = tableName;
    result['columns'] = selectedCols;
    result['params'] = findParams(norm);
    return result;
};

// ---- Main entry points ----

export const parseSql = (sql: any) => {
    if (typeof sql != 'string' || sql.trim().length == 0) return 'Error: empty query';
    let norm: string = normalizeSql(sql);
    let upper: string = norm.toUpperCase();
    if (upper.startsWith('SELECT ')) return parseSelect(norm, upper);
    if (upper.startsWith('INSERT INTO ')) return parseInsert(norm, upper);
    if (upper.startsWith('UPDATE ')) return parseUpdate(norm, upper);
    if (upper.startsWith('DELETE FROM ')) return parseDelete(norm, upper);
    if (upper.startsWith('CREATE TABLE ')) return parseCreateTable(norm);
    return 'Error: unsupported statement (expected SELECT, INSERT, UPDATE, DELETE, or CREATE TABLE)';
};
export type Sql<T> = Intrinsic<typeof parseSql, [T]>;

// Typed generic wrapper so value-level calls carry the literal argument
// through the return type. `const q = parse("SELECT ...")` then gives
// `q` the concrete parsed object instead of the raw union of all possible
// parseSql return values.
export const parse = <T extends string>(sql: T): Sql<T> =>
    parseSql(sql) as unknown as Sql<T>;

export const typedQuery = (schemaSql: any, sql: any) => {
    if (typeof schemaSql != 'string' || typeof sql != 'string') return 'Error: invalid arguments';
    let schema = parseSchema(schemaSql);
    if (typeof schema == 'string') return schema;
    let norm: string = normalizeSql(sql);
    let upper: string = norm.toUpperCase();
    if (!upper.startsWith('SELECT ')) return 'Error: TypedQuery only supports SELECT';
    return parseTypedSelect(norm, upper, schema);
};
export type TypedQuery<Schema, Query> = Intrinsic<typeof typedQuery, [Schema, Query]>;

// Typed generic wrapper for typedQuery, same idea as `parse` above.
export const query = <S extends string, Q extends string>(schema: S, sql: Q): TypedQuery<S, Q> =>
    typedQuery(schema, sql) as unknown as TypedQuery<S, Q>;


// @filename: sql_test.ts

import { Sql, TypedQuery, parse, query } from "./sql";

// ==== SELECT: structured breakdown ====

type S01 = Sql<"SELECT * FROM users">;
type S02 = Sql<"SELECT name, age FROM users">;
type S03 = Sql<"SELECT name FROM users WHERE id = 1">;
type S04 = Sql<"SELECT * FROM users ORDER BY name ASC">;
type S05 = Sql<"SELECT * FROM users WHERE active = 1 LIMIT 10">;
type S06 = Sql<"SELECT * FROM users WHERE id = $1">;
type S07 = Sql<"SELECT * FROM users WHERE id = $1 AND status = $2">;

// ==== INSERT ====

type I01 = Sql<"INSERT INTO users VALUES ($1, $2, $3)">;
type I02 = Sql<"INSERT INTO users (name, age) VALUES ($1, $2)">;

// ==== UPDATE ====

type U01 = Sql<"UPDATE users SET name = $1 WHERE id = $2">;
type U02 = Sql<"UPDATE users SET active = 0">;

// ==== DELETE ====

type D01 = Sql<"DELETE FROM users WHERE id = $1">;
type D02 = Sql<"DELETE FROM sessions">;

// ==== CREATE TABLE ====

type C01 = Sql<"CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT NOT NULL, age INTEGER)">;

// ==== Errors produce descriptive messages ====
// Hover each alias to see the literal error string the parser returns.
// Messages are produced by the intrinsic body, not by the type system,
// so the reason is specific and human-readable.

type E01 = Sql<"">;
//   empty input: parseSql rejects whitespace-only strings up front.

type E02 = Sql<"SELCT * FROM users">;
//   misspelled keyword: dispatch on the first token falls through every
//   startsWith() branch and hits the "unsupported statement" default.

type E03 = Sql<"SELECT FROM users">;
//   missing column list: parseSelect sees FROM immediately after SELECT
//   and has nothing to put in the columns tuple.

type E04 = Sql<"SELECT *">;
//   missing FROM clause: column list parsed, but no FROM keyword means
//   there is no table to bind the projection to.

type E05 = Sql<"SELECT * FROM">;
//   FROM with no table name: parseSelect reads the FROM keyword but the
//   next token is empty.

type E06 = Sql<"INSERT INTO users">;
//   missing VALUES clause: parseInsert only sees the target table, which
//   is not enough to build a full statement.

type E07 = Sql<"UPDATE users WHERE id = 1">;
//   missing SET clause: parseUpdate requires a SET block before any
//   WHERE filter.

type E08 = Sql<"DROP TABLE users">;
//   unsupported statement: DROP is not one of the verbs this parser
//   handles (SELECT, INSERT, UPDATE, DELETE, CREATE TABLE).

// ==== Complex queries ====

type CQ1 = Sql<"SELECT * FROM users WHERE id IN (SELECT user_id FROM orders)">;
type CQ2 = Sql<"SELECT u.name, o.total FROM users u LEFT JOIN orders o ON u.id = o.user_id">;
type CQ3 = Sql<"SELECT * FROM users WHERE age BETWEEN 18 AND 65">;
type CQ4 = Sql<"SELECT * FROM users WHERE status IS NOT NULL ORDER BY created_at DESC LIMIT 20">;
type CQ5 = Sql<"SELECT name FROM users WHERE email LIKE '%@example.com'">;
type CQ6 = Sql<"UPDATE products SET price = price * 1.1, updated_at = $1 WHERE category = $2">;
type CQ7 = Sql<"DELETE FROM sessions WHERE expires_at < $1 AND user_id IS NOT NULL">;
type CQ8 = Sql<"INSERT INTO order_items (order_id, product_id, qty) VALUES ($1, $2, $3)">;


// ==== Schema-aware validation using CREATE TABLE statements ====

const db = `
CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, email TEXT, age INTEGER, active BOOLEAN);
CREATE TABLE posts (id INTEGER PRIMARY KEY, title TEXT, body TEXT, user_id INTEGER, published BOOLEAN);
CREATE TABLE orders (id INTEGER PRIMARY KEY, user_id INTEGER, total REAL, status TEXT);
` as const;

// Valid queries: columns exist in the schema
type TQ1 = TypedQuery<typeof db, "SELECT name, email FROM users">;
type TQ2 = TypedQuery<typeof db, "SELECT * FROM users">;
type TQ3 = TypedQuery<typeof db, "SELECT title, body FROM posts WHERE published = 1">;
type TQ4 = TypedQuery<typeof db, "SELECT id, total FROM orders WHERE status = $1">;
type TQ5 = TypedQuery<typeof db, "SELECT COUNT(*) FROM orders">;
type TQ6 = TypedQuery<typeof db, "SELECT name FROM users WHERE id = $1 LIMIT 1">;

// Invalid queries. Each one gets the schema parsed from `db` and the
// query validated against it. Hover to see the specific rejection.

type TQ10 = TypedQuery<typeof db, "SELECT phone FROM users">;
//   `phone` is not in the users table (only id, name, email, age, active).

type TQ11 = TypedQuery<typeof db, "SELECT title FROM users">;
//   `title` exists, but only in the posts table, not users.

type TQ12 = TypedQuery<typeof db, "SELECT * FROM sessions">;
//   `sessions` is not declared in the schema at all.

type TQ13 = TypedQuery<typeof db, "SELECT name FROM users WHERE phone = 1">;
//   Projection is fine, but the WHERE references a column that does not
//   exist on users. Error comes from WHERE validation, not SELECT.

type TQ14 = TypedQuery<typeof db, "SELECT SUM(missing) FROM orders">;
//   Aggregate function wraps a column name that is not on orders.


// ==== Value-level call sites ====
// `parse` and `query` are typed generic wrappers. Hover each binding to
// see the intrinsic reduction flow through the return type of the call.
// Success cases resolve to structured objects, failures resolve to the
// literal error string the parser would have returned at runtime.

const q1 = parse("SELECT name, age FROM users WHERE id = $1 ORDER BY name LIMIT 10");
//   Parses cleanly: columns, where, orderBy, limit all recognised.

const q2 = parse("INSERT INTO users (name, email) VALUES ($1, $2)");
//   Insert with explicit column list and two parameter placeholders.

const q3 = parse("SELCT * FROM users");
//   Typo: `SELCT` is not a recognised verb, so the dispatch falls through
//   and tsgo resolves q3 to the "unsupported statement" error literal.

const q4 = parse("DELETE FROM sessions WHERE id = $1");
//   Delete with a WHERE clause and a single parameter.

const tq1 = query(db, "SELECT name, email FROM users WHERE id = 1");
//   Both columns exist on users in `db`, so validation passes.

const tq2 = query(db, "SELECT phone FROM users");
//   `phone` is not a column on the users table in the schema above, so
//   tsgo resolves tq2 to the specific column-not-found error literal.
