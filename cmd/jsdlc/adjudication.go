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
	"time"
)

type adjudicationDecision struct {
	Kind        string   `json:"kind"`
	ID          string   `json:"id"`
	Disposition string   `json:"disposition"`
	Rationale   string   `json:"rationale"`
	EvidenceIDs []string `json:"evidenceIds"`
}

type adjudicationInput struct {
	SchemaVersion int                    `json:"schemaVersion"`
	TeamDigest    string                 `json:"teamEvidenceDigest"`
	Decisions     []adjudicationDecision `json:"decisions"`
}

type adjudicationEvidence struct {
	SchemaVersion    int                    `json:"schemaVersion"`
	RunID            string                 `json:"runId"`
	Repository       string                 `json:"repository"`
	Candidate        string                 `json:"candidateLabel"`
	RepositoryDigest string                 `json:"repositoryDigest"`
	TeamDigest       string                 `json:"teamEvidenceDigest"`
	Trust            string                 `json:"trust"`
	Decisions        []adjudicationDecision `json:"decisions"`
	RecordedAt       string                 `json:"recordedAt"`
}

func adjudicate(args []string) (result, error) {
	fs := flag.NewFlagSet("adjudicate", flag.ContinueOnError)
	repo := fs.String("repo", ".", "repository path")
	candidate := fs.String("candidate", "", "active candidate label")
	inputPath := fs.String("file", "", "adjudication decision JSON")
	authorized := fs.Bool("authorized", false, "confirm these explicit adjudication decisions")
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	if *candidate == "" || *inputPath == "" || !*authorized {
		return nil, errors.New("--candidate, --file, and --authorized are required")
	}
	input, err := readAdjudicationInput(*inputPath)
	if err != nil {
		return nil, err
	}
	abs, key, root, err := stateLocation(*repo)
	if err != nil {
		return nil, err
	}
	statePath := filepath.Join(root, key, "active.json")
	unlock, err := acquireLock(statePath)
	if err != nil {
		return nil, err
	}
	defer unlock()
	state, _, err := readState(root, key)
	if err != nil {
		return nil, err
	}
	if state.Repository != abs || state.Candidate != *candidate || isTerminalState(state.State) {
		return nil, errors.New("adjudication input does not match an active run")
	}
	currentDigest, err := digestRepository(abs)
	if err != nil || currentDigest != state.ContentDigest {
		return nil, errors.New("candidate changed before adjudication")
	}
	bundle, teamPath, err := loadLatestTeamEvidence(root, key, state.ID)
	if err != nil {
		return nil, err
	}
	teamDigest := strings.TrimSuffix(filepath.Base(teamPath), ".json")
	if input.TeamDigest != teamDigest {
		return nil, errors.New("adjudication is not bound to the latest team evidence")
	}
	checks, err := loadCheckEvidence(root, key, state.ID)
	if err != nil {
		return nil, err
	}
	if err := validateAdjudicationDecisions(input.Decisions, bundle, checks); err != nil {
		return nil, err
	}
	evidence := adjudicationEvidence{SchemaVersion: 1, RunID: state.ID, Repository: abs, Candidate: state.Candidate, RepositoryDigest: currentDigest, TeamDigest: teamDigest, Trust: "LOCAL_USER_AUTHORIZED", Decisions: input.Decisions, RecordedAt: time.Now().UTC().Format(time.RFC3339Nano)}
	dir := filepath.Join(root, key, "adjudications", state.ID, teamDigest)
	if err := os.MkdirAll(filepath.Dir(dir), 0o700); err != nil {
		return nil, err
	}
	if err := os.Mkdir(dir, 0o700); err == nil {
		// The directory itself is the one-shot reservation for this team digest.
	} else if os.IsExist(err) {
		return nil, errors.New("this team evidence was already adjudicated")
	} else {
		return nil, err
	}
	b, err := json.MarshalIndent(evidence, "", "  ")
	if err != nil {
		return nil, err
	}
	digest := sha256.Sum256(b)
	digestString := hex.EncodeToString(digest[:])
	referencePath := filepath.Join(root, key, "adjudications", state.ID, teamDigest+".digest")
	if err := writeExclusiveFile(referencePath, []byte(digestString+"\n"), 0o400); err != nil {
		return nil, errors.New("this team evidence was already adjudicated")
	}
	path := filepath.Join(dir, digestString+".json")
	if err := writeAtomic(path, b); err != nil {
		return nil, err
	}
	return result{"adjudication": evidence, "path": path, "digest": digestString, "readinessEffect": "DISPOSITIONS_ONLY_NO_ASSURANCE_UPGRADE"}, nil
}

func readAdjudicationInput(path string) (adjudicationInput, error) {
	b, err := readBoundedRegularFile(path, 256*1024)
	if err != nil {
		return adjudicationInput{}, err
	}
	dec := json.NewDecoder(strings.NewReader(string(b)))
	dec.DisallowUnknownFields()
	var input adjudicationInput
	if err := dec.Decode(&input); err != nil {
		return adjudicationInput{}, err
	}
	var trailing any
	if err := dec.Decode(&trailing); err != io.EOF {
		return adjudicationInput{}, errors.New("adjudication input contains trailing JSON")
	}
	if input.SchemaVersion != 1 || !validSHA256(input.TeamDigest) || len(input.Decisions) == 0 || len(input.Decisions) > 512 {
		return adjudicationInput{}, errors.New("invalid adjudication input envelope")
	}
	return input, nil
}

