package evidence

import "testing"

func FuzzDecodeNeverPanics(f *testing.F) {
	f.Add([]byte(`apiVersion: testule.dev/v1alpha1
kind: Evidence
metadata:
  name: seed
plan:
  name: parser
  fingerprint: sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
subject:
  component: parser
  revision: rev-1
environment:
  id: linux
provenance:
  producer: seed
  runId: run-1
observations:
  - id: unit
    status: passed
    coverage:
      levels: [unit]
`))
	f.Fuzz(func(t *testing.T, data []byte) {
		if int64(len(data)) > MaxEvidenceBytes {
			data = data[:MaxEvidenceBytes]
		}
		_, _ = Decode(data)
	})
}
