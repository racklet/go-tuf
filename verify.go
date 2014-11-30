package tuf

import (
	"encoding/json"
	"errors"

	"github.com/agl/ed25519"
	"github.com/flynn/tuf/data"
	"github.com/flynn/tuf/keys"
	"github.com/tent/canonical-json-go"
)

var (
	ErrMissingKey    = errors.New("tuf: missing key")
	ErrNoSignatures  = errors.New("tuf: data has no signatures")
	ErrInvalid       = errors.New("tuf: signature verificate failed")
	ErrWrongMethod   = errors.New("tuf: invalid signature type")
	ErrUnknownRole   = errors.New("tuf: unknown role")
	ErrRoleThreshold = errors.New("tuf: valid signatures did not meet threshold")
)

func VerifySigned(db *keys.DB, s *data.Signed, role string) error {
	if len(s.Signatures) == 0 {
		return ErrNoSignatures
	}

	roleData := db.GetRole(role)
	if roleData == nil {
		return ErrUnknownRole
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(s.Signed, &decoded); err != nil {
		return err
	}
	msg, err := cjson.Marshal(decoded)
	if err != nil {
		return err
	}

	valid := make(map[string]struct{})
	var sigBytes [ed25519.SignatureSize]byte
	for _, sig := range s.Signatures {
		if sig.Method != "ed25519" {
			return ErrWrongMethod
		}
		if len(sig.Signature) != len(sigBytes) {
			return ErrInvalid
		}

		if !roleData.ValidKey(sig.KeyID) {
			continue
		}
		key := db.GetKey(sig.KeyID)
		if key == nil {
			continue
		}

		copy(sigBytes[:], sig.Signature)
		if !ed25519.Verify(&key.Public, msg, &sigBytes) {
			return ErrInvalid
		}
		valid[sig.KeyID] = struct{}{}
	}
	if len(valid) < roleData.Threshold {
		return ErrRoleThreshold
	}
	return nil
}
