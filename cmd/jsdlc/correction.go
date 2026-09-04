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
	"time"
)

type correctionInput struct {
	SchemaVersion int      `json:"schemaVersion"`
	TeamDigest    string   `json:"teamEvidenceDigest"`
	FindingIDs    []string `json:"findingIds"`
	AllowedPaths  []string `json:"allowedPaths"`
}

type manifestEntry struct {
	Path   string `json:"path"`
	Mode   string `json:"mode"`
	Digest string `json:"digest,omitempty"`
}

type correctionAuthorization struct {
	SchemaVersion    int             `json:"schemaVersion"`
	RunID            string          `json:"runId"`
	Repository       string          `json:"repository"`
	Candidate        string          `json:"candidateLabel"`
	RepositoryDigest string          `json:"repositoryDigest"`
	TeamDigest       string          `json:"teamEvidenceDigest"`
	FindingIDs       []string        `json:"findingIds"`
	AllowedPaths     []string        `json:"allowedPaths"`
	Baseline         []manifestEntry `json:"baselineManifest"`
	CorrectionCycle  int             `json:"correctionCycle"`
	Trust            string          `json:"trust"`
	AuthorizedAt     string          `json:"authorizedAt"`
}

type correctionEvidence struct {
	SchemaVersion       int      `json:"schemaVersion"`
	AuthorizationDigest string   `json:"authorizationDigest"`
	PreviousRunID       string   `json:"previousRunId"`
	NewRunID            string   `json:"newRunId"`
	Repository          string   `json:"repository"`
	PreviousCandidate   string   `json:"previousCandidateLabel"`
	NewCandidate        string   `json:"newCandidateLabel"`
	PreviousDigest      string   `json:"previousRepositoryDigest"`
	NewDigest           string   `json:"newRepositoryDigest"`
	FindingIDs          []string `json:"findingIds"`
	ChangedPaths        []string `json:"changedPaths"`
	CorrectionCycle     int      `json:"correctionCycle"`
	CompletedAt         string   `json:"completedAt"`
}

