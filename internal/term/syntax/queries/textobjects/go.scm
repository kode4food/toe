; Functions
(function_declaration) @function.around
(function_declaration (block) @function.inside)

(method_declaration) @function.around
(method_declaration (block) @function.inside)

(func_literal) @function.around
(func_literal (block) @function.inside)

; Types
(type_declaration) @class.around
(type_spec (struct_type (field_declaration_list) @class.inside))

; Parameters and arguments
(parameter_list) @parameter.around
(parameter_list) @parameter.inside

(argument_list) @parameter.around
(argument_list) @parameter.inside

; Call expressions
(call_expression) @call.around
(call_expression (argument_list) @call.inside)

; Keyed elements (struct / map literal entries)
(keyed_element) @entry.around
