Kala - A **single-threaded**, lightweight, dynamically-typed programming language designed for **scripting**, **embedding** in-database execution of client scripts**.

---

## Key Features

* **Single-threaded Execution** — Simple, predictable, easy to embed
* **Dynamically Typed** — Flexible for scripting and rapid prototyping
* **Embeddable VM** — Designed to integrate inside databases, applications, or services
* **Custom Language** — Tailored for lightweight scripting tasks
* **Written in Pure Go** — No external C dependencies

## Use Cases

* Embedding custom script logic inside databases
* Safe execution of user-defined scripts in servers
* Building lightweight programmable components in Go applications
* Prototyping new language features for scripting environments

---

## Why Single-Threaded?

* Easier to reason about for scripting
* Reduced complexity for embedding in constrained environments
* Eliminates concurrency hazards for isolated script execution

---

## Language Overview

* Dynamically typed: No type declarations needed
* Types supported: `Nil`, `Number`, `String`, `Dict`, `List`, `Function`
* Control structures: `if`, `while`, `for`
* Functions and simple standard library
* Future support planned for user-defined functions and more complex data types

---

## Embedding in Databases

Ideal for:

* Running client-provided scripts within query pipelines
* Safely isolating execution from core database processes
* Adding programmable logic without external dependencies

---

## Roadmap

* [ ] Improve standard library functions
* [ ] Sandbox resource limits (CPU time, memory)
* [ ] Debugging tools
* [ ] Extend language syntax and features

---

## Contributing

Contributions welcome! Feel free to open issues, suggest features, or submit PRs.

---

## Inspiration

Inspired by minimalist VMs like Lua and WASM, but tailored for Go projects needing simple, safe, and embeddable scripting.
