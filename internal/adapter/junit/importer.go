package junit

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"strings"

	adaptercontract "github.com/hackelia-micrantha/testule/internal/adapter"
	"github.com/hackelia-micrantha/testule/internal/evidence"
)

const (
	AdapterID     = "junit-xml/v1alpha1"
	MaxInputBytes = 1 << 20
)

type Adapter struct{}

func (Adapter) Describe() adaptercontract.Descriptor {
	return adaptercontract.Descriptor{
		ProtocolVersion: adaptercontract.ProtocolVersion,
		ID:              AdapterID,
		Version:         "v1alpha1",
		Capabilities:    []adaptercontract.Capability{adaptercontract.CapabilityEvidenceImport},
	}
}

func (Adapter) Probe(context.Context) adaptercontract.ProbeResult {
	return adaptercontract.ProbeResult{
		Adapter: AdapterID,
		Availability: []adaptercontract.Availability{{
			Capability: adaptercontract.CapabilityEvidenceImport,
			Available:  true,
		}},
	}
}

func (Adapter) Invoke(_ context.Context, invocation adaptercontract.Invocation) adaptercontract.Result {
	if invocation.Capability != adaptercontract.CapabilityEvidenceImport {
		return adaptercontract.Result{Status: adaptercontract.StatusUnsupported, Diagnostics: []string{"unsupported capability"}}
	}
	if invocation.TargetID != "" {
		return adaptercontract.Result{Status: adaptercontract.StatusInvalidRequest, Diagnostics: []string{"JUnit import does not accept a target ID"}}
	}
	if len(invocation.AdapterOptions) != 0 {
		return adaptercontract.Result{Status: adaptercontract.StatusInvalidRequest, Diagnostics: []string{"JUnit import does not accept adapter options"}}
	}
	if len(invocation.Input) == 0 || len(invocation.Input) > MaxInputBytes {
		return adaptercontract.Result{Status: adaptercontract.StatusInvalidRequest, Diagnostics: []string{"JUnit XML input must be between 1 byte and 1 MiB"}}
	}
	upper := bytes.ToUpper(invocation.Input)
	if bytes.Contains(upper, []byte("<!DOCTYPE")) || bytes.Contains(upper, []byte("<!ENTITY")) {
		return adaptercontract.Result{Status: adaptercontract.StatusInvalidRequest, Diagnostics: []string{"DTD/entity declarations are not accepted"}}
	}
	cases, err := decodeCases(invocation.Input)
	if err != nil {
		return adaptercontract.Result{Status: adaptercontract.StatusInvalidRequest, Diagnostics: []string{err.Error()}}
	}
	if len(cases) == 0 || len(cases) > 256 {
		return adaptercontract.Result{Status: adaptercontract.StatusInvalidRequest, Diagnostics: []string{"JUnit XML must contain between 1 and 256 test cases"}}
	}

	observations := make([]evidence.Observation, 0, len(cases))
	for i, testCase := range cases {
		status := "passed"
		if testCase.Failure != nil || testCase.Error != nil {
			status = "failed"
		} else if testCase.Skipped != nil {
			status = "skipped"
		}
		observations = append(observations, evidence.Observation{
			ID:       fmt.Sprintf("junit:%03d:%s", i+1, boundedLabel(testCase.Classname, testCase.Name)),
			Status:   status,
			Coverage: coverage(invocation.Coverage),
		})
	}

	record := &evidence.Evidence{
		APIVersion:  invocation.Plan.APIVersion,
		Kind:        evidence.Kind,
		Metadata:    &evidence.Metadata{Name: "junit-import-" + invocation.RunID},
		Plan:        &evidence.PlanReference{Name: invocation.Plan.Name, Fingerprint: invocation.Plan.Fingerprint},
		Subject:     &evidence.Subject{Component: invocation.Plan.Component, Revision: invocation.SubjectRevision},
		Environment: &evidence.Environment{ID: invocation.EnvironmentID},
		Provenance:  &evidence.Provenance{Producer: AdapterID, RunID: invocation.RunID},
		Observations: observations,
	}
	if diagnostics := evidence.Validate(record); len(diagnostics) != 0 {
		return adaptercontract.Result{Status: adaptercontract.StatusInfrastructureFailed, Diagnostics: []string{fmt.Sprintf("generated invalid evidence: %v", diagnostics)}}
	}
	return adaptercontract.Result{Status: adaptercontract.StatusCompleted, Evidence: record}
}

type testSuites struct {
	Suites []testSuite `xml:"testsuite"`
}

type testSuite struct {
	Cases []testCase `xml:"testcase"`
}

type testCase struct {
	Name      string  `xml:"name,attr"`
	Classname string  `xml:"classname,attr"`
	Failure   *marker `xml:"failure"`
	Error     *marker `xml:"error"`
	Skipped   *marker `xml:"skipped"`
}

type marker struct{}

func decodeCases(data []byte) ([]testCase, error) {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	decoder.Strict = true
	var root struct {
		XMLName xml.Name
	}
	if err := decoder.Decode(&root); err != nil {
		return nil, fmt.Errorf("decode JUnit XML: %w", err)
	}

	switch root.XMLName.Local {
	case "testsuite":
		var suite testSuite
		if err := xml.Unmarshal(data, &suite); err != nil {
			return nil, fmt.Errorf("decode JUnit testsuite: %w", err)
		}
		return suite.Cases, nil
	case "testsuites":
		var suites testSuites
		if err := xml.Unmarshal(data, &suites); err != nil {
			return nil, fmt.Errorf("decode JUnit testsuites: %w", err)
		}
		var cases []testCase
		for _, suite := range suites.Suites {
			cases = append(cases, suite.Cases...)
		}
		return cases, nil
	default:
		return nil, fmt.Errorf("unsupported JUnit root element %q", root.XMLName.Local)
	}
}

func coverage(value adaptercontract.Coverage) evidence.Coverage {
	result := evidence.Coverage{}
	if value.Level != "" {
		result.Levels = []string{value.Level}
	}
	if value.Behavior != "" {
		result.Behaviors = []string{value.Behavior}
	}
	if value.Generation != "" {
		result.Generation = []string{value.Generation}
	}
	if value.Visibility != "" {
		result.Visibility = []string{value.Visibility}
	}
	if value.QualityAttribute != "" {
		result.QualityAttributes = []string{value.QualityAttribute}
	}
	return result
}

func boundedLabel(classname, name string) string {
	label := strings.Trim(strings.TrimSpace(classname)+"."+strings.TrimSpace(name), ".")
	if label == "" {
		return "case"
	}
	runes := []rune(label)
	if len(runes) > 108 {
		runes = runes[:108]
	}
	return string(runes)
}
