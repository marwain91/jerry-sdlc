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
	"sort"
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
	Lenses         []lensDescriptor `json:"lenses"`
	Packs          []packDescriptor `json:"packs"`
	ActivationRule string           `json:"activationRule"`
}
type packCompatibility struct {
	PluginSchema    int    `json:"pluginSchema"`
	Workflow        string `json:"workflow"`
	WorkflowVersion int    `json:"workflowVersion"`
}
type lensDescriptor struct {
	ID             string `json:"id"`
	Version        int    `json:"version"`
	Status         string `json:"status"`
	Domain         string `json:"domain"`
	Objective      string `json:"objective"`
	ContractSource string `json:"contractSource"`
	PolicySource   string `json:"policySource"`
	PolicyDigest   string `json:"policyDigest"`
}
type packDescriptor struct {
	ID            string            `json:"id"`
	Version       int               `json:"version"`
	Status        string            `json:"status"`
	Compatibility packCompatibility `json:"compatibility"`
	Triggers      []string          `json:"triggers"`
	Domains       []string          `json:"domains"`
	Lenses        []string          `json:"lenses"`
	Dependencies  []string          `json:"dependencies"`
	Conflicts     []string          `json:"conflicts"`
	PolicySource  string            `json:"policySource"`
	PolicyDigest  string            `json:"policyDigest"`
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
	conformSet := fs.String("conform-set", "", "comma-separated inert pack IDs to merge-check")
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
	root := os.Getenv("JSDLC_PLUGIN_ROOT")
	if err := validatePluginJSONReferences(root); err != nil {
		return nil, err
	}
	if err := validatePackCatalog(root, catalog); err != nil {
		return nil, err
	}
	merged, conflicts, err := conformPackSet(catalog, *conformSet)
	if err != nil {
		return nil, err
	}
	return result{"schemaVersion": catalog.SchemaVersion, "status": catalog.Status, "packs": catalog.Packs, "lenses": catalog.Lenses, "activationAllowed": false, "conformanceOnly": true, "merged": merged, "conflicts": conflicts}, nil
}

func validatePackCatalog(root string, catalog packCatalog) error {
	if catalog.SchemaVersion != 2 || catalog.Status != "GATED_BY_PHASE_3" || catalog.ActivationRule != "Do not activate packs or additional workflows until Phase-3 graduation evidence passes." || len(catalog.Packs) == 0 || len(catalog.Lenses) == 0 {
		return errors.New("pack catalog is not safely gated")
	}
	policyPath := filepath.Join(root, "workflows", "release-readiness.json")
	policy, err := readBoundedRegularFile(policyPath, 128*1024)
	if err != nil {
		return err
	}
	policyHash := sha256.Sum256(policy)
	expectedPolicyDigest := hex.EncodeToString(policyHash[:])
	lenses := map[string]bool{}
	for _, lens := range catalog.Lenses {
		id := normalizeFixtureText(lens.ID)
		if id == "" || id != lens.ID || lenses[id] || lens.Version != 1 || lens.Status != "GATED_BY_PHASE_3" || !contains(requiredReleaseDomains, lens.Domain) || strings.TrimSpace(lens.Objective) == "" || strings.TrimSpace(lens.Objective) != lens.Objective || len(lens.Objective) > 512 || lens.ContractSource != "../roles/specialist-reviewer.md" || lens.PolicySource != "../workflows/release-readiness.json" || lens.PolicyDigest != expectedPolicyDigest {
			return errors.New("lens descriptor is invalid, duplicated, or not canonically bound")
		}
		lenses[id] = true
	}
	seen := map[string]bool{}
	for _, pack := range catalog.Packs {
		id := normalizeFixtureText(pack.ID)
		if id == "" || id != pack.ID || seen[id] || pack.Version != 1 || pack.Status != "GATED_BY_PHASE_3" || pack.Compatibility != (packCompatibility{PluginSchema: 1, Workflow: "release-readiness", WorkflowVersion: 1}) || len(pack.Triggers) == 0 || len(pack.Domains) == 0 || pack.Lenses == nil || pack.Dependencies == nil || pack.Conflicts == nil || pack.PolicySource != "../workflows/release-readiness.json" || pack.PolicyDigest != expectedPolicyDigest {
			return errors.New("pack descriptor is invalid")
		}
		seen[id] = true
		values := map[string]bool{}
		for _, trigger := range pack.Triggers {
			normalized := normalizeFixtureText(trigger)
			if normalized == "" || normalized != trigger || values["trigger:"+normalized] {
				return errors.New("pack has invalid or duplicate trigger")
			}
			values["trigger:"+normalized] = true
		}
		for _, domain := range pack.Domains {
			if !contains(requiredReleaseDomains, domain) || values[domain] {
				return errors.New("pack has invalid or duplicate domain")
			}
			values[domain] = true
		}
		for _, lens := range pack.Lenses {
			if !lenses[lens] || values["lens:"+lens] {
				return errors.New("pack has unknown or duplicate lens")
			}
			values["lens:"+lens] = true
		}
	}
	for _, pack := range catalog.Packs {
		for _, relation := range append(append([]string{}, pack.Dependencies...), pack.Conflicts...) {
			if !seen[relation] || relation == pack.ID {
				return errors.New("pack dependency or conflict is dangling or self-referential")
			}
		}
		if duplicateNormalized(pack.Dependencies) || duplicateNormalized(pack.Conflicts) {
			return errors.New("pack dependencies or conflicts are duplicated")
		}
		for _, dependency := range pack.Dependencies {
			if contains(pack.Conflicts, dependency) {
				return errors.New("a pack cannot both depend on and conflict with the same pack")
			}
		}
		for _, conflict := range pack.Conflicts {
			other := findPack(catalog.Packs, conflict)
			if other == nil || !contains(other.Conflicts, pack.ID) {
				return errors.New("pack conflict must be symmetric")
			}
		}
	}
	if packDependencyCycle(catalog.Packs) {
		return errors.New("pack dependencies contain a cycle")
	}
	return nil
}

