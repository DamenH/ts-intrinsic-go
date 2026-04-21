// @module: esnext
// @moduleResolution: bundler

const NeverType = Symbol.for("tsi.never") as any;

// switch: not supported
const withSwitch = (x: any) => {
    switch (typeof x) {
        case 'string': return x;
        default: return NeverType;
    }
};
type E1 = Intrinsic<typeof withSwitch, ["hello"]>;

// const (only let is supported)
const withConst = (x: number) => {
    const y = x + 1;
    return y;
};
type E2 = Intrinsic<typeof withConst, [5]>;

// optional chaining: not supported
const withOptional = (x: any) => x?.name;
type E3 = Intrinsic<typeof withOptional, [{ name: "Alice" }]>;

// destructured params: not supported
const withDestructure = ({name, age}: {name: string, age: number}) => name;
type E4 = Intrinsic<typeof withDestructure, [{ name: "Alice", age: 30 }]>;

// === (only == is supported)
const withStrict = (x: any) => x === 'hello' ? x : NeverType;
type E5 = Intrinsic<typeof withStrict, ["hello"]>;

// nullish coalescing: not supported
const withNullish = (x: any) => x ?? 'default';
type E6 = Intrinsic<typeof withNullish, [null]>;

// template literal: not supported
const withTemplate = (name: string) => `hello ${name}`;
type E7 = Intrinsic<typeof withTemplate, ["world"]>;

// string form is not supported
type E8 = Intrinsic<"not valid dsl at all", [42]>;

// runtime error: division by zero produces Infinity, not an error
const divByZero = (n: number) => n / 0;
type E9 = Intrinsic<typeof divByZero, [42]>;

// runtime error: property access on number
const badAccess = (n: any) => n.foo;
type E10 = Intrinsic<typeof badAccess, [42]>;

// budget exceeded: infinite loop
const infinite = (n: number) => {
    let i: number = 0;
    while (i < n) { i = i + 1; }
    return i;
};
type E11 = Intrinsic<typeof infinite, [100000]>;
