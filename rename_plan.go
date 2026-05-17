package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
)

type renamePlan struct {
	from string
	to   string
	temp string
}

func (fr *findReplace) queueRename(from, to string) {
	fr.renameMu.Lock()
	fr.renames = append(fr.renames, renamePlan{from: from, to: to})
	fr.renameMu.Unlock()
}

func (fr *findReplace) applyRenames() error {
	if len(fr.renames) == 0 {
		return nil
	}

	active := make([]renamePlan, 0, len(fr.renames))
	sources := make(map[string]bool, len(fr.renames))
	targets := make(map[string]bool, len(fr.renames))
	for _, r := range fr.renames {
		if r.from == r.to {
			continue
		}
		if targets[r.to] {
			return fmt.Errorf("duplicate rename target %q", r.to)
		}
		active = append(active, r)
		sources[r.from] = true
		targets[r.to] = true
	}

	if len(active) == 0 {
		return nil
	}

	for to := range targets {
		if sources[to] {
			continue
		}
		if _, err := os.Stat(to); err == nil {
			return fmt.Errorf("refusing rename: %q already exists", to)
		}
	}

	needsTwoPhase := false
	for _, r := range active {
		if sources[r.to] {
			needsTwoPhase = true
			break
		}
	}

	sort.Slice(active, func(i, j int) bool {
		return len(active[i].from) > len(active[j].from)
	})

	if needsTwoPhase {
		return fr.applyRenamesTwoPhase(active)
	}
	return fr.applyRenamesDirect(active)
}

func (fr *findReplace) applyRenamesDirect(plans []renamePlan) error {
	for _, r := range plans {
		log.Printf("Renaming %v to %v", r.from, filepath.Base(r.to))
		if err := os.Rename(r.from, r.to); err != nil {
			return fmt.Errorf("rename %q -> %q: %w", r.from, r.to, err)
		}
	}
	return nil
}

func (fr *findReplace) applyRenamesTwoPhase(plans []renamePlan) error {
	rollback := func() {
		for _, r := range plans {
			if r.temp != "" {
				_ = os.Remove(r.temp)
			}
		}
	}

	seq := 0
	for i := range plans {
		r := &plans[i]
		r.temp = filepath.Join(filepath.Dir(r.from), fmt.Sprintf(".find-replace.tmp.%d.%d", os.Getpid(), seq))
		seq++
		log.Printf("Renaming %v to %v (phase 1)", r.from, filepath.Base(r.temp))
		if err := os.Rename(r.from, r.temp); err != nil {
			rollback()
			return fmt.Errorf("phase 1 rename %q: %w", r.from, err)
		}
	}

	phase2 := append([]renamePlan(nil), plans...)
	sort.Slice(phase2, func(i, j int) bool {
		return len(phase2[i].to) > len(phase2[j].to)
	})

	for _, r := range phase2 {
		log.Printf("Renaming %v to %v (phase 2)", filepath.Base(r.temp), filepath.Base(r.to))
		if err := os.Rename(r.temp, r.to); err != nil {
			return fmt.Errorf("phase 2 rename %q -> %q: %w", r.temp, r.to, err)
		}
	}
	return nil
}
