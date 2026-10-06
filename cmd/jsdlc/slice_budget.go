package main

import (
	"errors"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type modelUsage struct {
	Input  int64 `json:"inputTokens"`
	Cached int64 `json:"cachedInputTokens"`
	Output int64 `json:"outputTokens"`
}

func validModelUsage(u modelUsage) bool {
	return u.Input >= 0 && u.Input <= 1_000_000_000_000 && u.Cached >= 0 && u.Cached <= u.Input && u.Output >= 0 && u.Output <= 1_000_000_000_000
}

type sliceEvent struct {
	ID              string      `json:"id"`
	Kind            string      `json:"kind"`
	Evidence        string      `json:"evidence"`
	MaterialBlocker string      `json:"materialBlocker,omitempty"`
	Usage           *modelUsage `json:"usage,omitempty"`
	Coverage        string      `json:"tokenCoverage,omitempty"`
	ActiveSeconds   *int64      `json:"activeSeconds,omitempty"`
	WaitSeconds     *int64      `json:"waitSeconds,omitempty"`
	HumanSeconds    *int64      `json:"humanSeconds,omitempty"`
	FindingOutcome  string      `json:"findingOutcome,omitempty"`
	Severity        string      `json:"severity,omitempty"`
	EscapedDefects  *int64      `json:"escapedDefects,omitempty"`
	ReopenedTasks   *int64      `json:"reopenedTasks,omitempty"`
}
type sliceLedger struct {
	SchemaVersion      int          `json:"schemaVersion"`
	ID                 string       `json:"id"`
	MaxReviews         int          `json:"maxReviews"`
	MaxActiveSeconds   int64        `json:"maxActiveSeconds"`
	MaxAccountedTokens int64        `json:"maxUncachedInputPlusOutputTokens"`
	CreatedAt          string       `json:"createdAt"`
	Events             []sliceEvent `json:"events"`
}

func validateSliceEvent(e sliceEvent) error {
	if !validEvidenceID(e.ID) || !contains([]string{"review", "measurement", "finding", "checkpoint"}, e.Kind) || strings.TrimSpace(e.Evidence) == "" || len(e.Evidence) > 4096 || len(e.MaterialBlocker) > 4096 || e.MaterialBlocker != "" && e.Kind != "review" {
		return errors.New("invalid slice event identity, evidence, or kind")
	}
	if e.Usage != nil && (!validModelUsage(*e.Usage) || !contains([]string{"ALL_WORKERS", "PARTIAL"}, e.Coverage)) || e.Usage == nil && e.Coverage != "" {
		return errors.New("token usage needs valid counters and explicit worker coverage")
	}
	for _, v := range []*int64{e.ActiveSeconds, e.WaitSeconds, e.HumanSeconds, e.EscapedDefects, e.ReopenedTasks} {
		if v != nil && (*v < 0 || *v > 1_000_000_000) {
			return errors.New("slice measurements must be bounded and nonnegative")
		}
	}
	if e.Kind == "finding" {
		if !contains([]string{"CONFIRMED", "REJECTED", "UNRESOLVED"}, e.FindingOutcome) || severityWeight(e.Severity) == 0 {
			return errors.New("findings require adjudication status and severity")
		}
	} else if e.FindingOutcome != "" || e.Severity != "" {
		return errors.New("finding fields require a finding event")
	}
	return nil
}

func validateSliceLedger(l sliceLedger) error {
	if l.SchemaVersion != 1 || !validEvidenceID(l.ID) || l.MaxReviews < 1 || l.MaxReviews > 10 || l.MaxActiveSeconds <= 0 || l.MaxActiveSeconds > 7*24*3600 || l.MaxAccountedTokens <= 0 || l.MaxAccountedTokens > 1_000_000_000 || parseRFC3339(l.CreatedAt) != nil || l.Events == nil || len(l.Events) > 1000 {
		return errors.New("invalid slice budget ledger")
	}
	seen := map[string]bool{}
	reviews := 0
	for _, e := range l.Events {
		if err := validateSliceEvent(e); err != nil {
			return err
		}
		if seen[e.ID] {
			return errors.New("duplicate slice event ID")
		}
		seen[e.ID] = true
		if e.Kind == "review" {
			reviews++
			if reviews > l.MaxReviews+1 || reviews > l.MaxReviews && strings.TrimSpace(e.MaterialBlocker) == "" {
				return errors.New("review limit exceeded without a bounded material blocker extension")
			}
		}
	}
	return nil
}

func sliceAssessment(l sliceLedger) result {
	var gross, cached, accounted, active, wait, human, escaped, reopened int64
	reviews, confirmed, rejected, unresolved := 0, 0, 0, 0
	usageKnown, activeKnown, waitKnown, humanKnown, escapedKnown, reopenedKnown := false, false, false, false, false, false
	coverage := "UNKNOWN"
	for _, e := range l.Events {
		if e.Kind == "review" {
			reviews++
		}
		if e.Usage != nil {
			usageKnown = true
			gross += e.Usage.Input + e.Usage.Output
			cached += e.Usage.Cached
			accounted += e.Usage.Input - e.Usage.Cached + e.Usage.Output
			if coverage != "PARTIAL" {
				coverage = e.Coverage
			}
		}
		if e.ActiveSeconds != nil {
			activeKnown = true
			active += *e.ActiveSeconds
		}
		if e.WaitSeconds != nil {
			waitKnown = true
			wait += *e.WaitSeconds
		}
		if e.HumanSeconds != nil {
			humanKnown = true
			human += *e.HumanSeconds
		}
		if e.EscapedDefects != nil {
			escapedKnown = true
			escaped += *e.EscapedDefects
		}
		if e.ReopenedTasks != nil {
			reopenedKnown = true
			reopened += *e.ReopenedTasks
		}
		switch e.FindingOutcome {
		case "CONFIRMED":
			confirmed++
		case "REJECTED":
			rejected++
		case "UNRESOLVED":
			unresolved++
		}
	}
	outcome := "WITHIN_RECORDED_LIMITS"
	if reviews >= l.MaxReviews || accounted >= l.MaxAccountedTokens || active >= l.MaxActiveSeconds {
		outcome = "CHECKPOINT_REQUIRED"
	}
	optional := func(known bool, value int64) any {
		if !known {
			return nil
		}
		return value
	}
	return result{"outcome": outcome, "slice": l.ID, "reviewRounds": reviews, "maxReviewRounds": l.MaxReviews, "materialBlockerExtensionUsed": reviews > l.MaxReviews, "grossTokens": optional(usageKnown, gross), "cachedInputTokens": optional(usageKnown, cached), "uncachedInputPlusOutputTokens": optional(usageKnown, accounted), "tokenCoverage": coverage, "activeSeconds": optional(activeKnown, active), "waitSeconds": optional(waitKnown, wait), "humanSeconds": optional(humanKnown, human), "confirmedFindings": confirmed, "rejectedFindings": rejected, "unresolvedFindings": unresolved, "escapedDefects": optional(escapedKnown, escaped), "reopenedTasks": optional(reopenedKnown, reopened), "limits": result{"activeSeconds": l.MaxActiveSeconds, "uncachedInputPlusOutputTokens": l.MaxAccountedTokens}, "enforcement": "RECORDED_REVIEW_ACTIONS_ONLY", "assurance": "LOCAL_UNATTESTED", "releaseReadinessEffect": "NONE"}
}

func sliceBudget(name string, args []string) (result, error) {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	file := fs.String("file", "", "private ledger path outside the consuming repository")
	id := fs.String("id", "", "neutral slice ID")
	reviews := fs.Int("max-reviews", 2, "initial review plus one correction review")
	active := fs.Int64("max-active-seconds", 3600, "active execution budget per slice")
	tokens := fs.Int64("max-accounted-tokens", 500000, "uncached input plus output token budget, including workers")
	eventFile := fs.String("event", "", "sanitized event JSON file")
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	if *file == "" || fs.NArg() != 0 {
		return nil, errors.New("--file is required; positional arguments are forbidden")
	}
	path, err := filepath.Abs(*file)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, err
	}
	unlock, err := acquireLock(path)
	if err != nil {
		return nil, err
	}
	defer unlock()
	var l sliceLedger
	if name == "slice-start" {
		if _, err := os.Lstat(path); !os.IsNotExist(err) {
			return nil, errors.New("slice ledger already exists or cannot be inspected")
		}
		l = sliceLedger{1, *id, *reviews, *active, *tokens, time.Now().UTC().Format(time.RFC3339Nano), []sliceEvent{}}
		if err := validateSliceLedger(l); err != nil {
			return nil, err
		}
		if err := writeDeliveryJSON(path, l); err != nil {
			return nil, err
		}
	} else {
		b, err := readBoundedRegularFile(path, 4*1024*1024)
		if err != nil {
			return nil, err
		}
		if err := decodeDeliveryJSON(b, &l, "slice ledger"); err != nil {
			return nil, err
		}
		if err := validateSliceLedger(l); err != nil {
			return nil, err
		}
		if name == "slice-record" {
			if *eventFile == "" {
				return nil, errors.New("--event is required")
			}
			b, err := readBoundedRegularFile(*eventFile, 128*1024)
			if err != nil {
				return nil, err
			}
			var e sliceEvent
			if err := decodeDeliveryJSON(b, &e, "slice event"); err != nil {
				return nil, err
			}
			if err := validateSliceEvent(e); err != nil {
				return nil, err
			}
			previous := sliceAssessment(l)
			if e.Kind == "review" && previous["outcome"] == "CHECKPOINT_REQUIRED" {
				// One extra targeted pass can address a named material blocker.
				// Token/time exhaustion still prevents further review admission.
				if strings.TrimSpace(e.MaterialBlocker) == "" || previous["materialBlockerExtensionUsed"] == true || previous["uncachedInputPlusOutputTokens"] != nil && previous["uncachedInputPlusOutputTokens"].(int64) >= l.MaxAccountedTokens || previous["activeSeconds"] != nil && previous["activeSeconds"].(int64) >= l.MaxActiveSeconds {
					return nil, errors.New("slice requires a checkpoint; further review exceeds its recorded budget")
				}
			}
			l.Events = append(l.Events, e)
			if err := validateSliceLedger(l); err != nil {
				return nil, err
			}
			if err := writeDeliveryJSON(path, l); err != nil {
				return nil, err
			}
		}
	}
	out := sliceAssessment(l)
	out["path"] = path
	return out, nil
}
