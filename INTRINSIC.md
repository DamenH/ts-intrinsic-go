This is a proposal with the goal of generating discussion around an idea I've had for a long time. It started back when TypeScript was still running in Node, with the thought that maybe an intrinsic type could be user definable by letting people write it in js/ts. That was a much easier thing to imagine before the Go rewrite of the compiler was announced, so the idea had to evolve once that changed.

Regardless, I think there is a compelling case for giving TypeScript a real imperative language for compile-time computation. Something expressive enough to define dependent relationships directly instead of forcing everything through conditional types, inference, and template literals. I think a direction like that could substantially expand what TypeScript enables in static analysis.

The proposal is to let users write type-level functions in a syntax that looks a lot like ordinary TypeScript, but that lives in the type-only area of the grammar. The functions never exist at runtime. They are declarations about how a type is computed, and they can live in `.d.ts` files alongside everything else that is already type-only. Something like `Add(A, B) = A + B` declared this way would compile to nothing, import like a type, and compose with other type-level functions by reference, the way conditional types already do.

The reason this prototype does not implement this is that extending TypeScript's grammar is a bigger undertaking than I was willing for the sake of this experiment. You need new AST nodes, new parser work, and you need the tooling. Lint rules, syntax highlighting, editor services. For a prototype meant to explore whether the idea has any legs at all, that cost was too high. So the prototype piggybacks on existing Typescript grammar and linting, adding the `Intrinsic<typeof fn, [Args]>` surface, reusing the existing function parser and the existing type surface, and interprets an extracted subset of TypeScript at compile time. The runtime function is just a place to put the source the compiler reads.

The prototype surface is this:

```ts
// lib.d.ts
type Intrinsic<Fun, Args extends any[] = []> = intrinsic;

// userland.ts
type Result = Intrinsic<typeof someFunction, [Arg1, Arg2]>;
```

The first argument is a function type, written as `typeof fn`. The second is a tuple of compile-time arguments. When the inputs are statically known, the compiler evaluates the extracted body during type checking and uses the result as a type.
```ts
const add = (a: number, b: number) => a + b;
type Add<A extends number, B extends number> = Intrinsic<typeof add, [A, B]>;
```

```ts
const parseEmail = (s: string) => {
    let at = s.indexOf('@');
    if (at < 1) return void { error: "Invalid email: missing @" };
    let domain = s.slice(at + 1);
    if (!domain.includes('.')) return void { error: "Invalid email: bad domain" };
    return s;
};

type Email<T extends string> = Intrinsic<typeof parseEmail, [T]>;
```

See tests for further examples. I demo a mini zod like thing, SQL parsing, and OpenAPI type generation.

There are a few strong decisions baked into this prototype. Compile-time evaluation is deterministic and compiler-owned. There is no I/O, no file system access, no network, no environment, no time or randomness, and no ambient runtime state. The same inputs should produce the same result, and there is an explicit computation and memory budget. Failures are first-class. The prototype supports collapsing to `never`, but it also supports explicit custom diagnostics. And dependencies have to be statically extractable.

The prototype also makes some concessions that would not survive into a real type-level feature, but that are fine under its current framing of extracting and interpreting a body of existing TypeScript. `typeof null` evaluates to `"null"`. `typeof [1, 2, 3]` evaluates to `"tuple"`. `==` does not perform JavaScript coercion. Tuple equality is structural rather than reference-based. Function-valued object properties collapse when converted back into TypeScript types. I do not think those exact semantics should be defended as some ideal endpoint. They exist because the prototype is interpreting runtime-shaped syntax in a context where there is no runtime.

One thing I specifically wanted to learn from building this was what kind of compiler footprint the idea would have. It is significant, but not inscrutable. I was able to do quite a lot while keeping the surface area of changes to existing compiler functionality small. That does not make the complexity disappear, but it does make it easier to reason about, and most of the weight is in the evaluator rather than in changes to the rest of the checker.

I would appreciate thoughts and opinions on the idea itself, and on whether something in this direction could ever make sense as a real feature. I did this mostly because I wanted a playground where I could experiment with dependent types. I hope this is at least thought provoking, and I appreciate anyone taking the time to think about it.
