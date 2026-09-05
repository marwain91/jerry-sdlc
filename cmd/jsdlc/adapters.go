package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
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
	Workflow           string              `json:"workflow"`
	WorkflowVersion    int                 `json:"workflowVersion"`
	Role               string              `json:"role"`
	AssignmentID       string              `json:"assignmentId"`
	AssignmentDigest   string              `json:"assignmentDigest"`
	RepositoryDigest   string              `json:"repositoryDigest"`
	PolicyDigest       string              `json:"policyDigest"`
	RoleContractDigest string              `json:"roleContractDigest"`
	ContractSetDigest  string              `json:"contractSetDigest"`
	SchemaDigest       string              `json:"schemaDigest"`
	WorkerInstanceID   string              `json:"workerInstanceId"`
	ResultDigest       string              `json:"resultDigest"`
	ReplayID           string              `json:"replayId"`
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
		if name == "" || seen[name] || adapter.ProtocolVersion != 2 || !adapter.Capabilities.complete || !contains([]string{"EXPERIMENTAL", "GATED", "UNAVAILABLE"}, adapter.Status) || adapter.PolicySource != "../workflows/release-readiness.json" || strings.TrimSpace(adapter.Reason) == "" {
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
	resultPath := fs.String("result", "", "exact worker result bytes")
	assignmentPath := fs.String("assignment", "", "exact assignment input bytes")
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	if *path == "" || *resultPath == "" || *assignmentPath == "" || fs.NArg() != 0 {
		return nil, errors.New("--file, --result, and --assignment are required and positional arguments are forbidden")
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
	if record.SchemaVersion != 2 || record.ProtocolVersion != 2 || normalizeFixtureText(record.AdapterName) == "" || strings.TrimSpace(record.AdapterName) != record.AdapterName || len(record.AdapterName) > 128 || strings.TrimSpace(record.RunID) == "" || strings.TrimSpace(record.RunID) != record.RunID || len(record.RunID) > 256 || strings.TrimSpace(record.Candidate) == "" || strings.TrimSpace(record.Candidate) != record.Candidate || len(record.Candidate) > 256 || record.Workflow != "release-readiness" || record.WorkflowVersion != 1 || !contains([]string{"qa-architect", "qa-executor", "specialist-reviewer", "independent-verifier"}, record.Role) || !validRoleAssignment(record.Role, record.AssignmentID) || !validSHA256(record.AssignmentDigest) || !validSHA256(record.RepositoryDigest) || !validSHA256(record.PolicyDigest) || !validSHA256(record.RoleContractDigest) || !validSHA256(record.ContractSetDigest) || !validSHA256(record.SchemaDigest) || !validSHA256(record.ResultDigest) || !validSHA256(record.ReplayID) || strings.TrimSpace(record.WorkerInstanceID) == "" || strings.TrimSpace(record.WorkerInstanceID) != record.WorkerInstanceID || len(record.WorkerInstanceID) > 256 || record.Mode != "READ_ONLY" || !record.Capabilities.complete || record.Trust != "OBSERVED_UNATTESTED" || parseRFC3339(record.RecordedAt) != nil {
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
	schemaBytes, err := readBoundedRegularFile(filepath.Join(pluginRoot, "schemas", "worker-result.schema.json"), 128*1024)
	if err != nil {
		return nil, err
	}
	resultBytes, err := readBoundedRegularFile(*resultPath, 4*1024*1024)
	if err != nil {
		return nil, err
	}
	if err := validateWorkerReport(string(resultBytes)); err != nil {
		return nil, fmt.Errorf("worker result does not match the canonical result contract: %w", err)
	}
	assignmentBytes, err := readBoundedRegularFile(*assignmentPath, 256*1024)
	if err != nil {
		return nil, err
	}
	policyHash, roleHash, schemaHash := sha256.Sum256(policy), sha256.Sum256(roleContract), sha256.Sum256(schemaBytes)
	resultHash, assignmentHash := sha256.Sum256(resultBytes), sha256.Sum256(assignmentBytes)
	contractHasher := sha256.New()
	_, _ = contractHasher.Write(policy)
	_, _ = contractHasher.Write(schemaBytes)
	_, _ = io.WriteString(contractHasher, record.Role+"\x00")
	_, _ = contractHasher.Write(roleContract)
	contractSetDigest := hex.EncodeToString(contractHasher.Sum(nil))
	if record.PolicyDigest != hex.EncodeToString(policyHash[:]) || record.RoleContractDigest != hex.EncodeToString(roleHash[:]) || record.SchemaDigest != hex.EncodeToString(schemaHash[:]) || record.ContractSetDigest != contractSetDigest || record.ResultDigest != hex.EncodeToString(resultHash[:]) || record.AssignmentDigest != hex.EncodeToString(assignmentHash[:]) {
		return nil, errors.New("adapter record is not bound to the supplied bytes and canonical contract set")
	}
	replayID := adapterReplayID(record)
	if record.ReplayID != replayID {
		return nil, errors.New("adapter replay identity does not match the bound evidence")
	}
	recordHash := sha256.Sum256(b)
	return result{"record": record, "valid": true, "assuranceEffect": "EVIDENCE_ONLY", "activationAllowed": false, "recordDigest": hex.EncodeToString(recordHash[:]), "resultDigest": record.ResultDigest, "assignmentDigest": record.AssignmentDigest, "replayId": replayID, "provenance": "CALLER_SUPPLIED_UNATTESTED_BYTES", "reason": "protocol validation checks bytes and canonical contracts but does not authenticate the adapter, prevent replay, or attest isolation"}, nil
}

func adapterReplayID(record adapterRecord) string {
	values := []string{record.AdapterName, record.RunID, record.Candidate, record.Workflow, fmt.Sprint(record.WorkflowVersion), record.Role, record.AssignmentID, record.RepositoryDigest, record.AssignmentDigest, record.PolicyDigest, record.RoleContractDigest, record.ContractSetDigest, record.SchemaDigest, record.WorkerInstanceID, record.ResultDigest, record.Mode, record.Trust,
		fmt.Sprint(record.Capabilities.EphemeralWorkersObserved), fmt.Sprint(record.Capabilities.WriteBlockingObserved), fmt.Sprint(record.Capabilities.AttestedWorkerIdentity), fmt.Sprint(record.Capabilities.WriteIsolationAttested), fmt.Sprint(record.Capabilities.SecretIsolationAttested), fmt.Sprint(record.Capabilities.NetworkIsolationAttested), fmt.Sprint(record.Capabilities.StructuredOutputObserved)}
	h := sha256.New()
	for _, value := range values {
		_, _ = io.WriteString(h, value+"\x00")
	}
	return hex.EncodeToString(h.Sum(nil))
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