func authorizeCorrection(args []string) (result, error) {
	fs := flag.NewFlagSet("authorize-correction", flag.ContinueOnError)
	repo := fs.String("repo", ".", "repository path")
	candidate := fs.String("candidate", "", "active candidate label")
	inputPath := fs.String("file", "", "correction authorization JSON")
	authorized := fs.Bool("authorized", false, "authorize the bounded correction")
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	if *candidate == "" || *inputPath == "" || !*authorized {
		return nil, errors.New("--candidate, --file, and --authorized are required")
	}
	input, err := readCorrectionInput(*inputPath)
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
		return nil, errors.New("correction authorization does not match an active run")
	}
	if state.CorrectionCycle >= 2 {
		return nil, errors.New("maximum of two correction cycles reached")
	}
	currentDigest, err := digestRepository(abs)
	if err != nil || currentDigest != state.ContentDigest {
		return nil, errors.New("candidate changed before correction authorization")
	}
	bundle, teamPath, err := loadLatestTeamEvidence(root, key, state.ID)
	if err != nil {
		return nil, err
	}
	teamDigest := strings.TrimSuffix(filepath.Base(teamPath), ".json")
	if input.TeamDigest != teamDigest {
		return nil, errors.New("correction is not bound to the latest team evidence")
	}
	adjudication, err := loadAdjudication(root, key, state.ID, teamDigest)
	if err != nil || adjudication == nil {
		return nil, errors.New("valid adjudication is required before correction")
	}
	checks, err := loadCheckEvidence(root, key, state.ID)
	if err != nil {
		return nil, err
	}
	if err := validateAdjudicationDecisions(adjudication.Decisions, bundle, checks); err != nil {
		return nil, errors.New("valid adjudication is required before correction")
	}
	accepted := map[string]bool{}
	for _, decision := range adjudication.Decisions {
		if decision.Kind == "FINDING" && decision.Disposition == "ACCEPTED" {
			accepted[decision.ID] = true
		}
	}
	for _, id := range input.FindingIDs {
		if !accepted[id] {
			return nil, fmt.Errorf("finding %q is not accepted for correction", id)
		}
	}
	manifest, err := repositoryManifest(abs)
	if err != nil {
		return nil, err
	}
	manifestDigest, err := digestRepository(abs)
	if err != nil || manifestDigest != currentDigest {
		return nil, errors.New("candidate changed while correction authorization was captured")
	}
	evidence := correctionAuthorization{SchemaVersion: 1, RunID: state.ID, Repository: abs, Candidate: state.Candidate, RepositoryDigest: currentDigest, TeamDigest: teamDigest, FindingIDs: input.FindingIDs, AllowedPaths: input.AllowedPaths, Baseline: manifest, CorrectionCycle: state.CorrectionCycle + 1, Trust: "LOCAL_USER_AUTHORIZED", AuthorizedAt: time.Now().UTC().Format(time.RFC3339Nano)}
	b, err := json.MarshalIndent(evidence, "", "  ")
	if err != nil {
		return nil, err
	}
	digest := sha256.Sum256(b)
	digestString := hex.EncodeToString(digest[:])
	authorizationDir := filepath.Join(root, key, "corrections", "authorizations", state.ID)
	path := filepath.Join(authorizationDir, teamDigest+".json")
	if err := writeAtomicExclusive(path, b, 0o400); err != nil {
		existingBytes, readErr := readBoundedRegularFile(path, 64*1024*1024)
		if readErr != nil {
			return nil, errors.New("correction authorization already exists or cannot be persisted")
		}
		existingHash := sha256.Sum256(existingBytes)
		existingDigest := hex.EncodeToString(existingHash[:])
		existing, loadErr := loadCorrectionAuthorization(path, existingDigest)
		if loadErr != nil || existing.RunID != state.ID || existing.Repository != abs || existing.Candidate != state.Candidate || existing.RepositoryDigest != currentDigest || existing.TeamDigest != teamDigest || existing.CorrectionCycle != state.CorrectionCycle+1 || !equalStrings(existing.FindingIDs, input.FindingIDs) || !equalStrings(existing.AllowedPaths, input.AllowedPaths) || !equalManifest(existing.Baseline, manifest) {
			return nil, errors.New("different or invalid correction authorization already exists for this team evidence")
		}
		return result{"authorization": existing, "authorizationDigest": existingDigest, "path": path, "recovered": true, "next": "apply only the authorized findings within allowedPaths, then run finish-correction"}, nil
	}
	return result{"authorization": evidence, "authorizationDigest": digestString, "path": path, "next": "apply only the authorized findings within allowedPaths, then run finish-correction"}, nil
}

