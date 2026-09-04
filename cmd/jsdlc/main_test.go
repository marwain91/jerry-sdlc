package main

import "testing"

func TestClassifyRelease(t *testing.T) {
	got, err := classify([]string{"--request", "Prepare this project for release"})
	if err != nil {
		t.Fatal(err)
	}
	if got["workflow"] != "release-readiness" || got["risk"] != "HIGH" {
		t.Fatalf("unexpected classification: %#v", got)
	}
}

func TestClassifyMigration(t *testing.T) {
	got, err := classify([]string{"--request", "implement account export", "--files", "db/042.sql"})
	if err != nil {
		t.Fatal(err)
	}
	if got["risk"] != "HIGH" {
		t.Fatalf("expected HIGH risk: %#v", got)
	}
	triggers := got["triggers"].([]string)
	if len(triggers) == 0 || triggers[0] != "data-migration" {
		t.Fatalf("expected migration trigger: %#v", got)
	}
}

func TestReleaseRoles(t *testing.T) {
	got, err := roles([]string{"--workflow", "release-readiness"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got["roles"].([]string)) != 5 {
		t.Fatalf("unexpected roles: %#v", got)
	}
}
