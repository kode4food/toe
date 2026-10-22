package language

import (
	"os"
	"slices"
	"sync"

	"github.com/kode4food/toe/internal/loader"
)

// Registry holds the language configuration merged for the working directory,
// loading it on first use and again after Invalidate. The zero value is ready
type Registry struct {
	mu     sync.Mutex
	config Languages
	byName map[string]*Language
	files  []string
	chroma *lexerIndex
	loaded bool
}

// Get returns the definition for a language name, or an empty one when
// no user configuration directory exists or no definition has that name
func (r *Registry) Get(name string) *Language {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.ensureLoaded()
	if l, ok := r.byName[name]; ok {
		return l
	}
	return &Language{}
}

// DetectArgs is a file path and its content, both used to identify the
// language, plus the name to fall back to when nothing matches
type DetectArgs struct {
	Path    string
	Content string
	Default string
}

// Detect identifies a language from the file name, then its shebang, then its
// content, trying Chroma's lexers when no definition matches and returning
// Default when nothing does
func (r *Registry) Detect(args DetectArgs) string {
	r.mu.Lock()
	r.ensureLoaded()
	langs := r.config
	if r.chroma == nil {
		r.chroma = newLexerIndex()
	}
	index := r.chroma
	r.mu.Unlock()
	if lang, ok := definedLanguage(langs, args); ok {
		return lang
	}
	if lang, ok := chromaLanguage(index, args); ok {
		return lang
	}
	return args.Default
}

// Invalidate drops the loaded configuration so the next use reloads it
func (r *Registry) Invalidate() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.reset()
}

// FileChanged drops the loaded configuration when path is a file it came from
func (r *Registry) FileChanged(path string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if slices.Contains(r.files, loader.CanonicalPath(path)) {
		r.reset()
	}
}

func (r *Registry) ensureLoaded() {
	if r.loaded {
		return
	}
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}
	global, named := loader.LanguagesFile()
	files := loader.WorkspaceFiles{
		Global:    global,
		Workspace: loader.WorkspaceLanguagesFile(cwd),
		Dir:       cwd,
	}
	r.config, _ = LoadLanguagesForWorkspace(files)
	r.byName = map[string]*Language{}
	if named {
		for i := range r.config.Languages {
			l := &r.config.Languages[i]
			if _, ok := r.byName[l.Name]; !ok {
				r.byName[l.Name] = l
			}
		}
	}
	r.files = []string{loader.CanonicalPath(files.Workspace)}
	if named {
		r.files = append(r.files, loader.CanonicalPath(global))
	}
	if trust, ok := loader.WorkspaceTrustFile(); ok {
		r.files = append(r.files, loader.CanonicalPath(trust))
	}
	r.loaded = true
}

func (r *Registry) reset() {
	r.config = Languages{}
	r.byName = nil
	r.files = nil
	r.loaded = false
}
