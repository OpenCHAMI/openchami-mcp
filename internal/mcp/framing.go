package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/OpenCHAMI/openchami-mcp/internal/config"
)

func readMessage(reader *bufio.Reader, state *framingState) (map[string]any, error) {
	line, err := readNonEmptyLine(reader)
	if err != nil {
		return nil, err
	}

	if looksLikeJSON(line) {
		state.lastInput = config.FramingJSONL
		return parseJSONMessage(line)
	}

	state.lastInput = config.FramingContentLength
	length, err := parseContentLength(line, reader)
	if err != nil {
		return nil, err
	}

	body := make([]byte, length)
	if _, err := io.ReadFull(reader, body); err != nil {
		return nil, err
	}
	return parseJSONMessage(body)
}

func writeMessage(writer io.Writer, state *framingState, configuredFraming string, message map[string]any) error {
	raw, err := json.Marshal(message)
	if err != nil {
		return err
	}

	mode := configuredFraming
	if mode == config.FramingAuto {
		mode = state.lastInput
		if mode == "" {
			mode = config.FramingContentLength
		}
	}

	switch mode {
	case config.FramingJSONL:
		_, err = fmt.Fprintf(writer, "%s\n", raw)
	default:
		_, err = fmt.Fprintf(writer, "Content-Length: %d\r\n\r\n%s", len(raw), raw)
	}
	return err
}

func readNonEmptyLine(reader *bufio.Reader) ([]byte, error) {
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF && len(line) > 0 {
				return line, nil
			}
			return nil, err
		}
		trimmed := strings.TrimSpace(string(line))
		if trimmed != "" {
			return []byte(trimmed), nil
		}
	}
}

func looksLikeJSON(line []byte) bool {
	trimmed := strings.TrimSpace(string(line))
	return strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[")
}

func parseJSONMessage(raw []byte) (map[string]any, error) {
	var message map[string]any
	if err := json.Unmarshal(raw, &message); err != nil {
		return nil, err
	}
	return message, nil
}

func parseContentLength(firstLine []byte, reader *bufio.Reader) (int, error) {
	lines := []string{string(firstLine)}
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return 0, err
		}
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			break
		}
		lines = append(lines, trimmed)
	}

	for _, line := range lines {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(parts[0]), "Content-Length") {
			length, err := strconv.Atoi(strings.TrimSpace(parts[1]))
			if err != nil {
				return 0, err
			}
			return length, nil
		}
	}
	return 0, fmt.Errorf("missing Content-Length header")
}
