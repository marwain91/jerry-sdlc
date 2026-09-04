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

type triggerSuite struct {
	SchemaVersion int           `json:"schemaVersion"`
	Subjects      []string      `json:"subjects"`
	Cases         []triggerCase `json:"cases"`
}

type triggerCase struct {
	ID       string `json:"id"`
	Template string `json:"template"`
	Release  bool   `json:"release"`
}

type expandedTrigger struct {
	id      string
	subject string
	prompt  string
	release bool
}

func evalTriggers(args []string) (result, error) {
	fs := flag.NewFlagSet("eval-triggers", flag.ContinueOnError)
	fixture := fs.String("fixture", "", "trigger-suite JSON file")
	repeats := fs.Int("repeats", 3, "trials per labelled prompt")
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	if *fixture == "" || *repeats < 1 || *repeats > 20 {
		return nil, errors.New("--fixture and --repeats between 1 and 20 are required")
	}
	b, err := readBoundedRegularFile(*fixture, 1024*1024)
	if err != nil {
		return nil, err
	}
	fixtureHash := sha256.Sum256(b)
	dec := json.NewDecoder(strings.NewReader(string(b)))
	dec.DisallowUnknownFields()
	var suite triggerSuite
	if err := dec.Decode(&suite); err != nil {
		return nil, err
	}
	var trailing any
	if err := dec.Decode(&trailing); err != io.EOF {
		return nil, errors.New("trigger suite contains trailing JSON")
	}
	if suite.SchemaVersion != 1 || len(suite.Subjects) == 0 || len(suite.Cases) == 0 {
		return nil, errors.New("invalid or empty trigger suite")
	}
	seenSubjects, seenCases, seenTemplates := map[string]bool{}, map[string]bool{}, map[string]bool{}
	normalizedSubjects := make([]string, 0, len(suite.Subjects))
	for _, rawSubject := range suite.Subjects {
		subject := normalizeFixtureText(rawSubject)
		if subject == "" || seenSubjects[subject] {
			return nil, errors.New("trigger suite contains blank or duplicate subjects")
		}
		seenSubjects[subject] = true
		normalizedSubjects = append(normalizedSubjects, subject)
	}
	expanded := make([]expandedTrigger, 0, len(suite.Cases)*len(normalizedSubjects))
	seenPrompts := map[string]bool{}
	positive, negative := 0, 0
	for _, rawCase := range suite.Cases {
		id, template := normalizeFixtureText(rawCase.ID), normalizeFixtureText(rawCase.Template)
		if id == "" || template == "" || seenCases[id] {
			return nil, errors.New("trigger suite contains blank or duplicate cases")
		}
		if !strings.Contains(template, "{project}") || seenTemplates[template] {
			return nil, errors.New("trigger suite contains a duplicate template or one without {project}")
		}
		seenCases[id], seenTemplates[template] = true, true
		for _, subject := range normalizedSubjects {
			prompt := normalizeFixtureText(strings.ReplaceAll(template, "{project}", subject))
			if seenPrompts[prompt] {
				return nil, errors.New("trigger suite expands to duplicate prompts")
			}
			seenPrompts[prompt] = true
			expanded = append(expanded, expandedTrigger{id: id, subject: subject, prompt: prompt, release: rawCase.Release})
			if rawCase.Release {
				positive++
			} else {
				negative++
			}
		}
	}
	if positive == 0 || negative == 0 {
		return nil, errors.New("trigger suite must contain positive and negative labels")
	}
	tp, fp, tn, fn := 0, 0, 0, 0
	failures := []string{}
	failureCount := 0
	labels := len(expanded)
	for _, item := range expanded {
		for trial := 0; trial < *repeats; trial++ {
			classified, classifyErr := classify([]string{"--request", item.prompt})
			if classifyErr != nil {
				return nil, classifyErr
			}
			got := classified["workflow"] == "release-readiness"
			switch {
			case item.release && got:
				tp++
			case item.release:
				fn++
			case got:
				fp++
			default:
				tn++
			}
			if got != item.release && len(failures) < 50 {
				failures = append(failures, fmt.Sprintf("%s/%s trial %d", item.id, item.subject, trial+1))
			}
			if got != item.release {
				failureCount++
			}
		}
	}
	precision, recall := ratio(tp, tp+fp), ratio(tp, tp+fn)
	minimum, precisionPass, recallPass := labels >= 100, precision >= .98, recall >= .95
	return result{"evaluationSchemaVersion": 1, "evaluatedAt": time.Now().UTC().Format(time.RFC3339Nano), "fixtureDigest": hex.EncodeToString(fixtureHash[:]), "suiteSchemaVersion": suite.SchemaVersion, "labelledPrompts": labels, "repeats": *repeats, "trials": labels * *repeats, "truePositive": tp, "falsePositive": fp, "trueNegative": tn, "falseNegative": fn, "precision": precision, "recall": recall, "passed": minimum && precisionPass && recallPass, "thresholds": result{"minimumLabelledPrompts": minimum, "precisionAtLeast98Percent": precisionPass, "recallAtLeast95Percent": recallPass}, "failureCount": failureCount, "failuresTruncated": failureCount > len(failures), "failures": failures, "scope": "DETERMINISTIC_CLASSIFIER_ONLY_NOT_IMPLICIT_SKILL_SELECTION"}, nil
}

func normalizeFixtureText(value string) string {
	return strings.ToLower(strings.Join(strings.Fields(value), " "))
}

func ratio(numerator, denominator int) float64 {
	if denominator == 0 {
		return 0
	}
	return float64(numerator) / float64(denominator)
}
