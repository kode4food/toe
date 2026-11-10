; Functions
(function_definition) @function.around
(function_definition (compound_statement) @function.inside)

; Types
(struct_specifier) @class.around
(struct_specifier (field_declaration_list) @class.inside)

(union_specifier) @class.around
(union_specifier (field_declaration_list) @class.inside)

(enum_specifier) @class.around
(enum_specifier (enumerator_list) @class.inside)

; Parameters and arguments
(parameter_list) @parameter.around
(parameter_list) @parameter.inside

(argument_list) @parameter.around
(argument_list) @parameter.inside

; Call expressions
(call_expression) @call.around
(call_expression (argument_list) @call.inside)

; Initializer entries
(initializer_pair) @entry.around
