package sse

import (
	"bufio"
	"io"
	"strings"
)

type Event struct {
	Event string
	Data  string
	ID    string
}

func ReadEvents(r io.Reader, onEvent func(Event) error) error {
	scanner := bufio.NewScanner(r)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	curr := Event{}
	flush := func() error {
		if curr.Data == "" && curr.Event == "" && curr.ID == "" {
			return nil
		}
		curr.Data = strings.TrimSuffix(curr.Data, "\n")
		err := onEvent(curr)
		curr = Event{}
		return err
	}

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			if err := flush(); err != nil {
				return err
			}
			continue
		}
		if strings.HasPrefix(line, ":") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		field := parts[0]
		value := ""
		if len(parts) > 1 {
			value = strings.TrimPrefix(parts[1], " ")
		}
		switch field {
		case "event":
			curr.Event = value
		case "data":
			curr.Data += value + "\n"
		case "id":
			curr.ID = value
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return flush()
}
