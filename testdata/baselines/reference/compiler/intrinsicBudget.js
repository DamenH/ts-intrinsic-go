//// [tests/cases/compiler/intrinsic/intrinsicBudget.ts] ////

//// [intrinsicBudget.ts]
// Verify that budget exhaustion does not degrade the rest of the file.

const infinite = (n: number) => {
    let i: number = 0;
    while (i < n) { i = i + 1; }
    return i;
};

const add = (a: number, b: number) => a + b;
const toUpper = (s: string) => s.toUpperCase();
const validate = (n: any) => {
    if (typeof n != 'number') return void { error: "Expected number, got " + typeof n };
    if (n < 0) return void { error: n + " must be non-negative" };
    return n;
};

// Budget exceeded
type Blown = Intrinsic<typeof infinite, [100000]>;

// These should all still resolve correctly after the budget failure above
type A1 = Intrinsic<typeof add, [10, 20]>;           // 30
type A2 = Intrinsic<typeof add, [999, 1]>;           // 1000
type U1 = Intrinsic<typeof toUpper, ["hello"]>;      // "HELLO"
type V1 = Intrinsic<typeof validate, [42]>;           // 42
type V2 = Intrinsic<typeof validate, [-1]>;           // error: -1 must be non-negative
type V3 = Intrinsic<typeof validate, ["x"]>;          // error: Expected number, got string

// Regular TypeScript inference should also be unaffected
const x: number = 42;
const y: string = "hello";
const z = [1, 2, 3].map(n => n * 2);
type Arr = typeof z;                                  // number[]

// A second budget failure should not cascade
type Blown2 = Intrinsic<typeof infinite, [999999]>;

// Still works after second failure
type A3 = Intrinsic<typeof add, [1, 2]>;              // 3


//// [intrinsicBudget.js]
"use strict";
// Verify that budget exhaustion does not degrade the rest of the file.
const infinite = (n) => {
    let i = 0;
    while (i < n) {
        i = i + 1;
    }
    return i;
};
const add = (a, b) => a + b;
const toUpper = (s) => s.toUpperCase();
const validate = (n) => {
    if (typeof n != 'number')
        return void { error: "Expected number, got " + typeof n };
    if (n < 0)
        return void { error: n + " must be non-negative" };
    return n;
};
// Regular TypeScript inference should also be unaffected
const x = 42;
const y = "hello";
const z = [1, 2, 3].map(n => n * 2);