func finishCorrection(args []string) (result, error) {
	fs := flag.NewFlagSet("finish-correction", flag.ContinueOnError)
	repo := fs.String("repo", ".", "repository path")
	candidate := fs.String("candidate", "", "current candidate label")
	newCandidate := fs.String("new-candidate", "", "new candidate label")
	authorizationDigest := fs.String("authorization", "", "correction authorization digest")
	authorized := fs.Bool("authorized", false, "confirm correction completion and run rollover")
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	if *candidate == "" || *newCandidate == "" || *newCandidate == *candidate || !validSHA256(*authorizationDigest) || !*authorized {
		return nil, errors.New("distinct --candidate and --new-candidate, --authorization, and --authorized are required")
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
		return nil, errors.New("correction completion does not match an active run")
	}
	authorizationDir := filepath.Join(root, key, "corrections", "authorizations", state.ID)
	auth, _, err := loadCorrectionAuthorizationForRun(authorizationDir, *authorizationDigest)
	if err != nil {
		return nil, err
	}
	if auth.RunID != state.ID || auth.Repository != abs || auth.Candidate != state.Candidate || auth.RepositoryDigest != state.ContentDigest || auth.CorrectionCycle != state.CorrectionCycle+1 {
		return nil, errors.New("correction authorization is stale or identity-mismatched")
	}
	beforeManifestDigest, err := digestRepository(abs)
	if err != nil {
		return nil, err
	}
	current, err := repositoryManifest(abs)
	if err != nil {
		return nil, err
	}
	changed := changedManifestPaths(auth.Baseline, current)
	if len(changed) == 0 {
		return nil, errors.New("correction made no repository changes")
	}
	for _, path := range changed {
		if !pathAllowed(path, auth.AllowedPaths) {
			return nil, fmt.Errorf("correction changed unauthorized path %q", path)
		}
	}
	newDigest, err := digestRepository(abs)
	if err != nil {
		return nil, err
	}
	if newDigest != beforeManifestDigest {
		return nil, errors.New("candidate changed while correction completion was captured")
	}
	completionPath := filepath.Join(root, key, "corrections", "completed", state.ID, *authorizationDigest+".json")
	completed, err := loadCorrectionEvidence(completionPath)
	if os.IsNotExist(err) {
		newID, idErr := newRunID(key)
		if idErr != nil {
			return nil, idErr
		}
		now := time.Now().UTC().Format(time.RFC3339Nano)
		completed = correctionEvidence{SchemaVersion: 1, AuthorizationDigest: *authorizationDigest, PreviousRunID: state.ID, NewRunID: newID, Repository: abs, PreviousCandidate: state.Candidate, NewCandidate: *newCandidate, PreviousDigest: state.ContentDigest, NewDigest: newDigest, FindingIDs: auth.FindingIDs, ChangedPaths: changed, CorrectionCycle: auth.CorrectionCycle, CompletedAt: now}
		b, marshalErr := json.MarshalIndent(completed, "", "  ")
		if marshalErr != nil {
			return nil, marshalErr
		}
		if err := writeExclusiveFile(completionPath, b, 0o400); err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}
	if completed.AuthorizationDigest != *authorizationDigest || completed.PreviousRunID != state.ID || completed.Repository != abs || completed.PreviousCandidate != state.Candidate || completed.NewCandidate != *newCandidate || completed.PreviousDigest != state.ContentDigest || completed.NewDigest != newDigest || completed.CorrectionCycle != auth.CorrectionCycle || !equalStrings(completed.FindingIDs, auth.FindingIDs) || !equalStrings(completed.ChangedPaths, changed) {
		return nil, errors.New("persisted correction completion does not match the authorized change")
	}
	now := completed.CompletedAt
	newState := runState{SchemaVersion: 2, ID: completed.NewRunID, Workflow: state.Workflow, State: "BASELINED", Assurance: state.Assurance, Repository: abs, Candidate: *newCandidate, ContentDigest: newDigest, ParentRunID: state.ID, CorrectionCycle: auth.CorrectionCycle, CreatedAt: now, UpdatedAt: now}
	terminal := state
	terminal.State, terminal.UpdatedAt = "NOT_READY", now
	if err := archiveState(filepath.Join(root, key), terminal); err != nil {
		return nil, err
	}
	if finishCorrectionBeforeStateWriteHook != nil {
		if err := finishCorrectionBeforeStateWriteHook(); err != nil {
			return nil, err
		}
	}
	if err := writeState(statePath, newState); err != nil {
		return nil, err
	}
	return result{"previousRun": terminal, "run": newState, "changedPaths": changed, "completionPath": completionPath, "requiresFreshChecksAndTeam": true}, nil
}

