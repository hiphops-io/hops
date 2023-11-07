package nats

import (
	"encoding/base64"
	"fmt"
	"os"

	"github.com/goccy/go-json"
)

type KeyFile struct {
	AccountId  string `json:"account_id"`
	Password   string `json:"password"`
	NatsDomain string `json:"nats_domain"`
}

func NewKeyFile(keyfilePath string) (KeyFile, error) {
	keyfile := KeyFile{}

	body, err := os.ReadFile(keyfilePath)
	if err != nil {
		return keyfile, err
	}

	jsonKey, err := base64.StdEncoding.DecodeString(string(body))
	if err != nil {
		return keyfile, err
	}

	err = json.Unmarshal(jsonKey, &keyfile)
	if err != nil {
		return keyfile, err
	}

	return keyfile, nil
}

func (k *KeyFile) NatsUrl() string {
	return fmt.Sprintf("nats://%s:%s@%s:4222", k.AccountId, k.Password, k.NatsDomain)
}
