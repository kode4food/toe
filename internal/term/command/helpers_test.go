package command_test

import "github.com/kode4food/toe/internal/term/command"

func commandTokens(input string, validate bool) ([]string, error) {
	tok := command.NewTokenizer(input, validate)
	var args []string
	for {
		token, ok, err := tok.Next()
		if err != nil {
			return nil, err
		}
		if !ok {
			return args, nil
		}
		args = append(args, token.Content)
	}
}
