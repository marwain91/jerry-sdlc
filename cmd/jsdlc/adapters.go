package main

import (
	"crypto/sha256"
	"encoding/hex"
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
	Name            string              `json:"name"`
	ProtocolVersion int                 `json:"protocolVersion"`
	Status          string              `json:"status"`
	Capabilities    adapterCapabilities `json:"capabilities"`
	PolicySource    string              `json:"policySource"`
	Reason          string              `json:"reason"`
}

type adapterRecord struct {
	SchemaVersion      int                 `json:"schemaVersion"`
	ProtocolVersion    int                 `json:"protocolVersion"`
	AdapterName        string              `json:"adapterName"`
	RunID              string              `json:"runId"`
	Candidate          string              `json:"candidateLabel"`
	Role               string              `json:"role"`
	RepositoryDigest   string              `json:"repositoryDigest"`
	PolicyDigest       string              `json:"policyDigest"`
	RoleContractDigest string              `json:"roleContractDigest"`
	WorkerInstanceID   string              `json:"workerInstanceId"`
	ResultDigest       string              `json:"resultDigest"`
	Mode               string              `json:"mode"`
	Capabilities       adapterCapabilities `json:"capabilities"`
	Trust              string              `json:"trust"`
	RecordedAt         string              `json:"recordedAt"`
}
type adapterCapabilities struct {
	EphemeralWorkersObserved bool `json:"ephemeralWorkersObserved"`
	WriteBlockingObserved    bool `json:"writeBlockingObserved"`
	AttestedWorkerIdentity   bool `json:"attestedWorkerIdentity"`
	WriteIsolationAttested   bool `json:"writeIsolationAttested"`
	SecretIsolationAttested  bool `json:"secretIsolationAttested"`
	NetworkIsolationAttested bool `json:"networkIsolationAttested"`
	StructuredOutputObserved bool `json:"structuredOutputObserved"`
	complete                 bool
}

func (c *adapterCapabilities) UnmarshalJSON(data []byte) error {
	type requiredCapabilities struct {
		EphemeralWorkersObserved *bool `json:"ephemeralWorkersObserved"`
		WriteBlockingObserved    *bool `json:"writeBlockingObserved"`
		AttestedWorkerIdentity   *bool `json:"attestedWorkerIdentity"`
		WriteIsolationAttested   *bool `json:"writeIsolationAttested"`
		SecretIsolationAttested  *bool `json:"secretIsolationAttested"`
		NetworkIsolationAttested *bool `json:"networkIsolationAttested"`
		StructuredOutputObserved *bool `json:"structuredOutputObserved"`
	}
	dec := json.NewDecoder(strings.NewReader(string(data)))
	dec.DisallowUnknownFields()
	var raw requiredCapabilities
	if err := dec.Decode(&raw); err != nil {
		return err
	}
	var trailing any
	if err := dec.Decode(&trailing); err != io.EOF {
		return errors.New("adapter capabilities contain trailing JSON")
	}
	if raw.EphemeralWorkersObserved == nil || raw.WriteBlockingObserved == nil || raw.AttestedWorkerIdentity == nil || raw.WriteIsolationAttested == nil || raw.SecretIsolationAttested == nil || raw.NetworkIsolationAttested == nil || raw.StructuredOutputObserved == nil {
		return errors.New("adapter capabilities omit a required field")
	}
	c.EphemeralWorkersObserved, c.WriteBlockingObserved = *raw.EphemeralWorkersObserved, *raw.WriteBlockingObserved
	c.AttestedWorkerIdentity, c.WriteIsolationAttested = *raw.AttestedWorkerIdentity, *raw.WriteIsolationAttested
	c.SecretIsolationAttested, c.NetworkIsolationAttested = *raw.SecretIsolationAttested, *raw.NetworkIsolationAttested
	c.StructuredOutputObserved, c.complete = *raw.StructuredOutputObserved, true
	return nil
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
		if name == "" || seen[name] || adapter.ProtocolVersion != 1 || !adapter.Capabilities.complete || !contains([]string{"EXPERIMENTAL", "GATED", "UNAVAILABLE"}, adapter.Status) || adapter.PolicySource != "../workflows/release-readiness.json" || strings.TrimSpace(adapter.Reason) == "" {
			return nil, errors.New("adapter descriptor is invalid")
		}
		if adapter.Capabilities.AttestedWorkerIdentity || adapter.Capabilities.WriteIsolationAttested || adapter.Capabilities.SecretIsolationAttested || adapter.Capabilities.NetworkIsolationAttested {
			return nil, errors.New("metadata-only adapter claims an attested capability")
		}
		seen[name] = true
	}
	return result{"schemaVersion": catalog.SchemaVersion, "adapters": catalog.Adapters, "assuranceRule": "DESCRIPTORS_NEVER_ESTABLISH_MANAGED_INDEPENDENT"}, nil
}

func validateAdapter(args []string) (result, error) {
	fs := flag.NewFlagSet("validate-adapter", flag.ContinueOnError)
	path := fs.String("file", "", "adapter protocol record JSON")
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	if *path == "" || fs.NArg() != 0 {
		return nil, errors.New("--file is required and positional arguments are forbidden")
	}
	b, err := readBoundedRegularFile(*path, 256*1024)
	if err != nil {
		return nil, err
	}
	dec := json.NewDecoder(strings.NewReader(string(b)))
	dec.DisallowUnknownFields()
	var record adapterRecord
	if err := dec.Decode(&record); err != nil {
		return nil, err
	}
	var trailing any
	if err := dec.Decode(&trailing); err != io.EOF {
		return nil, errors.New("adapter record contains trailing JSON")
	}
	if record.SchemaVersion != 1 || record.ProtocolVersion != 1 || normalizeFixtureText(record.AdapterName) == "" || record.RunID == "" || record.Candidate == "" || !contains([]string{"qa-architect", "qa-executor", "specialist-reviewer", "independent-verifier"}, record.Role) || !validSHA256(record.RepositoryDigest) || !validSHA256(record.PolicyDigest) || !validSHA256(record.RoleContractDigest) || !validSHA256(record.ResultDigest) || record.WorkerInstanceID == "" || record.Mode != "READ_ONLY" || !record.Capabilities.complete || record.Trust != "OBSERVED_UNATTESTED" || parseRFC3339(record.RecordedAt) != nil {
		return nil, errors.New("adapter record envelope is invalid")
	}
	if record.Capabilities.AttestedWorkerIdentity || record.Capabilities.WriteIsolationAttested || record.Capabilities.SecretIsolationAttested || record.Capabilities.NetworkIsolationAttested {
		return nil, errors.New("unattested adapter record claims an attested capability")
	}
	pluginRoot := os.Getenv("JSDLC_PLUGIN_ROOT")
	policy, err := readBoundedRegularFile(filepath.Join(pluginRoot, "workflows", "release-readiness.json"), 128*1024)
	if err != nil {
		return nil, err
	}
	roleContract, err := readBoundedRegularFile(filepath.Join(pluginRoot, "roles", record.Role+".md"), 128*1024)
	if err != nil {
		return nil, err
	}
	policyHash, roleHash := sha256.Sum256(policy), sha256.Sum256(roleContract)
	if record.PolicyDigest != hex.EncodeToString(policyHash[:]) || record.RoleContractDigest != hex.EncodeToString(roleHash[:]) {
		return nil, errors.New("adapter record is not bound to the canonical policy and role contract")
	}
	return result{"record": record, "valid": true, "assuranceEffect": "EVIDENCE_ONLY", "activationAllowed": false, "reason": "protocol validation does not authenticate the adapter or attest isolation"}, nil
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