func duplicateNormalized(values []string) bool {
	seen := map[string]bool{}
	for _, value := range values {
		normalized := normalizeFixtureText(value)
		if normalized == "" || normalized != value || seen[normalized] {
			return true
		}
		seen[normalized] = true
	}
	return false
}

func findPack(packs []packDescriptor, id string) *packDescriptor {
	for i := range packs {
		if packs[i].ID == id {
			return &packs[i]
		}
	}
	return nil
}

func packDependencyCycle(packs []packDescriptor) bool {
	visiting, visited := map[string]bool{}, map[string]bool{}
	var visit func(string) bool
	visit = func(id string) bool {
		if visiting[id] {
			return true
		}
		if visited[id] {
			return false
		}
		visiting[id] = true
		pack := findPack(packs, id)
		if pack != nil {
			for _, dependency := range pack.Dependencies {
				if visit(dependency) {
					return true
				}
			}
		}
		delete(visiting, id)
		visited[id] = true
		return false
	}
	for _, pack := range packs {
		if visit(pack.ID) {
			return true
		}
	}
	return false
}

func conformPackSet(catalog packCatalog, requested string) (result, []string, error) {
	selected := map[string]bool{}
	var add func(string) error
	add = func(id string) error {
		pack := findPack(catalog.Packs, id)
		if pack == nil {
			return fmt.Errorf("unknown pack %q", id)
		}
		if selected[id] {
			return nil
		}
		selected[id] = true
		for _, dependency := range pack.Dependencies {
			if err := add(dependency); err != nil {
				return err
			}
		}
		return nil
	}
	if requested != "" {
		for _, id := range strings.Split(requested, ",") {
			if id == "" || normalizeFixtureText(id) != id {
				return nil, nil, errors.New("pack set contains a blank or non-canonical ID")
			}
			if err := add(id); err != nil {
				return nil, nil, err
			}
		}
	}
	packIDs, domains, lenses, conflicts := []string{}, []string{}, []string{}, []string{}
	domainSet, lensSet := map[string]bool{}, map[string]bool{}
	for _, pack := range catalog.Packs {
		if !selected[pack.ID] {
			continue
		}
		packIDs = append(packIDs, pack.ID)
		for _, domain := range pack.Domains {
			domainSet[domain] = true
		}
		for _, lens := range pack.Lenses {
			lensSet[lens] = true
		}
		for _, conflict := range pack.Conflicts {
			if selected[conflict] && pack.ID < conflict {
				conflicts = append(conflicts, pack.ID+":"+conflict)
			}
		}
	}
	for domain := range domainSet {
		domains = append(domains, domain)
	}
	for lens := range lensSet {
		lenses = append(lenses, lens)
	}
	sort.Strings(packIDs)
	sort.Strings(domains)
	sort.Strings(lenses)
	sort.Strings(conflicts)
	return result{"packs": packIDs, "domains": domains, "lenses": lenses, "conflictFree": len(conflicts) == 0}, conflicts, nil
}

func validatePluginJSONReferences(root string) error {
	absRoot, err := filepath.Abs(root)
	if err != nil || root == "" {
		return errors.New("plugin root is unavailable")
	}
	for _, directory := range []string{"adapters", "packs", "schemas", "workflows"} {
		err := filepath.Walk(filepath.Join(absRoot, directory), func(path string, info os.FileInfo, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if info.Mode()&os.ModeSymlink != 0 {
				return fmt.Errorf("plugin metadata path is a symlink: %s", path)
			}
			if info.IsDir() || filepath.Ext(path) != ".json" {
				return nil
			}
			b, readErr := readBoundedRegularFile(path, 256*1024)
			if readErr != nil {
				return readErr
			}
			var document any
			if err := json.Unmarshal(b, &document); err != nil {
				return fmt.Errorf("invalid shipped JSON %s: %w", path, err)
			}
			return validateJSONReferences(absRoot, filepath.Dir(path), document)
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func validateJSONReferences(root, base string, value any) error {
	switch typed := value.(type) {
	case []any:
		for _, item := range typed {
			if err := validateJSONReferences(root, base, item); err != nil {
				return err
			}
		}
	case map[string]any:
		for key, item := range typed {
			if text, ok := item.(string); ok && (key == "$ref" || strings.HasSuffix(key, "Source")) && !strings.HasPrefix(text, "#") && !strings.Contains(text, "://") {
				target := strings.SplitN(text, "#", 2)[0]
				resolved := filepath.Clean(filepath.Join(base, target))
				rel, err := filepath.Rel(root, resolved)
				if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
					return fmt.Errorf("metadata reference escapes plugin root: %s", text)
				}
				if _, err := readBoundedRegularFile(resolved, 256*1024); err != nil {
					return fmt.Errorf("metadata reference is invalid: %s: %w", text, err)
				}
			}
			if err := validateJSONReferences(root, base, item); err != nil {
				return err
			}
		}
	}
	return nil
}
