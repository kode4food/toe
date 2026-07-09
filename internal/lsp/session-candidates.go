package lsp

import (
	"sync"

	"github.com/kode4food/toe/internal/view"
)

// candidateState holds id-keyed results from completion, code-action, and
// document-link requests, kept around for a matching resolve/apply call
type candidateState struct {
	sync.RWMutex
	completions map[string]completionCandidate
	codeActions map[string]codeActionCandidate
	links       map[string]documentLinkCandidate
}

func (c *candidateState) clearLinksForDoc(docID view.DocumentId) {
	c.Lock()
	defer c.Unlock()
	c.clearLinksForDocLocked(docID)
}

func (c *candidateState) clearLinksForDocLocked(docID view.DocumentId) {
	for id, link := range c.links {
		if link.docID == docID {
			delete(c.links, id)
		}
	}
}

func (c *candidateState) reset() {
	c.Lock()
	defer c.Unlock()
	c.completions = map[string]completionCandidate{}
	c.codeActions = map[string]codeActionCandidate{}
	c.links = map[string]documentLinkCandidate{}
}
