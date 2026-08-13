package plan

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

func Fingerprint(p *TestPlan) (string, error) {
	data, err := json.Marshal(p)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}
