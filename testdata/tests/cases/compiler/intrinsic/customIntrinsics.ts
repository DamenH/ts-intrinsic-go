// @module: esnext
// @moduleResolution: bundler
// Core tests for Intrinsic<typeof fn, [Args]>.

const capitalize = (s: string) => s.charAt(0).toUpperCase() + s.slice(1);
function range(startOrEnd: number, end?: number): number[] {
    const s = end === undefined ? 0 : startOrEnd;
    const e = end === undefined ? startOrEnd : end;
    return Array.from({ length: Math.max(e - s, 0) }, (_, i) => s + i);
}

// --- Arithmetic ---

const add = (a: number, b: number) => a + b;
const sub = (a: number, b: number) => a - b;
const mul = (a: number, b: number) => a * b;
const div = (a: number, b: number) => a / b;
const mod = (a: number, b: number) => a % b;
const neg = (n: number) => -n;

type Add<A extends number, B extends number> = Intrinsic<typeof add, [A, B]>;
type Sub<A extends number, B extends number> = Intrinsic<typeof sub, [A, B]>;
type Mul<A extends number, B extends number> = Intrinsic<typeof mul, [A, B]>;

type R01 = Add<10, 20>;                          // 30
type R02 = Sub<100, 42>;                          // 58
type R03 = Mul<6, 7>;                             // 42
type R04 = Intrinsic<typeof div, [100, 4]>;       // 25
type R05 = Intrinsic<typeof mod, [17, 5]>;        // 2
type R06 = Intrinsic<typeof neg, [99]>;           // -99
const absVal = (n: number) => Math.abs(n);
type R07 = Intrinsic<typeof absVal, [-50]>;       // 50

// Chaining
type R08 = Mul<Add<10, 20>, 3>;                   // 90
type R09 = Sub<Mul<Add<2, 3>, 4>, 1>;             // 19

// --- Comparisons ---

const isPositive = (n: number) => n > 0;
const isEven = (n: number) => n % 2 == 0;

type R10 = Intrinsic<typeof isPositive, [5]>;     // true
type R11 = Intrinsic<typeof isPositive, [-3]>;    // false
type R12 = Intrinsic<typeof isEven, [4]>;         // true
type R13 = Intrinsic<typeof isEven, [3]>;         // false

// --- Strings ---

const toUpper = (s: string) => s.toUpperCase();
const toLower = (s: string) => s.toLowerCase();
const strLen = (s: string) => s.length;

type R14 = Intrinsic<typeof toUpper, ["hello"]>;  // "HELLO"
type R15 = Intrinsic<typeof toLower, ["LOUD"]>;   // "loud"
type R16 = Intrinsic<typeof strLen, ["hello world"]>; // 11

// --- Math builtins ---

const clamp = (n: number, lo: number, hi: number) => Math.min(Math.max(n, lo), hi);

const mySqrt = (n: number) => Math.sqrt(n);
const myPow = (a: number, b: number) => Math.pow(a, b);
type R17 = Intrinsic<typeof mySqrt, [16]>;         // 4
type R18 = Intrinsic<typeof myPow, [2, 10]>;       // 1024
type R19 = Intrinsic<typeof clamp, [-5, 0, 100]>;  // 0
type R20 = Intrinsic<typeof clamp, [50, 0, 100]>;  // 50
type R21 = Intrinsic<typeof clamp, [999, 0, 100]>; // 100

// --- Objects ---

const makeObj = (name: string, age: number) => {
    let r: Record<string, any> = {};
    r['name'] = name;
    r['age'] = age;
    return r;
};
type R22 = Intrinsic<typeof makeObj, ["Alice", 30]>;
// { name: "Alice", age: 30 }

// --- Tuples ---

const head = (t: any[]) => t[0];
const tail = (t: any[]) => t.slice(1);
const myRange = (n: number) => { let arr: number[] = []; let i = 0; while (i < n) { arr = [...arr, i]; i = i + 1; } return arr; };

type R23 = Intrinsic<typeof head, [[1, 2, 3]]>;    // 1
type R24 = Intrinsic<typeof tail, [[1, 2, 3]]>;    // [2, 3]
type R25 = Intrinsic<typeof myRange, [5]>;          // [0, 1, 2, 3, 4]

// --- Higher-order: map, filter, reduce ---

const keepPositive = (t: number[]) => t.filter((x: number) => x > 0);
const doubled = (t: number[]) => t.map((x: number) => x * 2);
const sum = (t: number[]) => t.reduce((a: number, b: number) => a + b, 0);
const unique = (t: any[]) => t.filter((x: any, i: number) => t.indexOf(x) == i);

type R26 = Intrinsic<typeof keepPositive, [[1, -2, 3, -4, 5]]>;  // [1, 3, 5]
type R27 = Intrinsic<typeof doubled, [[1, 2, 3]]>;                // [2, 4, 6]
type R28 = Intrinsic<typeof sum, [[1, 2, 3, 4, 5]]>;              // 15
type R29 = Intrinsic<typeof unique, [[1, 2, 1, 3, 2]]>;           // [1, 2, 3]

// --- Control flow ---

const factorial = (n: number) => {
    let result: number = 1;
    let i: number = n;
    while (i > 0) { result = result * i; i = i - 1; }
    return result;
};

// for-of loop
const sumLoop = (t: number[]) => {
    let s: number = 0;
    for (let x of t) { s = s + x; }
    return s;
};

// if/else dispatch
const describe = (x: any) => {
    if (typeof x == 'number') return 'a number';
    if (typeof x == 'string') return 'a string';
    if (typeof x == 'boolean') return 'a boolean';
    return 'something else';
};

type R30 = Intrinsic<typeof factorial, [5]>;       // 120
type R31 = Intrinsic<typeof sumLoop, [[1, 2, 3, 4, 5]]>; // 15
type R32 = Intrinsic<typeof describe, [42]>;       // "a number"
type R33 = Intrinsic<typeof describe, ["hi"]>;     // "a string"
type R34 = Intrinsic<typeof describe, [true]>;     // "a boolean"

// --- String transforms ---

const camelCase = (s: string) => {
    let [first, ...rest] = s.split('_');
    return first + rest.map((p: string) => capitalize(p)).join('');
};

// Deep property access: reduce over a split path
const deepGet = (t: any, path: string) =>
    path.split('.').reduce((obj: any, key: string) => obj[key], t);

type R35 = Intrinsic<typeof camelCase, ["user_first_name"]>;       // "userFirstName"
type R36 = Intrinsic<typeof camelCase, ["get_all_items_by_id"]>;   // "getAllItemsById"
type R37 = Intrinsic<typeof deepGet, [{ a: { b: { c: 42 } } }, "a.b.c"]>; // 42

// --- Generics: deferred evaluation ---

type GenericAdd<A extends number, B extends number> = Add<A, B>;
function testGeneric<X extends number>(x: X): GenericAdd<X, 1> {
    return undefined!;
}
