package main

import (
	"fmt"
	"os"

	app "github.com/kode4food/toe/cmd/toe/internal"
)

func main() {
	if err := app.Run(os.Args[1:], os.Stdout); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
