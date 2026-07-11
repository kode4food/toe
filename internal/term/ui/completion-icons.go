package ui

func completionKindMarker(kind string, mode CompletionIconMode) string {
	if kind == "" || mode == CompletionIconsNone {
		return ""
	}
	var icon string
	if mode == CompletionIconsASCII {
		icon = completionKindASCIIIcon(kind)
	} else {
		icon = completionKindCodicon(kind)
	}
	if icon == "" {
		return "?"
	}
	return icon
}

func completionKindCodicon(kind string) string {
	switch kind {
	case "text":
		return "\uea93" // '' - symbol-text: text/string icon
	case "function", "method", "constructor":
		return "\uea8c" // '' - symbol-method: function/method icon
	case "field":
		return "\ueb5f" // '' - symbol-field: field icon
	case "variable":
		return "\uea88" // '' - symbol-variable: variable icon
	case "class":
		return "\ueb5b" // '' - symbol-class: class icon
	case "interface":
		return "\ueb61" // '' - symbol-interface: interface icon
	case "module":
		return "\uea8b" // '' - symbol-module: module/package icon
	case "property":
		return "\ueb65" // '' - symbol-property: property icon
	case "unit":
		return "\uea96" // '' - symbol-ruler: unit/ruler icon
	case "value", "enum":
		return "\uea95" // '' - symbol-enum: enum/value icon
	case "keyword":
		return "\ueb62" // '' - symbol-keyword: keyword icon
	case "snippet":
		return "\ueb66" // '' - symbol-snippet: snippet icon
	case "color":
		return "\ueb5c" // '' - symbol-color: color swatch icon
	case "file":
		return "\uea7b" // '' - symbol-file: file icon
	case "reference":
		return "\uea94" // '' - symbol-reference: reference icon
	case "folder":
		return "\uea83" // '' - symbol-folder: folder icon
	case "constant":
		return "\ueb5d" // '' - symbol-constant: constant icon
	case "struct":
		return "\uea91" // '' - symbol-structure: struct icon
	case "event":
		return "\uea86" // '' - symbol-event: event icon
	case "operator":
		return "\ueb64" // '' - symbol-operator: operator icon
	case "type_param":
		return "\uea92" // '' - symbol-parameter: type-parameter icon
	case "enum_member":
		return "\ueb5e" // '' - symbol-enum-member: enum-member icon
	default:
		return ""
	}
}

func completionKindASCIIIcon(kind string) string {
	switch kind {
	case "function", "method":
		return "fn"
	case "constructor":
		return "+"
	case "field", "property":
		return "."
	case "variable":
		return "v"
	case "class":
		return "C"
	case "interface":
		return "I"
	case "module":
		return "M"
	case "keyword":
		return "K"
	case "snippet":
		return "S"
	case "file":
		return "F"
	case "folder":
		return "D"
	case "constant":
		return "k"
	case "struct":
		return "S"
	case "enum":
		return "E"
	case "enum_member":
		return "e"
	case "type_param":
		return "T"
	default:
		return ""
	}
}
