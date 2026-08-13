package capability

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

func DecodeInvocation(data []byte) (Invocation, error) {
	var invocation Invocation
	if err := decodeStrict(data, &invocation); err != nil {
		return Invocation{}, err
	}
	if err := ValidateInvocation(invocation); err != nil {
		return Invocation{}, err
	}
	return invocation, nil
}

func DecodeResult(data []byte) (Result, error) {
	var result Result
	if err := decodeStrict(data, &result); err != nil {
		return Result{}, err
	}
	if err := ValidateResult(result); err != nil {
		return Result{}, err
	}
	return result, nil
}

func decodeStrict(data []byte, target any) error {
	if len(data) == 0 {
		return errors.New("empty protocol envelope")
	}
	if len(data) > MaxEnvelopeBytes {
		return fmt.Errorf("protocol envelope exceeds %d bytes", MaxEnvelopeBytes)
	}
	if err := rejectDuplicateKeys(data); err != nil {
		return err
	}

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("decode protocol envelope: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("multiple JSON values are not allowed")
		}
		return fmt.Errorf("decode trailing protocol data: %w", err)
	}
	return nil
}

func rejectDuplicateKeys(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := scanJSONValue(decoder); err != nil {
		return err
	}
	if token, err := decoder.Token(); !errors.Is(err, io.EOF) {
		if err == nil {
			return fmt.Errorf("unexpected trailing JSON token %v", token)
		}
		return fmt.Errorf("decode trailing JSON token: %w", err)
	}
	return nil
}

func scanJSONValue(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return fmt.Errorf("decode JSON token: %w", err)
	}

	delim, ok := token.(json.Delim)
	if !ok {
		return nil
	}

	switch delim {
	case '{':
		seen := map[string]struct{}{}
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return fmt.Errorf("decode JSON object key: %w", err)
			}
			key, ok := keyToken.(string)
			if !ok {
				return errors.New("JSON object key is not a string")
			}
			if _, exists := seen[key]; exists {
				return fmt.Errorf("duplicate JSON field %q", key)
			}
			seen[key] = struct{}{}
			if err := scanJSONValue(decoder); err != nil {
				return err
			}
		}
		closing, err := decoder.Token()
		if err != nil {
			return fmt.Errorf("decode JSON object close: %w", err)
		}
		if closing != json.Delim('}') {
			return errors.New("invalid JSON object closing delimiter")
		}
	case '[':
		for decoder.More() {
			if err := scanJSONValue(decoder); err != nil {
				return err
			}
		}
		closing, err := decoder.Token()
		if err != nil {
			return fmt.Errorf("decode JSON array close: %w", err)
		}
		if closing != json.Delim(']') {
			return errors.New("invalid JSON array closing delimiter")
		}
	default:
		return fmt.Errorf("unexpected JSON delimiter %q", delim)
	}
	return nil
}
