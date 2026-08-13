package plan

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"

	"go.yaml.in/yaml/v3"
)

const MaxPlanBytes int64 = 1 << 20

func Decode(data []byte) (*TestPlan, []Diagnostic) {
	if int64(len(data)) > MaxPlanBytes {
		return nil, []Diagnostic{{
			Code:    "input_too_large",
			Message: fmt.Sprintf("plan exceeds maximum size of %d bytes", MaxPlanBytes),
		}}
	}

	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)

	var p TestPlan
	if err := decoder.Decode(&p); err != nil {
		return nil, []Diagnostic{decodeDiagnostic(err)}
	}

	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return nil, []Diagnostic{{Code: "multiple_documents", Message: "exactly one YAML document is supported"}}
		}
		return nil, []Diagnostic{decodeDiagnostic(err)}
	}

	return &p, Validate(&p)
}

func decodeDiagnostic(err error) Diagnostic {
	message := err.Error()
	code := "invalid_yaml"
	if strings.Contains(message, "field ") && strings.Contains(message, " not found in type ") {
		code = "unknown_field"
	}
	return Diagnostic{Code: code, Message: message}
}
