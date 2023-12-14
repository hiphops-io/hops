package dsl

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/goccy/go-json"
)

type SourceMeta struct {
	Source string `json:"source"`
	Event  string `json:"event"`
	Action string `json:"action"`
}

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

	hasher := sha256.New()
	hasher.Write(sourceBytes)
	sha256Hash := hex.EncodeToString(hasher.Sum(nil))

	return sourceBytes, sha256Hash, nil
}
