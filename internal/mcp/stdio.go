package mcp

import (
	"bufio"
	"context"
	"errors"
	"io"
)

type framingState struct {
	lastInput string
}

func (s *Server) ServeStdio(ctx context.Context, reader io.Reader, writer io.Writer) error {
	bufferedReader := bufio.NewReader(reader)
	state := &framingState{}

	for {
		message, err := readMessage(bufferedReader, state)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}

		result, err := s.HandleMessage(ctx, message)
		if err != nil {
			return err
		}
		for _, response := range result.Messages {
			if err := writeMessage(writer, state, s.cfg.Framing, response); err != nil {
				return err
			}
		}
		if result.Exit {
			return nil
		}
	}
}
