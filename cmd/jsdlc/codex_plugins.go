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
	"sort"
	"strings"
)

type codexPluginInventory struct {
	Installed []codexPlugin     `json:"installed"`
	Available []json.RawMessage `json:"available"`
}

type codexPlugin struct {
	PluginID          string          `json:"pluginId"`
	Name              string          `json:"name"`
	MarketplaceName   string          `json:"marketplaceName"`
	Version           string          `json:"version"`
	Installed         bool            `json:"installed"`
	Enabled           bool            `json:"enabled"`
	Source            json.RawMessage `json:"source"`
	MarketplaceSource json.RawMessage `json:"marketplaceSource,omitempty"`
	InstallPolicy     string          `json:"installPolicy"`
	AuthPolicy        string          `json:"authPolicy"`
}

type coexistenceRegistry struct {
	SchemaVersion              int      `json:"schemaVersion"`
	JerryPluginID              string   `json:"jerryPluginId"`
	BroadOrchestratorPluginIDs []string `json:"broadOrchestratorPluginIds"`
}

func inspectCodexPlugins(args []string) (result, error) {
	fs := flag.NewFlagSet("inspect-codex-plugins", flag.ContinueOnError)
	inventoryPath := fs.String("file", "", "captured output from codex plugin list --json")
	registryPath := fs.String("registry", "", "exact-ID coexistence registry (defaults to bundled registry)")
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	if *inventoryPath == "" || fs.NArg() != 0 {
		return nil, errors.New("--file is required; positional arguments are forbidden")
	}
	if *registryPath == "" {
		*registryPath = filepath.Join(os.Getenv("JSDLC_PLUGIN_ROOT"), "coexistence", "catalog.json")
	}
	inventoryBytes, err := readBoundedRegularFile(*inventoryPath, 1024*1024)
	if err != nil {
		return nil, err
	}
	registryBytes, err := readBoundedRegularFile(*registryPath, 64*1024)
	if err != nil {
		return nil, err
	}
	var inventory codexPluginInventory
	if err := decodeStrictJSON(inventoryBytes, &inventory, "Codex plugin inventory"); err != nil {
		return nil, err
	}
	var registry coexistenceRegistry
	if err := decodeStrictJSON(registryBytes, &registry, "coexistence registry"); err != nil {
		return nil, err
	}
	if err := validateCoexistenceRegistry(registry); err != nil {
		return nil, err
	}
	if inventory.Installed == nil || inventory.Available == nil || len(inventory.Installed) > 500 {
		return nil, errors.New("Codex plugin inventory is invalid or too large")
	}
	known := map[string]bool{}
	for _, id := range registry.BroadOrchestratorPluginIDs {
		known[id] = true
	}
	seen := map[string]bool{}
	active, competitors, unclassified := []string{}, []string{}, []string{}
	jerryDetected := false
	for _, plugin := range inventory.Installed {
		if err := validateInventoryPlugin(plugin); err != nil {
			return nil, err
		}
		key := strings.ToLower(plugin.PluginID)
		if seen[key] {
			return nil, errors.New("Codex plugin inventory contains duplicate plugin IDs")
		}
		seen[key] = true
		if !plugin.Installed || !plugin.Enabled {
			continue
		}
		active = append(active, plugin.PluginID)
		if plugin.PluginID == registry.JerryPluginID {
			jerryDetected = true
			continue
		}
		if known[plugin.PluginID] {
			competitors = append(competitors, plugin.PluginID)
		} else {
			unclassified = append(unclassified, plugin.PluginID)
		}
	}
	if !jerryDetected {
		return nil, errors.New("active Jerry plugin ID is absent from the Codex plugin inventory")
	}
	sort.Strings(active)
	sort.Strings(competitors)
	sort.Strings(unclassified)
	invDigest, regDigest := sha256.Sum256(inventoryBytes), sha256.Sum256(registryBytes)
	return result{
		"schemaVersion": 1, "discoveryEvidence": "CODEX_PLUGIN_LIST", "classificationRule": "EXACT_REGISTRY_IDS_ONLY",
		"activePluginIds": active, "competingBroadOrchestrators": competitors, "unclassifiedPluginIds": unclassified,
		"inventoryDigest": hex.EncodeToString(invDigest[:]), "registryDigest": hex.EncodeToString(regDigest[:]),
		"jerryPluginDetected": true, "sourcePathsOpened": false, "completeSemanticDiscovery": false,
	}, nil
}

func decodeStrictJSON(b []byte, target any, label string) error {
	dec := json.NewDecoder(strings.NewReader(string(b)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(target); err != nil {
		return err
	}
	var trailing any
	if err := dec.Decode(&trailing); err != io.EOF {
		return errors.New(label + " contains trailing JSON")
	}
	return nil
}

func validateCoexistenceRegistry(r coexistenceRegistry) error {
	if r.SchemaVersion != 1 || !validPluginID(r.JerryPluginID) || r.BroadOrchestratorPluginIDs == nil || len(r.BroadOrchestratorPluginIDs) > 100 {
		return errors.New("coexistence registry is invalid")
	}
	seen := map[string]bool{strings.ToLower(r.JerryPluginID): true}
	for _, id := range r.BroadOrchestratorPluginIDs {
		key := strings.ToLower(id)
		if !validPluginID(id) || seen[key] {
			return errors.New("coexistence registry contains an invalid, reserved, or duplicate plugin ID")
		}
		seen[key] = true
	}
	return nil
}

func validateInventoryPlugin(p codexPlugin) error {
	if !validPluginID(p.PluginID) || p.Name == "" || p.MarketplaceName == "" || p.Version == "" || len(p.Name) > 128 || len(p.MarketplaceName) > 128 || len(p.Version) > 256 || len(p.Source) == 0 || p.InstallPolicy == "" || p.AuthPolicy == "" {
		return errors.New("Codex plugin inventory contains an invalid plugin record")
	}
	if p.PluginID != p.Name+"@"+p.MarketplaceName {
		return errors.New("Codex plugin inventory plugin identity is inconsistent")
	}
	return nil
}

func validPluginID(id string) bool {
	if id == "" || id != strings.ToLower(id) || len(id) > 260 || strings.TrimSpace(id) != id || strings.Count(id, "@") != 1 {
		return false
	}
	parts := strings.SplitN(id, "@", 2)
	for _, part := range parts {
		if part == "" {
			return false
		}
		for _, ch := range part {
			if !(ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z' || ch >= '0' && ch <= '9' || ch == '-' || ch == '_' || ch == '.') {
				return false
			}
		}
	}
	return true
}
