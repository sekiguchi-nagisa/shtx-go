# Supported syntax/features

## Special Parameters

* [x] `@`
* [x] `*`
* [x] `#`
* [x] `?`
* [ ] `-`
* [x] `$`
* [ ] `!`
* [x] `0`

## Shell Variables and functions (affect shell behavior)

* [x] `IFS`
    * [x] global
    * [x] local
* [ ] `PS1`
* [x] `PROMPT_COMMAND`
    * [x] string variable
    * [x] array variable
* [ ] `BASH_REMATCH`
* [x] `command_not_found_handle`

## Word Expansion

* [ ] tilde expansion
    * [x] tilde expansion without quote removal and parameter expansion
    * [ ] tilde expansion with quote removal and parameter expansion
* [x] parameter expansion
* [x] command substitution
* [ ] arithmetic expansion
* [ ] field splitting
* [ ] glob expansion
    * [ ] literal glob expansion
    * [ ] glob expansion after field splitting

### Parameter Expansion Op

|                              | global | local | positional | `*` | `@` | `array[index]` | `array[*]` | `array[@]` |
|------------------------------|--------|-------|------------|-----|-----|----------------|------------|------------|
| `${parameter:-word}`         | ✔️     | ✔️    | ✔️         | ❌   | ❌   | ✔️             | ✔️         | ❌          |
| `${parameter-word}`          | ✔️     | ✔️    | ✔️         | ❌   | ❌   | ✔️             | ✔️         | ❌          |
| `${parameter:=word}`         | ✔️     | ✔️    | ✔️         | ❌   | ❌   | ✔️             | ✔️         | ❌          |
| `${parameter=word}`          | ✔️     | ✔️    | ✔️         | ❌   | ❌   | ✔️             | ✔️         | ❌          |
| `${parameter:?word}`         | ✔️     | ✔️    | ✔️         | ❌   | ❌   | ✔️             | ✔️         | ❌          |
| `${parameter?word}`          | ✔️     | ✔️    | ✔️         | ❌   | ❌   | ✔️             | ✔️         | ❌          |
| `${parameter:+word}`         | ✔️     | ✔️    | ✔️         | ❌   | ❌   | ✔️             | ✔️         | ❌          |
| `${parameter+word}`          | ✔️     | ✔️    | ✔️         | ❌   | ❌   | ✔️             | ✔️         | ❌          |
| `${#parameter}`              | ✔️     | ✔️    | ❌          | ❌   | ❌   | ❌              | ❌          | ❌          |
| `${parameter/pattern/word}`  | ✔️     | ✔️    | ❌          | ❌   | ❌   | ❌              | ❌          | ❌          |
| `${parameter//pattern/word}` | ✔️     | ✔️    | ❌          | ❌   | ❌   | ❌              | ❌          | ❌          |
| `${parameter/#pattern/word}` | ✔️     | ✔️    | ❌          | ❌   | ❌   | ❌              | ❌          | ❌          |
| `${parameter/%pattern/word}` | ✔️     | ✔️    | ❌          | ❌   | ❌   | ❌              | ❌          | ❌          |
| `${parameter#word}`          | ✔️     | ✔️    | ❌          | ❌   | ❌   | ❌              | ❌          | ❌          |
| `${parameter##word}`         | ✔️     | ✔️    | ❌          | ❌   | ❌   | ❌              | ❌          | ❌          |
| `${parameter%word}`          | ✔️     | ✔️    | ❌          | ❌   | ❌   | ❌              | ❌          | ❌          |
| `${parameter%%word}`         | ✔️     | ✔️    | ❌          | ❌   | ❌   | ❌              | ❌          | ❌          |


### Glob Expansion Op

* [x] `?`
* [x] `*`
* [x] `[^a-z]`

## Array variable

* [ ] ``declare -a AAA=(a b c)``
* [ ] ``declare -a AAA=([index]=a)``
* [x] ``AAA=(a b c)``
* [x] ``AAA=([index]=a)``
* [x] ``${AAA[@]}``
* [x] ``${AAA[*]}``
* [x] ``${AAA[0]}``
* [ ] ``${AAA[<arithmetic expr>]}``
* [x] sparse array
* [ ] negative index

## Commands

* [x] simple command
    * [x] literal command
    * [x] command from variable
* [x] pipeline
* [x] `!` op
    * [x] pipeline
    * [x] and/or list
    * [x] command
    * [x] test command
* [x] and/or list
* [ ] asynchronous list
* [x] group command ``( )``
* [x] group command ``{ }``
* [x] assignment
* [x] if
* [ ] case
    * [x] const glob pattern
    * [x] non-const glob pattern
    * [ ] tilde expansion with quote removal and parameter expansion
* [ ] for
    * [x] iter
    * [ ] c-style
* [ ] while
* [ ] until
* [x] function
* [x] ``[[ ]]``

## Builtins

* [x] echo
* [x] printf
* [x] read
* [x] test
* [x] source
* [x] eval
* [x] command
* [x] shift
* [ ] set
* [x] unset
* [ ] export
    * [x] not support options
* [x] declare
* [x] local
* [x] return
    * [x] return from function
    * [x] return from sourced
* [x] break
* [x] continue
* [x] trap