package procedure

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Entry is one indexed procedure file (repo-relative path and parsed mapping fields).
//
//nolint:govet // fieldalignment: keep id/path grouping for readability
type Entry struct {
	ID        string
	Path      string
	StateIDs  []string
	RouteIDs  []string
	Purpose   string
	AbsSource string
}

// LoadEntries scans procedureAbsDir for *.md, parses front matter, and returns sorted entries.
// If procedureAbsDir does not exist, returns (nil, false, nil).
// If the directory exists but contains no procedure .md files, returns (empty slice, true, nil).
func LoadEntries(repoRootAbs, procedureAbsDir string) ([]Entry, bool, error) {
	repoRootAbs = filepath.Clean(repoRootAbs)
	procedureAbsDir = filepath.Clean(procedureAbsDir)
	if _, statErr := os.Stat(procedureAbsDir); statErr != nil {
		if os.IsNotExist(statErr) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("procedure: stat %s: %w", procedureAbsDir, statErr)
	}
	de, err := os.ReadDir(procedureAbsDir)
	if err != nil {
		return nil, true, fmt.Errorf("procedure: read directory: %w", err)
	}
	var names []string
	for _, e := range de {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.EqualFold(filepath.Ext(name), ".md") {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)

	seenProcID := make(map[string]string)
	var entries []Entry
	for _, name := range names {
		absPath := filepath.Join(procedureAbsDir, name)
		data, rerr := os.ReadFile(absPath)
		if rerr != nil {
			return nil, true, fmt.Errorf("procedure: read %s: %w", absPath, rerr)
		}
		fm, perr := ParseFrontMatter(data)
		if perr != nil {
			return nil, true, fmt.Errorf("%s: %w", absPath, perr)
		}
		if prev, ok := seenProcID[fm.ID]; ok {
			return nil, true, fmt.Errorf("procedure: duplicate procedure id %q in %s and %s", fm.ID, prev, absPath)
		}
		seenProcID[fm.ID] = absPath

		rel, rerr := filepath.Rel(repoRootAbs, absPath)
		if rerr != nil {
			return nil, true, fmt.Errorf("procedure: rel path: %w", rerr)
		}
		rel = filepath.ToSlash(rel)

		entries = append(entries, Entry{
			ID:        fm.ID,
			Path:      rel,
			StateIDs:  append([]string(nil), fm.AppliesTo.StateIDs...),
			RouteIDs:  append([]string(nil), fm.AppliesTo.RouteIDs...),
			Purpose:   fm.Purpose,
			AbsSource: absPath,
		})
	}
	return entries, true, nil
}

// ValidateStateMapping checks that each state_id appears in at most one procedure entry
// and that every mapped state_id exists in declaredStateIDs (from control state rules).
func ValidateStateMapping(entries []Entry, declaredStateIDs map[string]struct{}) error {
	if len(entries) == 0 {
		return nil
	}
	stateOwner := make(map[string]string) // state_id -> procedure id
	for _, e := range entries {
		for _, sid := range e.StateIDs {
			if prev, ok := stateOwner[sid]; ok {
				return fmt.Errorf(
					"procedure: state_id %q is mapped by both %q and %q",
					sid, prev, e.ID,
				)
			}
			stateOwner[sid] = e.ID
		}
	}
	for sid, procID := range stateOwner {
		if _, ok := declaredStateIDs[sid]; !ok {
			return fmt.Errorf(
				"procedure: state_id %q in procedure %q has no matching control state rule",
				sid, procID,
			)
		}
	}
	return nil
}

func containsString(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

// HintEntry returns the procedure mapped to stateID, optionally filtered by routeID
// when the entry declares non-empty RouteIDs (must match a resolved route).
func HintEntry(entries []Entry, stateID string, routeResolved bool, routeID string) *Entry {
	for i := range entries {
		e := &entries[i]
		if !containsString(e.StateIDs, stateID) {
			continue
		}
		if len(e.RouteIDs) > 0 {
			if !routeResolved || routeID == "" || !containsString(e.RouteIDs, routeID) {
				continue
			}
		}
		return e
	}
	return nil
}
