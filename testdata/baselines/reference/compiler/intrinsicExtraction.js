//// [tests/cases/compiler/intrinsic/intrinsicExtraction.ts] ////

//// [extraction.ts]
// Multiline function body
const multiline = (n: number) => {
    let x: number = n
        + 1;
    let y: number = x
        * 2;
    return y;
};
type EX1 = Intrinsic<typeof multiline, [5]>;  // 12

// Type annotations on parameters and locals
const annotated = (
    s: string,
    n: number,
    flag: boolean
) => {
    let result: string = s;
    let count: number = n;
    if (flag) return result + count;
    return result;
};
type EX2 = Intrinsic<typeof annotated, ["x", 3, true]>;  // "x3"
type EX3 = Intrinsic<typeof annotated, ["x", 3, false]>; // "x"

// Complex type annotations (generics, arrays, unions)
const complexTypes = (
    arr: number[],
    obj: Record<string, any>,
    tuple: [string, number]
) => {
    return arr[0] + obj['x'] + tuple[1];
};
type EX4 = Intrinsic<typeof complexTypes, [[10], { x: 20 }, ["a", 30]]>;  // 60

// Single-expression arrow (no braces)
const oneliner = (a: number, b: number) => a * b + 1;
type EX5 = Intrinsic<typeof oneliner, [3, 4]>;  // 13

// Deeply nested ternary
const nested = (n: number) => n > 100 ? "big" : n > 10 ? "medium" : n > 0 ? "small" : "zero";
type EX6 = Intrinsic<typeof nested, [50]>;   // "medium"
type EX7 = Intrinsic<typeof nested, [0]>;    // "zero"

// Function with no parameters
const noParams = () => 42;
type EX8 = Intrinsic<typeof noParams, []>;  // 42

// String with special characters (quotes, backslashes, newlines)
const specials = (s: string) => s.split('\n').join('|');
type EX9 = Intrinsic<typeof specials, ["a\nb\nc"]>;  // "a|b|c"

// Trailing comma in parameter list
const trailing = (a: number, b: number,) => a + b;
type EX10 = Intrinsic<typeof trailing, [1, 2]>;  // 3

// Function expression: DSL only parses arrow syntax, so this is a parse error
const funcExpr = function(n: number) { return n * 2; };
type EX11 = Intrinsic<typeof funcExpr, [5]>;

// Comments inside the function body
const withComments = (n: number) => {
    // line comment
    let x: number = n + 1; /* inline */
    /*
       block
    */
    return x * 2;
};
type EX12 = Intrinsic<typeof withComments, [4]>;  // 10

// Nested arrow functions (closures)
const outer = (n: number) => {
    let inner = (x: number) => x * 2;
    return inner(n) + inner(n + 1);
};
type EX13 = Intrinsic<typeof outer, [3]>;  // 6 + 8 = 14

// Unusual whitespace and formatting
const   weirdSpacing   =   (   a  :  number  ,  b  :  number  )   =>   a   +   b  ;
type EX14 = Intrinsic<typeof weirdSpacing, [10, 20]>;  // 30


//// [extraction_errors.ts]
// as const in dependency (not extractable, runtime error)
const AS_CONST = "hello" as const;
const usesAsConst = (s: string) => s + AS_CONST;
type ERR1 = Intrinsic<typeof usesAsConst, ["x"]>;

// Function declaration (not const arrow, not extractable as dep)
function helperFn(n: number): number { return n + 1; }
const usesFunc = (n: number) => helperFn(n);
type ERR2 = Intrinsic<typeof usesFunc, [5]>;

// Class method (not an arrow function)
class Foo { bar(n: number) { return n; } }
const foo = new Foo();

// Function declaration: source extraction requires arrow or function expression
function regularFn(n: number) { return n * 2; }
type ERR3 = Intrinsic<typeof regularFn, [5]>;  // deferred (can't extract source)


//// [extraction.js]
"use strict";
// Multiline function body
const multiline = (n) => {
    let x = n
        + 1;
    let y = x
        * 2;
    return y;
};
// Type annotations on parameters and locals
const annotated = (s, n, flag) => {
    let result = s;
    let count = n;
    if (flag)
        return result + count;
    return result;
};
// Complex type annotations (generics, arrays, unions)
const complexTypes = (arr, obj, tuple) => {
    return arr[0] + obj['x'] + tuple[1];
};
// Single-expression arrow (no braces)
const oneliner = (a, b) => a * b + 1;
// Deeply nested ternary
const nested = (n) => n > 100 ? "big" : n > 10 ? "medium" : n > 0 ? "small" : "zero";
// Function with no parameters
const noParams = () => 42;
// String with special characters (quotes, backslashes, newlines)
const specials = (s) => s.split('\n').join('|');
// Trailing comma in parameter list
const trailing = (a, b) => a + b;
// Function expression: DSL only parses arrow syntax, so this is a parse error
const funcExpr = function (n) { return n * 2; };
// Comments inside the function body
const withComments = (n) => {
    // line comment
    let x = n + 1; /* inline */
    /*
       block
    */
    return x * 2;
};
// Nested arrow functions (closures)
const outer = (n) => {
    let inner = (x) => x * 2;
    return inner(n) + inner(n + 1);
};
// Unusual whitespace and formatting
const weirdSpacing = (a, b) => a + b;
//// [extraction_errors.js]
"use strict";
// as const in dependency (not extractable, runtime error)
const AS_CONST = "hello";
const usesAsConst = (s) => s + AS_CONST;
// Function declaration (not const arrow, not extractable as dep)
function helperFn(n) { return n + 1; }
const usesFunc = (n) => helperFn(n);
// Class method (not an arrow function)
class Foo {
    bar(n) { return n; }
}
const foo = new Foo();
// Function declaration: source extraction requires arrow or function expression
function regularFn(n) { return n * 2; }
