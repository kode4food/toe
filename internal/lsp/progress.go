package lsp

import (
	"fmt"
	"strings"
	"sync"

	"github.com/go-json-experiment/json"
	"go.lsp.dev/protocol"
)

type (
	// progressState tracks in-flight work-done progress notifications, keyed
	// by server name and then by progress token
	progressState struct {
		sync.RWMutex
		byServer map[string]map[string]progressEntry
	}

	progressEntry struct {
		title      string
		message    string
		percentage *uint32
		started    bool
	}

	progressKind struct {
		Kind string `json:"kind"`
	}
)

// Busy reports whether any language server has an in-flight progress
// notification, for the status-line activity spinner
func (s *Session) Busy() bool {
	s.progress.RLock()
	defer s.progress.RUnlock()
	return len(s.progress.byServer) > 0
}

func (s *Session) createProgress(server string, token protocol.ProgressToken) {
	s.progress.Lock()
	defer s.progress.Unlock()
	s.progress.forServer(server)[progressTokenKey(token)] = progressEntry{}
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
	s.progress.Lock()
	defer s.progress.Unlock()
	s.progress.forServer(server)[progressTokenKey(token)] = entry
}

func (s *Session) lookupProgress(
	server string, token protocol.ProgressToken,
) (progressEntry, bool) {
	s.progress.RLock()
	defer s.progress.RUnlock()
	entry, ok := s.progress.byServer[server][progressTokenKey(token)]
	return entry, ok
}

func (s *Session) clearProgress(server string, token protocol.ProgressToken) {
	s.progress.Lock()
	defer s.progress.Unlock()
	delete(s.progress.byServer[server], progressTokenKey(token))
	if len(s.progress.byServer[server]) == 0 {
		delete(s.progress.byServer, server)
	}
}

func (p *progressState) forServer(server string) map[string]progressEntry {
	entries := p.byServer[server]
	if entries == nil {
		entries = map[string]progressEntry{}
		p.byServer[server] = entries
	}
	return entries
}

func (p *progressState) reset() {
	p.Lock()
	defer p.Unlock()
	p.byServer = map[string]map[string]progressEntry{}
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
