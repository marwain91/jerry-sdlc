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
	"reflect"
	"strings"
)

type workflowContract struct {
	SchemaVersion         int                 `json:"schemaVersion"`
	Name                  string              `json:"name"`
	TerminalStates        []string            `json:"terminalStates"`
	Roles                 []string            `json:"roles"`
	SpecialistAssignments []string            `json:"specialistAssignments"`
	States                []string            `json:"states"`
	Transitions           map[string][]string `json:"transitions"`
	RequiredDomains       []string            `json:"requiredDomains"`
	ForbiddenActions      []string            `json:"forbiddenActions"`
}

var releaseWorkflowRoles = []string{"orchestrator", "qa-architect", "qa-executor", "specialist-reviewer", "independent-verifier"}
var releaseWorkflowStates = []string{"BASELINED", "STRATEGY_READY", "EVIDENCE_COLLECTED", "REVIEWED", "ADJUDICATED", "VERIFIED"}
var releaseWorkflowTerminalStates = []string{"NOT_READY", "INCONCLUSIVE", "CANCELLED", "BLOCKED"}
var releaseWorkflowForbiddenActions = []string{"deploy", "publish", "tag", "release"}

func validateWorkflow(args []string) (result, error) {
	fs := flag.NewFlagSet("validate-workflow", flag.ContinueOnError)
	path := fs.String("file", "", "workflow contract JSON (defaults to the bundled release-readiness contract)")
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	if fs.NArg() != 0 {
		return nil, errors.New("positional arguments are forbidden")
	}
	if *path == "" {
		*path = filepath.Join(os.Getenv("JSDLC_PLUGIN_ROOT"), "workflows", "release-readiness.json")
	}
	b, err := readBoundedRegularFile(*path, 128*1024)
	if err != nil {
		return nil, err
	}
	contract, err := decodeWorkflowContract(b)
	if err != nil {
		return nil, err
	}
	if err := validateReleaseWorkflowParity(contract); err != nil {
		return nil, err
	}
	digest := sha256.Sum256(b)
	return result{"schemaVersion": 1, "workflow": contract.Name, "valid": true, "runtimeParity": "EXACT", "activationAllowed": false, "contractDigest": hex.EncodeToString(digest[:])}, nil
}

func decodeWorkflowContract(b []byte) (workflowContract, error) {
	dec := json.NewDecoder(strings.NewReader(string(b)))
	dec.DisallowUnknownFields()
	var contract workflowContract
	if err := dec.Decode(&contract); err != nil {
		return workflowContract{}, err
	}
	var trailing any
	if err := dec.Decode(&trailing); err != io.EOF {
		return workflowContract{}, errors.New("workflow contract contains trailing JSON")
	}
	return contract, nil
}

func validateReleaseWorkflowParity(contract workflowContract) error {
	if contract.SchemaVersion != 1 || contract.Name != "release-readiness" {
		return errors.New("workflow identity is incompatible")
	}
	if !reflect.DeepEqual(contract.Roles, releaseWorkflowRoles) || !reflect.DeepEqual(contract.SpecialistAssignments, specialistAssignments) || !reflect.DeepEqual(contract.States, releaseWorkflowStates) || !reflect.DeepEqual(contract.TerminalStates, releaseWorkflowTerminalStates) || !reflect.DeepEqual(contract.RequiredDomains, requiredReleaseDomains) || !reflect.DeepEqual(contract.ForbiddenActions, releaseWorkflowForbiddenActions) || !reflect.DeepEqual(contract.Transitions, allowedTransitions) {
		return errors.New("workflow contract does not exactly match runtime policy")
	}
	return nil
}
