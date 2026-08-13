package evidence

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/hackelia-micrantha/testule/internal/plan"
	"go.yaml.in/yaml/v3"
)

const MaxEvidenceBytes int64 = 1 << 20

func Decode(data []byte) (*Evidence, []plan.Diagnostic) {
	if int64(len(data)) > MaxEvidenceBytes {
		return nil, []plan.Diagnostic{{
			Code:    "input_too_large",
			Message: fmt.Sprintf("evidence exceeds maximum size of %d bytes", MaxEvidenceBytes),
		}}
	}

	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)

	var e Evidence
	if err := decoder.Decode(&e); err != nil {
		return nil, []plan.Diagnostic{decodeDiagnostic(err)}
	}

	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return nil, []plan.Diagnostic{{Code: "multiple_documents", Message: "exactly one YAML document is supported"}}
		}
		return nil, []plan.Diagnostic{decodeDiagnostic(err)}
	}

	return &e, Validate(&e)
}

func decodeDiagnostic(err error) plan.Diagnostic {
	message := err.Error()
	code := "invalid_yaml"
	if strings.Contains(message, "field ") && strings.Contains(message, " not found in type ") {
		code = "unknown_field"
	}
	return plan.Diagnostic{Code: code, Message: message}
}