func readCorrectionInput(path string) (correctionInput, error) {
	b, err := readBoundedRegularFile(path, 256*1024)
	if err != nil {
		return correctionInput{}, err
	}
	dec := json.NewDecoder(strings.NewReader(string(b)))
	dec.DisallowUnknownFields()
	var input correctionInput
	if err := dec.Decode(&input); err != nil {
		return input, err
	}
	var trailing any
	if err := dec.Decode(&trailing); err != io.EOF {
		return input, errors.New("correction input contains trailing JSON")
	}
	if input.SchemaVersion != 1 || !validSHA256(input.TeamDigest) || len(input.FindingIDs) == 0 || len(input.FindingIDs) > 128 || len(input.AllowedPaths) == 0 || len(input.AllowedPaths) > 256 {
		return input, errors.New("invalid correction input envelope")
	}
	seenIDs, seenPaths := map[string]bool{}, map[string]bool{}
	for _, id := range input.FindingIDs {
		if id == "" || seenIDs[id] {
			return input, errors.New("invalid or duplicate correction finding ID")
		}
		seenIDs[id] = true
	}
	for _, path := range input.AllowedPaths {
		if !validAllowedPath(path) || seenPaths[path] {
			return input, errors.New("invalid or duplicate allowed path")
		}
		seenPaths[path] = true
	}
	return input, nil
}

func validAllowedPath(path string) bool {
	trimmed := strings.TrimSuffix(filepath.ToSlash(path), "/")
	return path != "" && !filepath.IsAbs(path) && trimmed != "." && trimmed != ".." && !strings.HasPrefix(trimmed, "../") && filepath.ToSlash(filepath.Clean(trimmed)) == trimmed
}

func pathAllowed(path string, allowed []string) bool {
	for _, rule := range allowed {
		clean := strings.TrimSuffix(filepath.ToSlash(rule), "/")
		if path == clean || (strings.HasSuffix(rule, "/") && strings.HasPrefix(path, clean+"/")) {
			return true
		}
	}
	return false
}

func repositoryManifest(root string) ([]manifestEntry, error) {
	entries := []manifestEntry{}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		if entry.IsDir() && (rel == ".git" || strings.HasPrefix(rel, ".git"+string(filepath.Separator))) {
			return filepath.SkipDir
		}
		if len(entries) >= 100000 {
			return errors.New("repository manifest exceeds 100000 entries")
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		item := manifestEntry{Path: filepath.ToSlash(rel), Mode: info.Mode().String()}
		if info.Mode()&os.ModeSymlink != 0 {
			return errors.New("correction scope auditing does not support repositories containing symlinks")
		} else if info.Mode().IsRegular() {
			b, err := readBoundedRegularFile(path, 1<<30)
			if err != nil {
				return err
			}
			sum := sha256.Sum256(b)
			item.Digest = hex.EncodeToString(sum[:])
		}
		entries = append(entries, item)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })
	return entries, nil
}

func changedManifestPaths(before, after []manifestEntry) []string {
	a, b := map[string]manifestEntry{}, map[string]manifestEntry{}
	for _, item := range before {
		a[item.Path] = item
	}
	for _, item := range after {
		b[item.Path] = item
	}
	changed := []string{}
	for path, old := range a {
		if current, ok := b[path]; !ok || current != old {
			changed = append(changed, path)
		}
		delete(b, path)
	}
	for path := range b {
		changed = append(changed, path)
	}
	sort.Strings(changed)
	return changed
}

