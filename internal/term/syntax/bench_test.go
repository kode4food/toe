package syntax_test

import (
	"strings"
	"testing"

	"github.com/kode4food/toe/internal/term/syntax"
)

var benchSrcs = map[string]string{
	"go": `package main

import (
	"fmt"
	"strings"
)

// Greet returns a greeting string
func Greet(name string) string {
	if name == "" {
		name = "world"
	}
	return fmt.Sprintf("Hello, %s!", strings.TrimSpace(name))
}

type Config struct {
	Host string
	Port int
}

const DefaultPort = 8080
var _ = DefaultPort
`,
	"javascript": `import { useState, useEffect } from 'react';

const fetchData = async (url) => {
	const res = await fetch(url);
	if (!res.ok) throw new Error(res.statusText);
	return res.json();
};

export function useData(url) {
	const [data, setData] = useState(null);
	useEffect(() => {
		fetchData(url).then(setData).catch(console.error);
	}, [url]);
	return data;
}
`,
	"yaml": strings.Repeat(`services:
  web:
    image: nginx:latest
    ports:
      - "80:80"
    environment:
      - NODE_ENV=production
`, 5),
}

func BenchmarkTokenize(b *testing.B) {
	for lang, src := range benchSrcs {
		b.Run(lang, func(b *testing.B) {
			sc := syntax.NewSyntaxCache()
			sc.Tokenize(src, lang) // warm up cache
			b.ResetTimer()
			for range b.N {
				sc.Tokenize(src, lang)
			}
		})
	}
}
