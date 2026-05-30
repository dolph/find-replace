package main

import (
	"errors"
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

	needsTwoPhase := false
	for _, r := range active {
		if sources[r.to] {
			needsTwoPhase = true
			break
		}
	}

	sort.Slice(active, func(i, j int) bool {
		if len(active[i].from) != len(active[j].from) {
			return len(active[i].from) > len(active[j].from)
		}
		return active[i].from < active[j].from
	})

	if needsTwoPhase {
		return fr.applyRenamesTwoPhase(active)
	}
	return fr.applyRenamesDirect(active)
}

func (fr *findReplace) applyRenamesDirect(plans []renamePlan) error {
	var errs []error
	for _, r := range plans {
		if err := fr.renamePath(r.from, r.to); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
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
		if len(phase2[i].to) != len(phase2[j].to) {
			return len(phase2[i].to) > len(phase2[j].to)
		}
		return phase2[i].to < phase2[j].to
	})

	for _, r := range phase2 {
		log.Printf("Renaming %v to %v (phase 2)", filepath.Base(r.temp), filepath.Base(r.to))
		if err := os.Rename(r.temp, r.to); err != nil {
			return fmt.Errorf("phase 2 rename %q -> %q: %w", r.temp, r.to, err)
		}
	}
	return nil
}

func (fr *findReplace) renamePath(from, to string) error {
	if _, err := os.Stat(to); err == nil {
		return fmt.Errorf("refusing to rename %v to %v: %v already exists", from, filepath.Base(to), to)
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("stat rename destination %v: %w", to, err)
	}

	log.Printf("Renaming %v to %v", from, filepath.Base(to))
	if err := os.Rename(from, to); err != nil {
		return fmt.Errorf("rename %v to %v: %w", from, filepath.Base(to), err)
	}
	return nil
}
