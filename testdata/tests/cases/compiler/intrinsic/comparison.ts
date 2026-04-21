// @module: esnext
// @moduleResolution: bundler
// Side-by-side: conditional types vs Intrinsic for the same problems.

const capitalize = (s: string) => s.charAt(0).toUpperCase() + s.slice(1);
function range(startOrEnd: number, end?: number): number[] {
    const s = end === undefined ? 0 : startOrEnd;
    const e = end === undefined ? startOrEnd : end;
    return Array.from({ length: Math.max(e - s, 0) }, (_, i) => s + i);
}

// --- CamelCase: conditional types hit recursion limits ---

type CamelCaseCond<S extends string> =
    S extends `${infer F}_${infer R}`
        ? `${F}${Capitalize<CamelCaseCond<R>>}`
        : S;

type CC1 = CamelCaseCond<"user_name">;
type CC2 = CamelCaseCond<"get_all_items">;

const toCamel = (s: string) => {
    let [first, ...rest] = s.split('_');
    return first + rest.map((p: string) => p.charAt(0).toUpperCase() + p.slice(1)).join('');
};
type CamelCaseI<S extends string> = Intrinsic<typeof toCamel, [S]>;

type CI1 = CamelCaseI<"user_name">;
type CI2 = CamelCaseI<"get_all_items">;
// Would hit TS2589 with conditional types at 35+ segments
type CI3 = CamelCaseI<"a_b_c_d_e_f_g_h_i_j_k_l_m_n_o_p_q_r_s_t_u_v_w_x_y_z_a_b_c_d_e_f_g_h_i_j_k_l_m_n_o">;


// --- String length: not possible with conditional types ---

const strLen = (s: string) => s.length;
type StringLength<S extends string> = Intrinsic<typeof strLen, [S]>;

type L1 = StringLength<"hello">;                   // 5
type L2 = StringLength<"">;                        // 0
type L3 = StringLength<"a longer string to measure">;  // 26


// --- Numeric arithmetic ---

const add = (a: number, b: number) => a + b;
const clamp = (n: number, lo: number, hi: number) => Math.min(Math.max(n, lo), hi);

type Add<A extends number, B extends number> = Intrinsic<typeof add, [A, B]>;
type Clamp<N extends number, Lo extends number, Hi extends number> = Intrinsic<typeof clamp, [N, Lo, Hi]>;

type A1 = Add<100, 200>;                           // 300
type A2 = Add<999, 1>;                             // 1000
type C1 = Clamp<-50, 0, 255>;                      // 0
type C2 = Clamp<300, 0, 255>;                      // 255
type C3 = Clamp<128, 0, 255>;                      // 128


// --- String parsing ---

type IsDateLike<S extends string> =
    S extends `${infer Y}${infer Y2}${infer Y3}${infer Y4}-${infer M}${infer M2}-${infer D}${infer D2}`
        ? true : false;

const parseDate = (s: any) => {
    if (typeof s != 'string' || s.length != 10) return void "never";
    if (s.slice(4, 5) != '-' || s.slice(7, 8) != '-') return void "never";
    let digits: string = '0123456789';
    for (let p of [0, 1, 2, 3, 5, 6, 8, 9]) {
        if (!digits.includes(s.slice(p, p + 1))) return void "never";
    }
    let year: number = digits.indexOf(s.slice(0, 1)) * 1000 + digits.indexOf(s.slice(1, 2)) * 100 + digits.indexOf(s.slice(2, 3)) * 10 + digits.indexOf(s.slice(3, 4));
    let month: number = digits.indexOf(s.slice(5, 6)) * 10 + digits.indexOf(s.slice(6, 7));
    let day: number = digits.indexOf(s.slice(8, 9)) * 10 + digits.indexOf(s.slice(9, 10));
    if (month < 1 || month > 12 || day < 1 || day > 31) return void "never";
    let result: Record<string, any> = {};
    result['year'] = year;
    result['month'] = month;
    result['day'] = day;
    return result;
};
type ParseDate<S extends string> = Intrinsic<typeof parseDate, [S]>;

type D1 = ParseDate<"2024-03-15">;                 // { year: 2024, month: 3, day: 15 }
type D2 = ParseDate<"2024-13-01">;                 // never (month 13)
type D3 = ParseDate<"not-a-date">;                 // never


// --- Composition through generics ---

type Apply<F, T> = Intrinsic<F, [T]>;

const parsePositive = (n: any) => {
    if (typeof n != 'number' || n <= 0) return void "never";
    return n;
};
const toUpper = (s: string) => s.toUpperCase();

type P1 = Apply<typeof parsePositive, 42>;            // 42
type P2 = Apply<typeof parsePositive, -1>;            // never
type P3 = Apply<typeof toUpper, "hello">;          // "HELLO"