func loadCorrectionAuthorization(path, expectedDigest string) (correctionAuthorization, error) {
	b, err := readBoundedRegularFile(path, 64*1024*1024)
	if err != nil {
		return correctionAuthorization{}, err
	}
	sum := sha256.Sum256(b)
	if hex.EncodeToString(sum[:]) != expectedDigest {
		return correctionAuthorization{}, errors.New("correction authorization digest mismatch")
	}
	dec := json.NewDecoder(strings.NewReader(string(b)))
	dec.DisallowUnknownFields()
	var auth correctionAuthorization
	if err := dec.Decode(&auth); err != nil {
		return auth, err
	}
	var trailing any
	if err := dec.Decode(&trailing); err != io.EOF {
		return auth, errors.New("correction authorization contains trailing JSON")
	}
	if auth.SchemaVersion != 1 || auth.Trust != "LOCAL_USER_AUTHORIZED" || !validSHA256(auth.RepositoryDigest) || !validSHA256(auth.TeamDigest) || auth.CorrectionCycle < 1 || auth.CorrectionCycle > 2 || parseRFC3339(auth.AuthorizedAt) != nil {
		return auth, errors.New("invalid correction authorization envelope")
	}
	if auth.RunID == "" || auth.Repository == "" || auth.Candidate == "" || len(auth.FindingIDs) == 0 || len(auth.FindingIDs) > 128 || len(auth.AllowedPaths) == 0 || len(auth.AllowedPaths) > 256 || len(auth.Baseline) > 100000 {
		return auth, errors.New("invalid correction authorization scope")
	}
	seenIDs, seenPaths, previousPath := map[string]bool{}, map[string]bool{}, ""
	for _, id := range auth.FindingIDs {
		if id == "" || seenIDs[id] {
			return auth, errors.New("invalid correction finding IDs")
		}
		seenIDs[id] = true
	}
	for _, path := range auth.AllowedPaths {
		if !validAllowedPath(path) || seenPaths[path] {
			return auth, errors.New("invalid correction allowed paths")
		}
		seenPaths[path] = true
	}
	for _, item := range auth.Baseline {
		if !validAllowedPath(item.Path) || item.Mode == "" || item.Path <= previousPath || (item.Digest != "" && !validSHA256(item.Digest)) {
			return auth, errors.New("invalid correction baseline manifest")
		}
		previousPath = item.Path
	}
	return auth, nil
}

func loadCorrectionAuthorizationForRun(dir, expectedDigest string) (correctionAuthorization, string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return correctionAuthorization{}, "", err
	}
	if len(entries) == 0 || len(entries) > 32 {
		return correctionAuthorization{}, "", errors.New("correction authorization count is invalid")
	}
	var found correctionAuthorization
	foundPath := ""
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			return correctionAuthorization{}, "", errors.New("correction authorization directory contains an unexpected entry")
		}
		path := filepath.Join(dir, entry.Name())
		auth, loadErr := loadCorrectionAuthorization(path, expectedDigest)
		if loadErr == nil {
			if foundPath != "" {
				return correctionAuthorization{}, "", errors.New("correction authorization digest is duplicated")
			}
			found, foundPath = auth, path
		}
	}
	if foundPath == "" {
		return correctionAuthorization{}, "", errors.New("correction authorization digest was not found")
	}
	return found, foundPath, nil
}

var finishCorrectionBeforeStateWriteHook func() error

func writeAtomicExclusive(path string, contents []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".jsdlc-exclusive-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(contents); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Link(tmpPath, path); err != nil {
		return err
	}
	dir, err := os.Open(filepath.Dir(path))
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}

func loadCorrectionEvidence(path string) (correctionEvidence, error) {
	b, err := readBoundedRegularFile(path, 4*1024*1024)
	if err != nil {
		return correctionEvidence{}, err
	}
	dec := json.NewDecoder(strings.NewReader(string(b)))
	dec.DisallowUnknownFields()
	var evidence correctionEvidence
	if err := dec.Decode(&evidence); err != nil {
		return evidence, err
	}
	var trailing any
	if err := dec.Decode(&trailing); err != io.EOF {
		return evidence, errors.New("correction evidence contains trailing JSON")
	}
	if evidence.SchemaVersion != 1 || !validSHA256(evidence.AuthorizationDigest) || evidence.PreviousRunID == "" || evidence.NewRunID == "" || evidence.Repository == "" || evidence.PreviousCandidate == "" || evidence.NewCandidate == "" || !validSHA256(evidence.PreviousDigest) || !validSHA256(evidence.NewDigest) || len(evidence.FindingIDs) == 0 || len(evidence.ChangedPaths) == 0 || evidence.CorrectionCycle < 1 || evidence.CorrectionCycle > 2 || parseRFC3339(evidence.CompletedAt) != nil {
		return evidence, errors.New("invalid correction evidence envelope")
	}
	return evidence, nil
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for index := range a {
		if a[index] != b[index] {
			return false
		}
	}
	return true
}

func equalManifest(a, b []manifestEntry) bool {
	if len(a) != len(b) {
		return false
	}
	for index := range a {
		if a[index] != b[index] {
			return false
		}
	}
	return true
}
