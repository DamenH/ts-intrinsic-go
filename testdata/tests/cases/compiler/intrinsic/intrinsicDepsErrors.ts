// @module: esnext
// @moduleResolution: bundler

// let declaration: not extracted
let mutableHelper = (n: number) => n + 1;
const usesMutable = (x: number) => mutableHelper(x);
type E1 = Intrinsic<typeof usesMutable, [5]>;

// Inner function: works via DSL's own let handling
const outerFn = (x: number) => {
    let innerHelper = (n: number) => n * 2;
    return innerHelper(x);
};
type E2 = Intrinsic<typeof outerFn, [5]>;  // 10

// Undefined reference
const callsGhost = (x: number) => ghostFn(x);
type E3 = Intrinsic<typeof callsGhost, [5]>;

// var declaration: not extracted
var varHelper = (n: number) => n + 1;
const usesVar = (x: number) => varHelper(x);
type E4 = Intrinsic<typeof usesVar, [5]>;

// Const value dependency
const NUM_VALUE = 42;
const usesNum = (x: number) => x + NUM_VALUE;
type E5 = Intrinsic<typeof usesNum, [8]>;  // 50

// Const expression dependency
const computed = 2 + 3;
const usesComputed = (x: number) => x + computed;
type E6 = Intrinsic<typeof usesComputed, [10]>;  // 15

// Division is not confused with comments
const divides = (a: number, b: number) => a / b;
type E7 = Intrinsic<typeof divides, [10, 2]>;  // 5
