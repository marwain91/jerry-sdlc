package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"
	"time"
)

type workflowRoutingSuite struct {
	SchemaVersion int                   `json:"schemaVersion"`
	Cases         []workflowRoutingCase `json:"cases"`
}

type workflowRoutingCase struct {
	ID       string `json:"id"`
	Request  string `json:"request"`
	Files    string `json:"files"`
	Workflow string `json:"workflow"`
}

type workflowScore struct {
	TruePositive  int `json:"truePositive"`
	FalsePositive int `json:"falsePositive"`
	FalseNegative int `json:"falseNegative"`
}

func evalWorkflows(args []string) (result, error) {
	fs := flag.NewFlagSet("eval-workflows", flag.ContinueOnError)
	fixture := fs.String("fixture", "", "multiclass workflow-routing fixture")
	repeats := fs.Int("repeats", 3, "trials per labelled request")
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	if *fixture == "" || *repeats < 1 || *repeats > 20 || fs.NArg() != 0 {
		return nil, errors.New("--fixture and --repeats between 1 and 20 are required; positional arguments are forbidden")
	}
	b, err := readBoundedRegularFile(*fixture, 1024*1024)
	if err != nil {
		return nil, err
	}
	digest := sha256.Sum256(b)
	dec := json.NewDecoder(strings.NewReader(string(b)))
	dec.DisallowUnknownFields()
	var suite workflowRoutingSuite
	if err := dec.Decode(&suite); err != nil {
		return nil, err
	}
	var trailing any
	if err := dec.Decode(&trailing); err != io.EOF {
		return nil, errors.New("workflow-routing suite contains trailing JSON")
	}
	if suite.SchemaVersion != 1 || len(suite.Cases) < 35 || len(suite.Cases) > 500 {
		return nil, errors.New("workflow-routing suite requires schema version 1 and 35 to 500 cases")
	}
	seenIDs, seenInputs, labelCounts := map[string]bool{}, map[string]bool{}, map[string]int{}
	for index := range suite.Cases {
		item := &suite.Cases[index]
		item.ID, item.Request, item.Files, item.Workflow = normalizeFixtureText(item.ID), normalizeFixtureText(item.Request), normalizeFixtureText(item.Files), normalizeFixtureText(item.Workflow)
		fingerprint := item.Request + "\x00" + item.Files
		if item.ID == "" || item.Request == "" || !validClassifiedWorkflow(item.Workflow) || seenIDs[item.ID] || seenInputs[fingerprint] {
			return nil, errors.New("workflow-routing cases require unique IDs and inputs with supported workflow labels")
		}
		seenIDs[item.ID], seenInputs[fingerprint] = true, true
		labelCounts[item.Workflow]++
	}
	for _, workflow := range []string{"release-readiness", "feature", "bug-fix", "bug-diagnosis", "pr-review", "trivial-change", "incident"} {
		if labelCounts[workflow] < 5 {
			return nil, fmt.Errorf("workflow %q requires at least five labelled cases", workflow)
		}
	}
	scores := map[string]*workflowScore{}
	for workflow := range labelCounts {
		scores[workflow] = &workflowScore{}
	}
	failures, failureCount, correct := []string{}, 0, 0
	for _, item := range suite.Cases {
		for trial := 1; trial <= *repeats; trial++ {
			got, classifyErr := classify([]string{"--request", item.Request, "--files", item.Files})
			if classifyErr != nil {
				return nil, classifyErr
			}
			actual := got["workflow"].(string)
			if actual == item.Workflow {
				correct++
				scores[item.Workflow].TruePositive++
				continue
			}
			failureCount++
			scores[item.Workflow].FalseNegative++
			scores[actual].FalsePositive++
			if len(failures) < 50 {
				failures = append(failures, fmt.Sprintf("%s trial %d: got %s, want %s", item.ID, trial, actual, item.Workflow))
			}
		}
	}
	perWorkflow, passed := result{}, true
	for workflow, score := range scores {
		precision := ratio(score.TruePositive, score.TruePositive+score.FalsePositive)
		recall := ratio(score.TruePositive, score.TruePositive+score.FalseNegative)
		workflowPassed := precision >= .95 && recall >= .95
		passed = passed && workflowPassed
		perWorkflow[workflow] = result{"truePositive": score.TruePositive, "falsePositive": score.FalsePositive, "falseNegative": score.FalseNegative, "precision": precision, "recall": recall, "passed": workflowPassed}
	}
	trials := len(suite.Cases) * *repeats
	return result{"evaluationSchemaVersion": 1, "evaluatedAt": time.Now().UTC().Format(time.RFC3339Nano), "fixtureDigest": hex.EncodeToString(digest[:]), "labelledRequests": len(suite.Cases), "repeats": *repeats, "trials": trials, "accuracy": ratio(correct, trials), "perWorkflow": perWorkflow, "passed": passed, "thresholdValues": result{"minimumCasesPerWorkflow": 5, "minimumPrecision": .95, "minimumRecall": .95}, "failureCount": failureCount, "failuresTruncated": failureCount > len(failures), "failures": failures, "scope": "DETERMINISTIC_MULTICLASS_ROUTER_ONLY_NOT_IMPLICIT_SKILL_SELECTION"}, nil
}
