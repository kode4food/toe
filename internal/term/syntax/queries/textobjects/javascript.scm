; Functions
(function_declaration) @function.around
(function_declaration (statement_block) @function.inside)

(function_expression) @function.around
(function_expression (statement_block) @function.inside)

(arrow_function) @function.around
(arrow_function (statement_block) @function.inside)

(method_definition) @function.around
(method_definition (statement_block) @function.inside)

; Classes
(class_declaration) @class.around
(class_declaration (class_body) @class.inside)

(class_expression) @class.around
(class_expression (class_body) @class.inside)

; Parameters and arguments
(formal_parameters) @parameter.around
(formal_parameters) @parameter.inside

(arguments) @parameter.around
(arguments) @parameter.inside

; Call expressions
(call_expression) @call.around
(call_expression (arguments) @call.inside)
