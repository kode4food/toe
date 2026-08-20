package ui

var completionKindIcons = map[string]string{
	"text":        "\uea93", // '' - symbol-text: text/string icon
	"function":    "\uea8c", // '' - symbol-method: function/method icon
	"method":      "\uea8c", // '' - symbol-method: function/method icon
	"constructor": "\uea8c", // '' - symbol-method: function/method icon
	"field":       "\ueb5f", // '' - symbol-field: field icon
	"variable":    "\uea88", // '' - symbol-variable: variable icon
	"class":       "\ueb5b", // '' - symbol-class: class icon
	"interface":   "\ueb61", // '' - symbol-interface: interface icon
	"module":      "\uea8b", // '' - symbol-module: module/package icon
	"property":    "\ueb65", // '' - symbol-property: property icon
	"unit":        "\uea96", // '' - symbol-ruler: unit/ruler icon
	"value":       "\uea95", // '' - symbol-enum: enum/value icon
	"enum":        "\uea95", // '' - symbol-enum: enum/value icon
	"keyword":     "\ueb62", // '' - symbol-keyword: keyword icon
	"snippet":     "\ueb66", // '' - symbol-snippet: snippet icon
	"color":       "\ueb5c", // '' - symbol-color: color swatch icon
	"file":        "\uea7b", // '' - symbol-file: file icon
	"reference":   "\uea94", // '' - symbol-reference: reference icon
	"folder":      "\uea83", // '' - symbol-folder: folder icon
	"constant":    "\ueb5d", // '' - symbol-constant: constant icon
	"struct":      "\uea91", // '' - symbol-structure: struct icon
	"event":       "\uea86", // '' - symbol-event: event icon
	"operator":    "\ueb64", // '' - symbol-operator: operator icon
	"type_param":  "\uea92", // '' - symbol-parameter: type-parameter icon
	"enum_member": "\ueb5e", // '' - symbol-enum-member: enum-member icon
}

var completionKindAscii = map[string]string{
	"text":        "Txt",
	"function":    "Fun",
	"method":      "Mth",
	"constructor": "Ctr",
	"field":       "Fld",
	"variable":    "Var",
	"class":       "Cls",
	"interface":   "Ifc",
	"module":      "Mod",
	"property":    "Prp",
	"unit":        "Unt",
	"value":       "Val",
	"enum":        "Enm",
	"keyword":     "Kwd",
	"snippet":     "Snp",
	"color":       "Clr",
	"file":        "Fil",
	"reference":   "Ref",
	"folder":      "Dir",
	"constant":    "Cst",
	"struct":      "Sct",
	"event":       "Evt",
	"operator":    "Opr",
	"type_param":  "Tpm",
	"enum_member": "Emb",
}

func completionKindMarker(kind string, nerd bool) string {
	if kind == "" {
		return ""
	}
	icon := completionKindIcon(kind, nerd)
	if icon == "" {
		return "?"
	}
	return icon
}

func completionKindIcon(kind string, nerd bool) string {
	if nerd {
		return completionKindIcons[kind]
	}
	return completionKindAscii[kind]
}
