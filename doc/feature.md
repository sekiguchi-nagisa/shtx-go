# Supported syntax/features
## Special Parameters
* [x] `@`
* [x] `*`
* [x] `#`
* [x] `?`
* [ ] `-`
* [ ] `$`
* [ ] `!`
* [x] `0`

## Shell Variables (affect shell behavior)
* [ ] `IFS`
  * [x] global
  * [ ] local
* [ ] `PS1`
* [ ] `PROMPT_COMMAND`

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

|                      | global | local | positional |
|----------------------|--------|-------|------------|
| `${parameter:-word}` | ✔️     | ✔️    | ✔️         |
| `${parameter-word}`  | ✔️     | ✔️    | ✔️         |
| `${parameter:=word}` | ✔️     | ✔️    | ✔️         |
| `${parameter=word}`  | ✔️     | ✔️    | ✔️         |
| `${parameter:?word}` | ✔️     | ✔️    | ✔️         |
| `${parameter?word}`  | ✔️     | ✔️    | ✔️         |
| `${parameter:+word}` | ✔️     | ✔️    | ✔️         |
| `${parameter+word}`  | ✔️     | ✔️    | ✔️         |

### Glob Expansion Op
* [x] `?`
* [x] `*`
* [ ] `[^a-z]`

## Commands
* [x] simple command
  * [x] literal command
  * [x] command from variable
* [x] pipeline
* [ ] `!` operator with pipeline
* [x] and/or list
* [ ] asynchronous list
* [ ] group command ``( )``
* [x] group command ``{ }``
* [x] assignment
* [x] if
* [ ] case
  * [x] const glob pattern
  * [x] non-const glob pattern
  * [ ] tilde expansion with quote removal and parameter expansion
* [ ] for
* [ ] while
* [ ] until
* [x] function

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
  * not support options
* [ ] local
  * not support options
* [ ] trap