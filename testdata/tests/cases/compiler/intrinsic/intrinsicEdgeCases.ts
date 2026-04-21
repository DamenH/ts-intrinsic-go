// @module: esnext
// @moduleResolution: bundler

// --- Circular / self-referencing type arguments ---

const identity = (x: any) => x;
type Id<T> = Intrinsic<typeof identity, [T]>;

// Self-referencing (should not infinite-loop)
type Recursive<T> = Id<Recursive<T>>;

// Mutual recursion
type PingPong<T> = Id<PongPing<T>>;
type PongPing<T> = Id<PingPong<T>>;

// Concrete nested use
type R1 = Id<Id<Id<42>>>;  // 42


// --- Deeply nested closures ---

const nestedClosures = (n: number) => {
    let a = (x: number) => {
        let b = (y: number) => {
            let c = (z: number) => {
                let d = (w: number) => w + z + y + x;
                return d(1);
            };
            return c(2);
        };
        return b(3);
    };
    return a(n);
};
type NC1 = Intrinsic<typeof nestedClosures, [10]>;  // 10 + 3 + 2 + 1 = 16

// Closure that builds and calls functions in a loop
const closureLoop = (n: number) => {
    let result: number = 0;
    let i: number = 0;
    while (i < n) {
        let captured: number = i;
        let f = (x: number) => x + captured;
        result = f(result);
        i = i + 1;
    }
    return result;
};
type NC2 = Intrinsic<typeof closureLoop, [10]>;  // 0+0+1+2+...+9 = 45


// --- Closing over top-level const values ---

const MULTIPLIER = 3;
const scaleUp = (n: number) => n * MULTIPLIER;
type CL1 = Intrinsic<typeof scaleUp, [7]>;  // 21

const OFFSET = 100;
const addOffset = (n: number) => n + OFFSET;
type CL2 = Intrinsic<typeof addOffset, [10]>;  // 110

const GREETING = "hi";
const greetWith = (name: string) => GREETING + " " + name;
type CL3 = Intrinsic<typeof greetWith, ["world"]>;  // "hi world"


// --- Union types as arguments ---

const double = (n: number) => n * 2;
type Double<T> = Intrinsic<typeof double, [T]>;

type U1 = Double<5>;          // 10
type U2 = Double<1 | 2 | 3>;  // distribution over union
type U3 = Double<number>;     // deferred (non-literal)

const toUpper = (s: string) => s.toUpperCase();
type Upper<T> = Intrinsic<typeof toUpper, [T]>;
type U4 = Upper<"a" | "b" | "c">;
type U5 = Upper<string>;  // deferred


// --- Type parameter forwarding ---

const add = (a: number, b: number) => a + b;
type Add<A extends number, B extends number> = Intrinsic<typeof add, [A, B]>;

type Increment<N extends number> = Add<N, 1>;
type I1 = Increment<41>;  // 42

type AddTen<N extends number> = Add<N, 10>;
type AddTwenty<N extends number> = AddTen<AddTen<N>>;
type F1 = AddTwenty<0>;   // 20
type F2 = AddTwenty<5>;   // 25

function incr<N extends number>(n: N): Increment<N> {
    return undefined!;
}

type PositiveOrZero<N extends number> = Add<N, 0> extends never ? 0 : Add<N, 0>;
type PZ1 = PositiveOrZero<5>;  // 5


// --- Argument count and type mismatches ---

const greet = (name: string) => "hello " + name;
type Greet<T> = Intrinsic<typeof greet, [T]>;

type A1 = Greet<"world">;  // "hello world"
type A2 = Greet<42>;       // "hello 42" (string + number coercion)

const needsTwo = (a: number, b: number) => a + b;
type A3 = Intrinsic<typeof needsTwo, [5]>;        // second arg is undefined
type A4 = Intrinsic<typeof greet, ["world", 42]>; // extra arg ignored
type A5 = Intrinsic<typeof greet, []>;             // no args
type A6 = Greet<string>;                           // deferred (non-literal)


// --- Semantic mismatches that should stay explicit ---

const kindOf = (x: any) => typeof x;

type S1 = Intrinsic<typeof kindOf, [null]>;        // "null" (not JS "object")
type S2 = Intrinsic<typeof kindOf, [undefined]>;   // "undefined"
type S3 = Intrinsic<typeof kindOf, [[1, 2, 3]]>;   // "tuple" (not JS "object")
type S4 = Intrinsic<typeof kindOf, [{ a: 1 }]>;    // "object"

const kindOfLocalFn = () => {
    let f = (x: number) => x + 1;
    return typeof f;
};

type S5 = Intrinsic<typeof kindOfLocalFn, []>;     // "function"

const looseEq = (a: any, b: any) => a == b;

type S6 = Intrinsic<typeof looseEq, [0, "0"]>;       // false (not JS true)
type S7 = Intrinsic<typeof looseEq, [false, 0]>;      // false (not JS true)
type S8 = Intrinsic<typeof looseEq, [null, undefined]>; // false (not JS true)
type S9 = Intrinsic<typeof looseEq, ["", 0]>;        // false (not JS true)
type S10 = Intrinsic<typeof looseEq, [[1, 2], [1, 2]]>; // true (not JS reference equality)

const objectWithFn = () => {
    let r: any = {};
    r['x'] = 1;
    r['fn'] = (n: number) => n + 1;
    return r;
};

type S11 = Intrinsic<typeof objectWithFn, []>;     // { x: 1, fn: never }
