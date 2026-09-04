package main

import (
	"encoding/json"
	"errors"
	"flag"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type adapterCatalog struct {
	SchemaVersion int                 `json:"schemaVersion"`
	Adapters      []adapterDescriptor `json:"adapters"`
}
type adapterDescriptor struct {
	Name         string              `json:"name"`
	Status       string              `json:"status"`
	Capabilities adapterCapabilities `json:"capabilities"`
	PolicySource string              `json:"policySource"`
	Reason       string              `json:"reason"`
}
type adapterCapabilities struct {
	EphemeralWorkersObserved bool `json:"ephemeralWorkersObserved"`
	WriteBlockingObserved    bool `json:"writeBlockingObserved"`
	AttestedWorkerIdentity   bool `json:"attestedWorkerIdentity"`
	WriteIsolationAttested   bool `json:"writeIsolationAttested"`
	SecretIsolationAttested  bool `json:"secretIsolationAttested"`
	NetworkIsolationAttested bool `json:"networkIsolationAttested"`
	StructuredOutputObserved bool `json:"structuredOutputObserved"`
}

type packCatalog struct {
	SchemaVersion  int              `json:"schemaVersion"`
	Status         string           `json:"status"`
	Packs          []packDescriptor `json:"packs"`
	ActivationRule string           `json:"activationRule"`
}
type packDescriptor struct {
	Name        string   `json:"name"`
	Domains     []string `json:"domains"`
	ExtraLenses []string `json:"extraLenses,omitempty"`
}

func adapters(args []string) (result, error) {
	fs := flag.NewFlagSet("adapters", flag.ContinueOnError)
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	if fs.NArg() != 0 {
		return nil, errors.New("adapters accepts no positional arguments")
	}
	root := os.Getenv("JSDLC_PLUGIN_ROOT")
	b, err := readBoundedRegularFile(filepath.Join(root, "adapters", "catalog.json"), 256*1024)
	if err != nil {
		return nil, err
	}
	dec := json.NewDecoder(strings.NewReader(string(b)))
	dec.DisallowUnknownFields()
	var catalog adapterCatalog
	if err := dec.Decode(&catalog); err != nil {
		return nil, err
	}
	var trailing any
	if err := dec.Decode(&trailing); err != io.EOF {
		return nil, errors.New("adapter catalog contains trailing JSON")
	}
	if catalog.SchemaVersion != 1 || len(catalog.Adapters) == 0 {
		return nil, errors.New("adapter catalog is empty or incompatible")
	}
	seen := map[string]bool{}
	for _, adapter := range catalog.Adapters {
		name := normalizeFixtureText(adapter.Name)
		if name == "" || seen[name] || !contains([]string{"EXPERIMENTAL", "GATED", "UNAVAILABLE"}, adapter.Status) || adapter.PolicySource != "../workflows/release-readiness.json" || strings.TrimSpace(adapter.Reason) == "" {
			return nil, errors.New("adapter descriptor is invalid")
		}
		if adapter.Capabilities.AttestedWorkerIdentity || adapter.Capabilities.WriteIsolationAttested || adapter.Capabilities.SecretIsolationAttested || adapter.Capabilities.NetworkIsolationAttested {
			return nil, errors.New("metadata-only adapter claims an attested capability")
		}
		seen[name] = true
	}
	return result{"schemaVersion": catalog.SchemaVersion, "adapters": catalog.Adapters, "assuranceRule": "DESCRIPTORS_NEVER_ESTABLISH_MANAGED_INDEPENDENT"}, nil
}

func packs(args []string) (result, error) {
	fs := flag.NewFlagSet("packs", flag.ContinueOnError)
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	if fs.NArg() != 0 {
		return nil, errors.New("packs accepts no positional arguments")
	}
	b, err := readBoundedRegularFile(filepath.Join(os.Getenv("JSDLC_PLUGIN_ROOT"), "packs", "catalog.json"), 256*1024)
	if err != nil {
		return nil, err
	}
	dec := json.NewDecoder(strings.NewReader(string(b)))
	dec.DisallowUnknownFields()
	var catalog packCatalog
	if err := dec.Decode(&catalog); err != nil {
		return nil, err
	}
	var trailing any
	if err := dec.Decode(&trailing); err != io.EOF {
		return nil, errors.New("pack catalog contains trailing JSON")
	}
	if catalog.SchemaVersion != 1 || catalog.Status != "GATED_BY_PHASE_3" || catalog.ActivationRule != "Do not activate packs or additional workflows until Phase-3 graduation evidence passes." || len(catalog.Packs) == 0 {
		return nil, errors.New("pack catalog is not safely gated")
	}
	seen := map[string]bool{}
	for _, pack := range catalog.Packs {
		name := normalizeFixtureText(pack.Name)
		if name == "" || seen[name] || len(pack.Domains) == 0 {
			return nil, errors.New("pack descriptor is invalid")
		}
		seen[name] = true
		values := map[string]bool{}
		for _, domain := range pack.Domains {
			if !contains(requiredReleaseDomains, domain) || values[domain] {
				return nil, errors.New("pack has invalid or duplicate domain")
			}
			values[domain] = true
		}
		for _, lens := range pack.ExtraLenses {
			lens = normalizeFixtureText(lens)
			if lens == "" || values["lens:"+lens] {
				return nil, errors.New("pack has blank or duplicate lens")
			}
			values["lens:"+lens] = true
		}
	}
	return result{"schemaVersion": catalog.SchemaVersion, "status": catalog.Status, "packs": catalog.Packs, "activationAllowed": false}, nil
}
