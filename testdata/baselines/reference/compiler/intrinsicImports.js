//// [tests/cases/compiler/intrinsic/intrinsicImports.ts] ////

//// [helpers.ts]
export const double = (n: number) => n * 2;
export const add1 = (n: number) => n + 1;
export const capitalize = (s: string) => s.charAt(0).toUpperCase() + s.slice(1);
export const normalize = (s: string) => s.trim().toLowerCase();

//// [validators.ts]
import { capitalize, normalize } from './helpers';

const _toCamelCase = (s: string) => {
    let [first, ...rest] = s.split('_');
    return first + rest.map((p: string) => capitalize(p)).join('');
};

const _clean = (s: string) => normalize(s) + "!";

const _positive = (n: any) => {
    if (typeof n != 'number' || n <= 0) return void "never";
    return n;
};

const _email = (s: any) => {
    if (typeof s != 'string') return void "never";
    let at: number = s.indexOf('@');
    if (at < 1) return void "never";
    let domain: string = s.slice(at + 1);
    if (domain.length < 3 || !domain.includes('.')) return void "never";
    return s;
};

export type ToCamelCase<T> = Intrinsic<typeof _toCamelCase, [T]>;
export type Clean<T> = Intrinsic<typeof _clean, [T]>;
export type Positive<T> = Intrinsic<typeof _positive, [T]>;
export type Email<T> = Intrinsic<typeof _email, [T]>;

//// [chain.ts]
import { capitalize } from './helpers';

const _titleCase = (s: string) => s.split(" ").map((w: string) => capitalize(w)).join(" ");

export type TitleCase<T> = Intrinsic<typeof _titleCase, [T]>;

//// [local_with_import.ts]
import { double, add1 } from './helpers';

const triple = (n: number) => n * 3;

const _compute = (n: number) => double(n) + triple(n) + add1(0);

export type Compute<T> = Intrinsic<typeof _compute, [T]>;

//// [re_export.ts]
export { double, capitalize } from './helpers';

//// [uses_re_export.ts]
import { double, capitalize } from './re_export';

const _doubleAndCap = (s: string) => capitalize(s) + double(1);

export type DoubleAndCap<T> = Intrinsic<typeof _doubleAndCap, [T]>;

//// [consumer.ts]
import type { ToCamelCase, Clean, Positive, Email } from './validators';
import type { TitleCase } from './chain';
import type { Compute } from './local_with_import';
import type { DoubleAndCap } from './uses_re_export';

type C1 = ToCamelCase<"user_first_name">;    // "userFirstName"
type C2 = ToCamelCase<"get_all_items">;       // "getAllItems"
type C3 = Clean<"  HELLO  ">;                // "hello!"
type C4 = Clean<"  World  ">;                // "world!"
type C5 = Positive<5>;                       // 5
type C6 = Positive<-1>;                      // never
type C7 = Positive<0>;                       // never
type C8 = Email<"user@example.com">;          // "user@example.com"
type C9 = Email<"bad">;                       // never
type C10 = TitleCase<"hello world foo">;      // "Hello World Foo"
type C11 = Compute<3>;                        // 6 + 9 + 1 = 16
type C12 = DoubleAndCap<"hello">;             // "Hello2"

//// [non_const_export.ts]
export let mutableHelper = (n: number) => n + 1;

//// [error_cases.ts]
import { mutableHelper } from './non_const_export';

const _usesMutable = (x: number) => mutableHelper(x);
type E1 = Intrinsic<typeof _usesMutable, [5]>;

const _usesGhost = (x: number) => ghostFn(x);
type E2 = Intrinsic<typeof _usesGhost, [5]>;


//// [helpers.js]
export const double = (n) => n * 2;
export const add1 = (n) => n + 1;
export const capitalize = (s) => s.charAt(0).toUpperCase() + s.slice(1);
export const normalize = (s) => s.trim().toLowerCase();
//// [validators.js]
import { capitalize, normalize } from './helpers';
const _toCamelCase = (s) => {
    let [first, ...rest] = s.split('_');
    return first + rest.map((p) => capitalize(p)).join('');
};
const _clean = (s) => normalize(s) + "!";
const _positive = (n) => {
    if (typeof n != 'number' || n <= 0)
        return void "never";
    return n;
};
const _email = (s) => {
    if (typeof s != 'string')
        return void "never";
    let at = s.indexOf('@');
    if (at < 1)
        return void "never";
    let domain = s.slice(at + 1);
    if (domain.length < 3 || !domain.includes('.'))
        return void "never";
    return s;
};
//// [chain.js]
import { capitalize } from './helpers';
const _titleCase = (s) => s.split(" ").map((w) => capitalize(w)).join(" ");
//// [local_with_import.js]
import { double, add1 } from './helpers';
const triple = (n) => n * 3;
const _compute = (n) => double(n) + triple(n) + add1(0);
//// [re_export.js]
export { double, capitalize } from './helpers';
//// [uses_re_export.js]
import { double, capitalize } from './re_export';
const _doubleAndCap = (s) => capitalize(s) + double(1);
//// [consumer.js]
export {};
//// [non_const_export.js]
export let mutableHelper = (n) => n + 1;
//// [error_cases.js]
import { mutableHelper } from './non_const_export';
const _usesMutable = (x) => mutableHelper(x);
const _usesGhost = (x) => ghostFn(x);
