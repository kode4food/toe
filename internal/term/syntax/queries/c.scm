; Generic identifiers first: on the same node, the later pattern wins

(identifier) @variable

((identifier) @constant
  (#match? @constant "^[A-Z][A-Z\\d_]*$"))

(field_identifier) @variable.other.member

(statement_identifier) @label

; Identifiers

(parameter_declaration
  declarator: (identifier) @variable.parameter)

(parameter_declaration
  declarator: (pointer_declarator
    declarator: (identifier) @variable.parameter))

; Types

(type_identifier) @type

[
  (primitive_type)
  (sized_type_specifier)
] @type.builtin

; Functions

(call_expression
  function: (identifier) @function)

(call_expression
  function: (field_expression
    field: (field_identifier) @function))

(function_declarator
  declarator: (identifier) @function)

(function_declarator
  declarator: (parenthesized_declarator
    (pointer_declarator
      declarator: (field_identifier) @function)))

(preproc_function_def
  name: (identifier) @function.macro)

; Constants

(preproc_def
  name: (identifier) @constant)

(enumerator
  name: (identifier) @constant)

; Keywords

[
  "break"
  "continue"
  "goto"
] @keyword.control

[
  "do"
  "for"
  "while"
] @keyword.control.repeat

[
  "case"
  "default"
  "else"
  "if"
  "switch"
] @keyword.control.conditional

"return" @keyword.control.return

[
  "enum"
  "struct"
  "typedef"
  "union"
] @keyword.storage.type

[
  "_Atomic"
  "_Noreturn"
  "auto"
  "const"
  "constexpr"
  "extern"
  "inline"
  "noreturn"
  "register"
  "restrict"
  "static"
  "thread_local"
  "volatile"
  (storage_class_specifier)
  (type_qualifier)
] @keyword.storage.modifier

[
  "_Alignof"
  "_Generic"
  "alignof"
  "asm"
  "defined"
  "offsetof"
  "sizeof"
  "__asm__"
  "__attribute__"
  "__extension__"
] @keyword

; Preprocessor

[
  "#define"
  "#elif"
  "#elifdef"
  "#elifndef"
  "#else"
  "#endif"
  "#if"
  "#ifdef"
  "#ifndef"
  (preproc_directive)
] @keyword.directive

"#include" @keyword.control.import

(preproc_include
  path: (system_lib_string) @string)

; Operators

[
  "--"
  "-"
  "-="
  "->"
  "="
  "!="
  "!"
  "*"
  "*="
  "/"
  "/="
  "%"
  "%="
  "&"
  "&&"
  "&="
  "+"
  "++"
  "+="
  "<"
  "<<"
  "<<="
  "<="
  "=="
  ">"
  ">="
  ">>"
  ">>="
  "^"
  "^="
  "|"
  "|="
  "||"
  "~"
  "?"
] @operator

; Punctuation

[
  "."
  ","
  ";"
  ":"
  "..."
] @punctuation.delimiter

[
  "("
  ")"
  "["
  "]"
  "{"
  "}"
  "[["
  "]]"
] @punctuation.bracket

; Literals

[
  (string_literal)
  (system_lib_string)
] @string

(escape_sequence) @constant.character.escape

(char_literal) @constant.character

(number_literal) @constant.numeric

[
  (true)
  (false)
] @constant.builtin.boolean

(null) @constant.builtin

; Comments

(comment) @comment
