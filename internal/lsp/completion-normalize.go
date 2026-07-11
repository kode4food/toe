package lsp

import (
	"slices"

	"go.lsp.dev/protocol"

	"github.com/kode4food/toe/internal/view"
)

var completionItemKindNames = map[protocol.CompletionItemKind]string{
	protocol.CompletionItemKindText:          "text",
	protocol.CompletionItemKindMethod:        "method",
	protocol.CompletionItemKindFunction:      "function",
	protocol.CompletionItemKindConstructor:   "constructor",
	protocol.CompletionItemKindField:         "field",
	protocol.CompletionItemKindVariable:      "variable",
	protocol.CompletionItemKindClass:         "class",
	protocol.CompletionItemKindInterface:     "interface",
	protocol.CompletionItemKindModule:        "module",
	protocol.CompletionItemKindProperty:      "property",
	protocol.CompletionItemKindUnit:          "unit",
	protocol.CompletionItemKindValue:         "value",
	protocol.CompletionItemKindEnum:          "enum",
	protocol.CompletionItemKindKeyword:       "keyword",
	protocol.CompletionItemKindSnippet:       "snippet",
	protocol.CompletionItemKindColor:         "color",
	protocol.CompletionItemKindFile:          "file",
	protocol.CompletionItemKindReference:     "reference",
	protocol.CompletionItemKindFolder:        "folder",
	protocol.CompletionItemKindEnumMember:    "enum_member",
	protocol.CompletionItemKindConstant:      "constant",
	protocol.CompletionItemKindStruct:        "struct",
	protocol.CompletionItemKindEvent:         "event",
	protocol.CompletionItemKindOperator:      "operator",
	protocol.CompletionItemKindTypeParameter: "type_param",
}

func normalizeCompletionResult(
	server string, result protocol.CompletionResult,
) CompletionList {
	switch r := result.(type) {
	case protocol.CompletionItemSlice:
		return CompletionList{
			Items: normalizeCompletionItems(server, r), Raw: r,
		}
	case *protocol.CompletionList:
		return CompletionList{
			Items:      normalizeCompletionItems(server, r.Items),
			Raw:        r.Items,
			Incomplete: r.IsIncomplete,
		}
	default:
		return CompletionList{}
	}
}

func normalizeCompletionItems(
	server string, items []protocol.CompletionItem,
) []view.CompletionItem {
	out := make([]view.CompletionItem, 0, len(items))
	for _, item := range items {
		out = append(out, normalizeCompletionItem(server, item))
	}
	return out
}

func normalizeCompletionItem(
	server string, item protocol.CompletionItem,
) view.CompletionItem {
	filter := item.Label
	if text, ok := item.FilterText.Get(); ok {
		filter = text
	}
	sortText := item.Label
	if text, ok := item.SortText.Get(); ok {
		sortText = text
	}
	insert := item.Label
	if text, ok := item.InsertText.Get(); ok {
		insert = text
	}
	detail, _ := item.Detail.Get()
	preselect, _ := item.Preselect.Get()
	deprecated := completionDeprecated(item.Tags)
	labelDetail, labelDescription := completionLabelDetails(item)
	return view.CompletionItem{
		Label:            item.Label,
		LabelDetail:      labelDetail,
		LabelDescription: labelDescription,
		Detail:           detail,
		Filter:           filter,
		Sort:             sortText,
		Insert:           insert,
		Kind:             completionItemKind(item.Kind),
		Docs:             completionDocumentation(detail, item.Documentation),
		Server:           server,
		Preselect:        preselect,
		Deprecated:       deprecated,
	}
}

func completionLabelDetails(item protocol.CompletionItem) (string, string) {
	if item.LabelDetails == nil {
		return "", ""
	}
	detail := ""
	if item.LabelDetails.Detail != nil {
		detail = *item.LabelDetails.Detail
	}
	description := ""
	if item.LabelDetails.Description != nil {
		description = *item.LabelDetails.Description
	}
	return detail, description
}

func completionDeprecated(tags []protocol.CompletionItemTag) bool {
	return slices.Contains(tags, protocol.CompletionItemTagDeprecated)
}

func completionItemKind(kind protocol.CompletionItemKind) string {
	return completionItemKindNames[kind]
}

func completionDocumentation(
	detail string, docs protocol.InlayHintTooltip,
) string {
	doc := markupText(docs)
	switch {
	case detail != "" && doc != "":
		return "```text\n" + detail + "\n```\n" + doc
	case detail != "":
		return "```text\n" + detail + "\n```"
	default:
		return doc
	}
}
