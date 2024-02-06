package dsl

import (
	"github.com/goccy/go-json"
	"github.com/google/uuid"
)

type SourceMeta struct {
	Source string `json:"source"`
	Event  string `json:"event"`
	Action string `json:"action"`
}

// Deprecated: Use github.com/hiphops-io/hops/nats CreateSourceEvent
func CreateSourceEvent(rawEvent map[string]any, source string, event string, action string) ([]byte, string, error) {
	rawEvent["hops"] = SourceMeta{
		Source: source,
		Event:  event,
		Action: action,
	}

	sourceBytes, err := json.Marshal(rawEvent)
	if err != nil {
		return nil, "", err
	}

	// We don't really care about the UUID namespace, so we just use an existing one
	sourceUUID := uuid.NewSHA1(uuid.NameSpaceDNS, sourceBytes)
	hash := sourceUUID.String()

	return sourceBytes, hash, nil
}
