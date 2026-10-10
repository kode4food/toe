; Identifiers

(identifier) @variable

(parameter
  name: (identifier) @variable.parameter)

; Types

(parameter
  type: (identifier) @type)

((identifier) @type
  (#match? @type "^[A-Z_][a-zA-Z0-9_]*$"))

[
  (builtin_type)
  "anyframe"
] @type.builtin

; Constants

((identifier) @constant
  (#match? @constant "^[A-Z][A-Z_0-9]+$"))

[
  "null"
  "unreachable"
  "undefined"
] @constant.builtin

(field_expression
  .
  member: (identifier) @constant)

(enum_declaration
  (container_field
    type: (identifier) @constant))

; Labels

(block_label (identifier) @label)

(break_label (identifier) @label)

; Fields

(field_initializer
  .
  (identifier) @variable.other.member)

(field_expression
  (_)
  member: (identifier) @variable.other.member)

(container_field
  name: (identifier) @variable.other.member)

; Functions

(builtin_identifier) @function.builtin

(call_expression
  function: (identifier) @function)

(call_expression
  function: (field_expression
    member: (identifier) @function.method))

(function_declaration
  name: (identifier) @function)

; Builtins

[
  "c"
  "..."
] @variable.builtin

((identifier) @variable.builtin
  (#eq? @variable.builtin "_"))

(calling_convention
  (identifier) @variable.builtin)

; Keywords

[
  "asm"
  "defer"
  "errdefer"
  "test"
  "error"
] @keyword

[
  "const"
  "var"
  "struct"
  "union"
  "enum"
  "opaque"
] @keyword.storage.type

[
  "async"
  "await"
  "suspend"
  "nosuspend"
  "resume"
  "try"
  "catch"
] @keyword.control

"fn" @keyword.function

[
  "and"
  "or"
  "orelse"
] @keyword.operator

"return" @keyword.control.return

[
  "if"
  "else"
  "switch"
] @keyword.control.conditional

[
  "for"
  "while"
  "break"
  "continue"
] @keyword.control.repeat

[
  "usingnamespace"
  "export"
] @keyword.control.import

[
  "volatile"
  "allowzero"
  "noalias"
  "addrspace"
  "align"
  "callconv"
  "linksection"
  "pub"
  "inline"
  "noinline"
  "extern"
  "comptime"
  "packed"
  "threadlocal"
] @keyword.storage.modifier

; Operators

[
  "="
  "*="
  "*%="
  "*|="
  "/="
  "%="
  "+="
  "+%="
  "+|="
  "-="
  "-%="
  "-|="
  "<<="
  "<<|="
  ">>="
  "&="
  "^="
  "|="
  "!"
  "~"
  "-"
  "-%"
  "&"
  "=="
  "!="
  ">"
  ">="
  "<="
  "<"
  "^"
  "|"
  "<<"
  ">>"
  "<<|"
  "+"
  "++"
  "+%"
  "+|"
  "-|"
  "*"
  "/"
  "%"
  "**"
  "*%"
  "*|"
  "||"
  ".*"
  ".?"
  "?"
  ".."
] @operator

; Literals

(character) @constant.character

[
  (string)
  (multiline_string)
] @string

(integer) @constant.numeric.integer

(float) @constant.numeric.float

(boolean) @constant.builtin.boolean

(escape_sequence) @constant.character.escape

; Punctuation

[
  "["
  "]"
  "("
  ")"
  "{"
  "}"
] @punctuation.bracket

[
  ";"
  "."
  ","
  ":"
  "=>"
  "->"
] @punctuation.delimiter

(payload "|" @punctuation.bracket)

; Comments

(comment) @comment

((comment) @comment.block.documentation
  (#match? @comment.block.documentation "^//!"))