func validateAdjudicationDecisions(decisions []adjudicationDecision, bundle teamEvidence, checks map[string]checkEvidence) error {
	findings, notApplicable := map[string]bool{}, map[string]bool{}
	for _, role := range bundle.Roles {
		var report workerReport
		if err := json.Unmarshal(role.Report, &report); err != nil {
			return err
		}
		for _, finding := range report.Findings {
			if findings[finding.ID] {
				return errors.New("finding ID is duplicated across role reports")
			}
			findings[finding.ID] = true
		}
		for _, domain := range report.Domains {
			if domain.Status == "NOT_APPLICABLE" {
				notApplicable[domain.Domain] = true
			}
		}
	}
	seen := map[string]bool{}
	for _, decision := range decisions {
		key := decision.Kind + "\x00" + decision.ID
		if seen[key] || strings.TrimSpace(decision.Rationale) == "" || decision.EvidenceIDs == nil || !contains([]string{"ACCEPTED", "REJECTED"}, decision.Disposition) {
			return errors.New("invalid or duplicate adjudication decision")
		}
		seen[key] = true
		seenEvidence := map[string]bool{}
		for _, id := range decision.EvidenceIDs {
			if !validEvidenceID(id) || seenEvidence[id] {
				return errors.New("adjudication decision cites an invalid or duplicate evidence ID")
			}
			seenEvidence[id] = true
		}
		switch decision.Kind {
		case "FINDING":
			if !findings[decision.ID] {
				return errors.New("finding decision references an unknown finding")
			}
			if decision.Disposition == "REJECTED" {
				if len(decision.EvidenceIDs) == 0 {
					return errors.New("rejected finding lacks checked evidence")
				}
				for _, id := range decision.EvidenceIDs {
					evidence, ok := checks[id]
					if !ok || evidence.ExitStatus != 0 || evidence.TimedOut || evidence.RepositoryChanged {
						return errors.New("rejected finding cites invalid checked evidence")
					}
				}
			}
		case "DOMAIN_NOT_APPLICABLE":
			if !notApplicable[decision.ID] || !contains(requiredReleaseDomains, decision.ID) || len(decision.EvidenceIDs) == 0 {
				return errors.New("domain N/A decision is unknown or lacks evidence")
			}
			for _, id := range decision.EvidenceIDs {
				evidence, ok := checks[id]
				if !ok || evidence.ExitStatus != 0 || evidence.TimedOut || evidence.RepositoryChanged || !contains(evidence.Domains, decision.ID) {
					return errors.New("domain N/A decision cites invalid checked evidence")
				}
			}
		default:
			return errors.New("unknown adjudication decision kind")
		}
	}
	return nil
}

func loadAdjudication(root, key, runID, teamDigest string) (*adjudicationEvidence, error) {
	referencePath := filepath.Join(root, key, "adjudications", runID, teamDigest+".digest")
	reference, err := readBoundedRegularFile(referencePath, 128)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	authorizedDigest := strings.TrimSpace(string(reference))
	if !validSHA256(authorizedDigest) {
		return nil, errors.New("adjudication authorization anchor is invalid")
	}
	dir := filepath.Join(root, key, "adjudications", runID, teamDigest)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	if len(entries) != 1 || entries[0].IsDir() {
		return nil, errors.New("adjudication directory must contain exactly one evidence file")
	}
	name := entries[0].Name()
	if filepath.Ext(name) != ".json" || !validSHA256(strings.TrimSuffix(name, ".json")) {
		return nil, errors.New("adjudication evidence filename is not content-addressed")
	}
	if name != authorizedDigest+".json" {
		return nil, errors.New("adjudication evidence does not match its authorization anchor")
	}
	path := filepath.Join(dir, name)
	b, err := readBoundedRegularFile(path, 512*1024)
	if err != nil {
		return nil, err
	}
	digest := sha256.Sum256(b)
	if hex.EncodeToString(digest[:])+".json" != name {
		return nil, errors.New("adjudication evidence digest does not match its filename")
	}
	dec := json.NewDecoder(strings.NewReader(string(b)))
	dec.DisallowUnknownFields()
	var evidence adjudicationEvidence
	if err := dec.Decode(&evidence); err != nil {
		return nil, err
	}
	var trailing any
	if err := dec.Decode(&trailing); err != io.EOF {
		return nil, errors.New("adjudication evidence contains trailing JSON")
	}
	if evidence.SchemaVersion != 1 || evidence.RunID != runID || evidence.Repository == "" || evidence.Candidate == "" || evidence.TeamDigest != teamDigest || evidence.Trust != "LOCAL_USER_AUTHORIZED" || !validSHA256(evidence.RepositoryDigest) || len(evidence.Decisions) == 0 || len(evidence.Decisions) > 512 || parseRFC3339(evidence.RecordedAt) != nil {
		return nil, errors.New("invalid adjudication evidence envelope")
	}
	return &evidence, nil
}

func writeExclusiveFile(path string, contents []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	ok := false
	defer func() {
		_ = f.Close()
		if !ok {
			_ = os.Remove(path)
		}
	}()
	if _, err := f.Write(contents); err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	dir, err := os.Open(filepath.Dir(path))
	if err != nil {
		return err
	}
	defer dir.Close()
	if err := dir.Sync(); err != nil {
		return err
	}
	ok = true
	return nil
}
