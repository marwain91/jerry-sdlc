package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"strings"
)

type collisionInput struct {
	SchemaVersion               int      `json:"schemaVersion"`
	ExplicitSelection           string   `json:"explicitSelection"`
	ActiveJerryRunID            string   `json:"activeJerryRunId,omitempty"`
	RequestedJerryRunID         string   `json:"requestedJerryRunId,omitempty"`
	CompetingBroadOrchestrators []string `json:"competingBroadOrchestrators"`
	DiscoveryEvidence           string   `json:"discoveryEvidence"`
}

func resolveCollision(args []string) (result, error) {
	fs := flag.NewFlagSet("resolve-collision", flag.ContinueOnError)
	path := fs.String("file", "", "collision evidence JSON")
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	if *path == "" || fs.NArg() != 0 {
		return nil, errors.New("--file is required and positional arguments are forbidden")
	}
	b, err := readBoundedRegularFile(*path, 64*1024)
	if err != nil {
		return nil, err
	}
	dec := json.NewDecoder(strings.NewReader(string(b)))
	dec.DisallowUnknownFields()
	var input collisionInput
	if err := dec.Decode(&input); err != nil {
		return nil, err
	}
	var trailing any
	if err := dec.Decode(&trailing); err != io.EOF {
		return nil, errors.New("collision input contains trailing JSON")
	}
	if err := validateCollisionInput(input); err != nil {
		return nil, err
	}

	decision, assurance, reason := "PROCEED_JERRY", "UNCHANGED", "no competing broad orchestrator was reported"
	launchAllowed := true
	if input.ExplicitSelection == "JERRY" {
		reason = "the user explicitly selected Jerry SDLC"
	} else if input.ExplicitSelection == "OTHER" {
		decision, assurance, reason, launchAllowed = "YIELD_TO_EXPLICIT_SELECTION", "UNCHANGED", "the user explicitly selected another orchestrator", false
	} else if input.ActiveJerryRunID != "" && input.ActiveJerryRunID == input.RequestedJerryRunID {
		decision, reason = "RESUME_JERRY", "the requested Jerry run is already active"
	} else if input.ActiveJerryRunID != "" {
		decision, assurance, reason, launchAllowed = "COLLISION", "OWNER_GATE", "a different Jerry run is active", false
	} else if len(input.CompetingBroadOrchestrators) > 0 {
		decision, assurance, reason, launchAllowed = "COLLISION", "OWNER_GATE", "one or more competing broad orchestrators were reported", false
	}

	digest := sha256.Sum256(b)
	return result{
		"schemaVersion":        1,
		"decision":             decision,
		"assurance":            assurance,
		"launchAllowed":        launchAllowed,
		"launchPerformed":      false,
		"reason":               reason,
		"discoveryEvidence":    input.DiscoveryEvidence,
		"inputDigest":          hex.EncodeToString(digest[:]),
		"runtimeDiscoveryRule": "CALLER_SUPPLIED_EVIDENCE_ONLY",
	}, nil
}

func validateCollisionInput(input collisionInput) error {
	if input.SchemaVersion != 1 || !contains([]string{"NONE", "JERRY", "OTHER"}, input.ExplicitSelection) || input.CompetingBroadOrchestrators == nil || !contains([]string{"RUNTIME_OBSERVED", "CALLER_DECLARED"}, input.DiscoveryEvidence) {
		return errors.New("collision input envelope is invalid")
	}
	if (input.ActiveJerryRunID == "") != (input.RequestedJerryRunID == "") {
		return errors.New("active and requested Jerry run IDs must be supplied together")
	}
	if len(input.ActiveJerryRunID) > 256 || len(input.RequestedJerryRunID) > 256 || strings.TrimSpace(input.ActiveJerryRunID) != input.ActiveJerryRunID || strings.TrimSpace(input.RequestedJerryRunID) != input.RequestedJerryRunID {
		return errors.New("Jerry run ID is invalid")
	}
	seen := map[string]bool{}
	for _, name := range input.CompetingBroadOrchestrators {
		if len(name) > 128 || strings.TrimSpace(name) != name {
			return errors.New("competing orchestrator name is invalid")
		}
		normalized := normalizeFixtureText(name)
		if normalized == "" || normalized == "jerry sdlc" || normalized == "jsdlc" || seen[normalized] {
			return errors.New("competing orchestrator name is blank, reserved, or duplicated")
		}
		seen[normalized] = true
	}
	return nil
}
