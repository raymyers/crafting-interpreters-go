# Ate Your Greens (EYG) Syntax Changes

This document summarizes the key syntax changes made to the test examples to reflect the ate-your-greens language syntax.

## Key Syntax Differences

### 1. Variables and Assignment
**Before (Lox):**
```lox
var a = "hello";
a = "world";
```

**After (EYG):**
```eyg
|| {
  a = "hello"
  a = "world"
  a
}
```

### 2. Operators → Builtin Functions
**Before (Lox):**
```lox
2 + 3
5 - 2
4 * 6
8 / 2
3 < 5
1 == 1
```

**After (EYG):**
```eyg
!int_add(2, 3)
!int_subtract(5, 2)
!int_multiply(4, 6)
!int_divide(8, 2)
!int_less_than(3, 5)
!equal(1, 1)
```

### 3. Boolean Values
**Before (Lox):**
```lox
true
false
```

**After (EYG):**
```eyg
True({})
False({})
```

### 4. Nil → Empty Record
**Before (Lox):**
```lox
nil
```

**After (EYG):**
```eyg
{}
```

### 5. Functions
**Before (Lox):**
```lox
fun greet(name) {
  print "Hello, " + name + "!";
}
greet("World");
```

**After (EYG):**
```eyg
|| {
  greet = |name| {
    perform Log(!string_append("Hello, ", !string_append(name, "!")))
  }
  greet("World")
}
```

### 6. Control Flow → Pattern Matching
**Before (Lox):**
```lox
if (condition) {
  doSomething();
} else {
  doSomethingElse();
}
```

**After (EYG):**
```eyg
match condition {
  True(_) -> doSomething()
  False(_) -> doSomethingElse()
}
```

### 7. Loops → Recursive Functions
**Before (Lox):**
```lox
while (foo < 3) {
  print foo;
  foo = foo + 1;
}
```

**After (EYG):**
```eyg
|| {
  loop = !fix(|loop, foo| {
    match !int_less_than(foo, 3) {
      True(_) -> (
        _ = perform Log(!int_to_string(foo))
        loop(!int_add(foo, 1))
      )
      False(_) -> {}
    }
  })
  loop(0)
}
```

## New Language Features

### 1. Records
```eyg
alice = {name: "Alice", age: 30}
alice.name  # Access field
```

### 2. Record Spread
```eyg
bob = {name: "Bob", height: 192}
{height: 100, ..bob}  # {height: 100, name: "Bob"}
```

### 3. Lists
```eyg
items = [1, 2]
items = [10, ..items]  # [10, 1, 2]
```

### 4. Pattern Matching
```eyg
match !int_parse("42") {
  Ok(value) -> value
  Error(_) -> -1
}
```

### 5. Union Types
```eyg
Cat("felix")
Dog("buddy")
```

### 6. Destructuring Assignment
```eyg
{food: f} = {food: "Greens", action: "eat"}
```

### 7. Effects System
```eyg
perform Log("hello")
perform Alert("message")
```

### 8. Effect Handlers
```eyg
handle Alert(
  |value, resume| {
    resume({})
  },
  |_| { {} }
)
```

### 9. Named References
```eyg
std = @std:1
std.list.contains([1], 0)
```

### 10. Higher-Order Functions
```eyg
inc = !int_add(1)
twice = |f, x| { f(f(x)) }
inc2 = twice(inc)
```

## New Tokens

- `!` - Builtin function call prefix
- `||` - Thunk (zero-argument function)
- `->` - Pattern match arrow
- `..` - Spread operator
- `@` - Named reference prefix
- `:` - Colon (for records and named refs)
- `[` `]` - List brackets
- `|` - Lambda parameter separator
- `#` - Comment prefix (instead of `//`)

## Keywords Added

- `match` - Pattern matching
- `perform` - Effect invocation
- `handle` - Effect handler

## Keywords Removed

- `var` - No explicit variable declaration
- `fun` - Functions are lambda expressions
- `if`/`else` - Replaced with pattern matching
- `while`/`for` - Replaced with recursive functions
- `print` - Replaced with `perform Log()`
- `class`/`super`/`this` - No classes in EYG
- `return` - Implicit returns