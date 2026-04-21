// @module: esnext
// @moduleResolution: bundler
// Same-file dependency resolution for intrinsic functions.

const double = (n: number) => n * 2;
const useDouble = (x: number) => double(x) + 1;
type D1 = Intrinsic<typeof useDouble, [5]>;  // 11

// Transitive: add4 -> add2 -> add1
const add1 = (n: number) => n + 1;
const add2 = (n: number) => add1(add1(n));
const add4 = (n: number) => add2(add2(n));
type D2 = Intrinsic<typeof add4, [10]>;  // 14

// Shared helper
const normalize = (s: string) => s.trim().toLowerCase();
const greet = (name: string) => "hello " + normalize(name);
type D3 = Intrinsic<typeof greet, ["  World  "]>;  // "hello world"
const shout = (name: string) => normalize(name).toUpperCase();
type D4 = Intrinsic<typeof shout, ["  World  "]>;  // "WORLD"

// Const value dependency
const PREFIX = "test_";
const withPrefix = (s: string) => PREFIX + s;
type D5 = Intrinsic<typeof withPrefix, ["hello"]>;  // "test_hello"

// Multiple helpers
const square = (n: number) => n * n;
const negate = (n: number) => 0 - n;
const negSquare = (x: number) => negate(square(x));
type D6 = Intrinsic<typeof negSquare, [3]>;  // -9

// Helper referenced inside a closure
const isEven = (n: number) => n % 2 == 0;
const filterEvens = (arr: number[]) => arr.filter((x: number) => isEven(x));
type D7 = Intrinsic<typeof filterEvens, [[1, 2, 3, 4, 5, 6]]>;  // [2, 4, 6]

// Object return
const makePoint = (x: number, y: number) => {
    let result: Record<string, any> = {};
    result["x"] = x;
    result["y"] = y;
    return result;
};
const shiftPoint = (x: number, y: number, dx: number, dy: number) => makePoint(x + dx, y + dy);
type D8 = Intrinsic<typeof shiftPoint, [1, 2, 10, 20]>;  // { x: 11, y: 22 }

// Diamond: combined -> doubleInc -> inc, combined -> tripleInc -> inc
const inc = (n: number) => n + 1;
const doubleInc = (n: number) => inc(n) * 2;
const tripleInc = (n: number) => inc(n) * 3;
const combined = (n: number) => doubleInc(n) + tripleInc(n);
type D9 = Intrinsic<typeof combined, [4]>;  // 10 + 15 = 25

// Helper calling helper
const capitalize = (s: string) => {
    if (s.length == 0) return s;
    return s.charAt(0).toUpperCase() + s.slice(1).toLowerCase();
};
const titleCase = (s: string) => s.split(" ").map((w: string) => capitalize(w)).join(" ");
type D10 = Intrinsic<typeof titleCase, ["hello world foo"]>;  // "Hello World Foo"

// Comments in function bodies
const withLineComments = (n: number) => {
    // double it
    let x: number = n * 2;
    // then add one
    return x + 1;
};
type D11 = Intrinsic<typeof withLineComments, [5]>;  // 11

const withBlockComments = (n: number) => {
    let x: number = n /* input */ + 1;
    /* square */
    return x * x;
};
type D12 = Intrinsic<typeof withBlockComments, [3]>;  // 16

const commented_helper = (s: string) => {
    // trim and lowercase
    return s.trim().toLowerCase();
};
const use_commented = (s: string) => commented_helper(s) + "!";
type D13 = Intrinsic<typeof use_commented, ["  HI  "]>;  // "hi!"
