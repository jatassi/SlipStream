package works

import (
	"encoding/json"
	"fmt"
)

// EncodeCandidates is how a search's answer crosses the Metadata Role's
// wire; DecodeCandidates reads it back.
func EncodeCandidates(cs []Candidate) ([]byte, error) {
	b, err := json.Marshal(cs)
	if err != nil {
		return nil, fmt.Errorf("encode candidates: %w", err)
	}
	return b, nil
}

func DecodeCandidates(b []byte) ([]Candidate, error) {
	var cs []Candidate
	if err := json.Unmarshal(b, &cs); err != nil {
		return nil, fmt.Errorf("decode candidates: %w", err)
	}
	return cs, nil
}

// EncodeFetched is how a fetched record crosses the Metadata Role's wire;
// DecodeFetched reads it back. Detail travels as base64.
func EncodeFetched(f Fetched) ([]byte, error) {
	b, err := json.Marshal(f)
	if err != nil {
		return nil, fmt.Errorf("encode a fetched record: %w", err)
	}
	return b, nil
}

func DecodeFetched(b []byte) (Fetched, error) {
	var f Fetched
	if err := json.Unmarshal(b, &f); err != nil {
		return Fetched{}, fmt.Errorf("decode a fetched record: %w", err)
	}
	return f, nil
}
