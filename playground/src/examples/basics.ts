// @filename: arithmetic.ts
// Plain JS arithmetic lifted to the type level.

const add = (a: number, b: number) => a + b;
const sub = (a: number, b: number) => a - b;
const mul = (a: number, b: number) => a * b;

type Sum<A extends number, B extends number> = Intrinsic<typeof add, [A, B]>;
type Diff<A extends number, B extends number> = Intrinsic<typeof sub, [A, B]>;
type Prod<A extends number, B extends number> = Intrinsic<typeof mul, [A, B]>;

type A01 = Sum<10, 20>;                  // 30
type A02 = Diff<100, 42>;                 // 58
type A03 = Prod<6, 7>;                    // 42

// Nested calls reduce inside-out.
type A04 = Prod<Sum<10, 20>, 3>;          // 90
type A05 = Diff<Prod<Sum<2, 3>, 4>, 1>;   // 19

// @filename: strings.ts
// Intrinsic bodies can call other extracted functions by name, like
// `cap` here.

const upper = (s: string) => s.toUpperCase();
const lower = (s: string) => s.toLowerCase();
const len = (s: string) => s.length;

const cap = (s: string) => s.charAt(0).toUpperCase() + s.slice(1);

const camelCase = (s: string) => {
    let [first, ...rest] = s.split('_');
    return first + rest.map((p: string) => cap(p)).join('');
};

type S01 = Intrinsic<typeof upper, ["hello"]>;                 // "HELLO"
type S02 = Intrinsic<typeof lower, ["LOUD"]>;                   // "loud"
type S03 = Intrinsic<typeof len, ["hello world"]>;              // 11
type S04 = Intrinsic<typeof camelCase, ["user_first_name"]>;    // "userFirstName"

// @filename: lists.ts
// Higher-order list operations. Inner lambdas need explicit parameter
// types; the DSL will not infer them.

const doubled = (xs: number[]) => xs.map((x: number) => x * 2);
const positives = (xs: number[]) => xs.filter((x: number) => x > 0);
const sum = (xs: number[]) => xs.reduce((a: number, b: number) => a + b, 0);
const unique = (xs: number[]) =>
    xs.filter((x: number, i: number) => xs.indexOf(x) == i);

type L01 = Intrinsic<typeof doubled, [[1, 2, 3]]>;              // [2, 4, 6]
type L02 = Intrinsic<typeof positives, [[1, -2, 3, -4, 5]]>;    // [1, 3, 5]
type L03 = Intrinsic<typeof sum, [[1, 2, 3, 4, 5]]>;            // 15
type L04 = Intrinsic<typeof unique, [[1, 2, 1, 3, 2, 3]]>;      // [1, 2, 3]

// @filename: control-flow.ts
// Loops, typeof branches, and if/else all work. Arrow-style recursion
// is not supported yet, so factorial uses a while loop. Branches want
// loose equality (`==`) and single-quoted strings.

const factorial = (n: number) => {
    let result: number = 1;
    let i: number = n;
    while (i > 0) { result = result * i; i = i - 1; }
    return result;
};

const range = (n: number) => {
    let out: number[] = [];
    let i: number = 0;
    while (i < n) { out = [...out, i]; i = i + 1; }
    return out;
};

const describe = (x: any) => {
    if (typeof x == 'number') return 'a number';
    if (typeof x == 'string') return 'a string';
    return 'something else';
};

type C01 = Intrinsic<typeof factorial, [10]>;       // 3628800
type C02 = Intrinsic<typeof range, [5]>;             // [0, 1, 2, 3, 4]
type C03 = Intrinsic<typeof describe, [42]>;         // "a number"
type C04 = Intrinsic<typeof describe, ["hi"]>;       // "a string"
type C05 = Intrinsic<typeof describe, [true]>;       // "something else"

// @filename: objects.ts
// Structured object results. Shorthand property syntax (`{ name, age }`)
// is not supported yet; use `obj[key] = value` assignment.

const makeUser = (name: string, age: number) => {
    let u: Record<string, any> = {};
    u['name'] = name;
    u['age'] = age;
    u['isAdult'] = age >= 18;
    return u;
};

const makePoint = (x: number, y: number) => {
    let p: Record<string, any> = {};
    p['x'] = x;
    p['y'] = y;
    p['origin'] = x == 0 && y == 0;
    return p;
};

type O01 = Intrinsic<typeof makeUser, ["Alice", 30]>;
type O02 = Intrinsic<typeof makeUser, ["Bob", 12]>;
type O03 = Intrinsic<typeof makePoint, [0, 0]>;
type O04 = Intrinsic<typeof makePoint, [3, 4]>;

// @filename: patterns.ts
// Compose intrinsics with the rest of the type system: mapped types,
// conditional types, and `infer` extraction all work on intrinsic
// results just like any other type.

const add = (a: number, b: number) => a + b;
const len = (s: string) => s.length;

// Mapped type: apply an intrinsic per element of an input tuple.
type Pairs = [[10, 20], [3, 4], [100, 200]];
type Sums = { [K in keyof Pairs]: Intrinsic<typeof add, Pairs[K]> };
type P01 = Sums[0];   // 30
type P02 = Sums[1];   // 7
type P03 = Sums[2];   // 300

// Conditional reduction: only call the intrinsic in one branch.
type LenIfString<X> = X extends string ? Intrinsic<typeof len, [X]> : never;
type P04 = LenIfString<"hello world">;   // 11
type P05 = LenIfString<42>;               // never

// `infer` extraction from a structured intrinsic result.
const parsePoint = (s: string) => {
    let parts = s.split(",");
    let r: Record<string, any> = {};
    r["x"] = parts[0];
    r["y"] = parts[1];
    return r;
};
type Parsed = Intrinsic<typeof parsePoint, ["12,34"]>;
type P06 = Parsed extends { x: infer X } ? X : never;   // "12"
type P07 = Parsed extends { y: infer Y } ? Y : never;   // "34"
