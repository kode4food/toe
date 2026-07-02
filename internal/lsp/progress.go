package lsp

import (
	"fmt"
	"strings"

	"github.com/go-json-experiment/json"
	"go.lsp.dev/protocol"
)

type progressEntry struct {
	title      string
	message    string
	percentage *uint32
	started    bool
}

type progressKind struct {
	Kind string `json:"kind"`
}

func (s *Session) createProgress(
	server string, token protocol.ProgressToken,
) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.progressForServer(server)[progressTokenKey(token)] = progressEntry{}
}

func (s *Session) updateProgress(
	server string, params *protocol.ProgressParams,
) {
	if params == nil {
		return
	}
	var kind progressKind
	if err := json.Unmarshal(params.Value, &kind); err != nil {
		return
	}
	switch kind.Kind {
	case "begin":
		s.beginProgress(server, params.Token, params.Value)
	case "report":
		s.reportProgress(server, params.Token, params.Value)
	case "end":
		s.endProgress(server, params.Token, params.Value)
	}
}

func (s *Session) beginProgress(
	server string, token protocol.ProgressToken, value protocol.LSPAny,
) {
	var begin protocol.WorkDoneProgressBegin
	if err := json.Unmarshal(value, &begin); err != nil {
		return
	}
	entry := progressEntry{
		title:      begin.Title,
		percentage: begin.Percentage,
		started:    true,
	}
	if begin.Message != nil {
		entry.message = *begin.Message
	}
	s.storeProgress(server, token, entry)
	s.showProgress(server, entry)
}

func (s *Session) reportProgress(
	server string, token protocol.ProgressToken, value protocol.LSPAny,
) {
	var report protocol.WorkDoneProgressReport
	if err := json.Unmarshal(value, &report); err != nil {
		return
	}
	entry, ok := s.lookupProgress(server, token)
	if !ok {
		return
	}
	entry.started = true
	if report.Message != nil {
		entry.message = *report.Message
	}
	if report.Percentage != nil {
		entry.percentage = report.Percentage
	}
	s.storeProgress(server, token, entry)
	s.showProgress(server, entry)
}

func (s *Session) endProgress(
	server string, token protocol.ProgressToken, value protocol.LSPAny,
) {
	var end protocol.WorkDoneProgressEnd
	if err := json.Unmarshal(value, &end); err != nil {
		return
	}
	entry, ok := s.lookupProgress(server, token)
	s.clearProgress(server, token)
	if end.Message == nil {
		return
	}
	if !ok {
		entry = progressEntry{}
	}
	entry.message = *end.Message
	s.showProgress(server, entry)
}

func (s *Session) storeProgress(
	server string, token protocol.ProgressToken, entry progressEntry,
) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.progressForServer(server)[progressTokenKey(token)] = entry
}

func (s *Session) lookupProgress(
	server string, token protocol.ProgressToken,
) (progressEntry, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.progress[server][progressTokenKey(token)]
	return entry, ok
}

func (s *Session) clearProgress(
	server string, token protocol.ProgressToken,
) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.progress[server], progressTokenKey(token))
	if len(s.progress[server]) == 0 {
		delete(s.progress, server)
	}
}

func (s *Session) progressForServer(
	server string,
) map[string]progressEntry {
	entries := s.progress[server]
	if entries == nil {
		entries = map[string]progressEntry{}
		s.progress[server] = entries
	}
	return entries
}

func (s *Session) showProgress(server string, entry progressEntry) {
	if s.editor == nil {
		return
	}
	msg := progressMessage(server, entry)
	if msg != "" {
		s.editor.SetStatusMsg(msg)
	}
}

func progressMessage(server string, entry progressEntry) string {
	var b strings.Builder
	b.WriteString(server)
	b.WriteString(": ")
	if entry.percentage != nil {
		_, _ = fmt.Fprintf(&b, "%2d%% ", *entry.percentage)
	}
	b.WriteString(entry.title)
	if entry.title != "" && entry.message != "" {
		b.WriteString(" · ")
	}
	b.WriteString(entry.message)
	msg := b.String()
	if msg == server+": " {
		return ""
	}
	return msg
}

func progressTokenKey(token protocol.ProgressToken) string {
	switch v := token.(type) {
	case protocol.String:
		return "s:" + string(v)
	case protocol.Integer:
		return fmt.Sprintf("i:%d", v)
	default:
		return fmt.Sprintf("%T:%v", token, token)
	}
}
