package dyson

import (
	"fmt"
	"maps"
	"slices"
	"strings"
)

type ProductionChain struct {
	df    *DataFile
	Steps []ProductionStep
}

type ProductionStep struct {
	Target  string
	Process *Process
}

func (df *DataFile) NewChain(reqs []string) *ProductionChain {
	pc := &ProductionChain{
		df: df,
	}
	for _, r := range reqs {
		pc.Steps = append(pc.Steps, ProductionStep{
			Target: r,
		})
	}
	return pc
}

func (pc *ProductionChain) fillOneChain(n int) error {
	ps := &pc.Steps[n]
	if ps.Process != nil {
		return fmt.Errorf("chain already filled")
	}
	found := false
	for _, proc := range pc.df.procsByTarget[ps.Target] {
		if !proc.Special {
			found = true
			ps.Process = &proc
			for _, con := range slices.Sorted(maps.Keys(proc.Consumes)) {
				alreadyHave := false
				for _, s := range pc.Steps {
					if s.Target == con {
						alreadyHave = true
						break
					}
				}
				if alreadyHave {
					continue
				}
				pc.Steps = append(pc.Steps, ProductionStep{
					Target: con,
				})
			}
			break
		}
	}
	if !found {
		return fmt.Errorf("no processes found for target %s", ps.Target)
	}
	return nil
}

func (pc *ProductionChain) FillChain() error {
	var complete bool
	for !complete {
		complete = true
		for i, step := range pc.Steps {
			if step.Process == nil {
				complete = false
				err := pc.fillOneChain(i)
				if err != nil {
					return err
				}
				break
			}
		}
	}
	return nil
}

func (pc *ProductionChain) GetAllProducible() error {
	return pc.GetAllProducibleExcluding(nil)
}

func (pc *ProductionChain) GetAllProducibleExcluding(exclusions []string) error {
	excluded := make(map[string]struct{})
	for _, ex := range exclusions {
		excluded[ex] = struct{}{}
	}
	have := make(map[string]struct{})
	for _, step := range pc.Steps {
		have[step.Target] = struct{}{}
	}
	for {
		foundAny := false
		for _, proc := range pc.df.Processes {
			if len(proc.Consumes) == 0 {
				continue
			}
			exc := false
			for _, m := range slices.Sorted(maps.Keys(proc.Makes)) {
				if _, ok := excluded[m]; ok {
					exc = true
					break
				}
			}
			if exc {
				continue
			}
			needed := false
			for _, m := range slices.Sorted(maps.Keys(proc.Makes)) {
				if _, ok := have[m]; !ok {
					needed = true
					break
				}
			}
			if !needed {
				continue
			}
			makeable := true
			for _, p := range slices.Sorted(maps.Keys(proc.Consumes)) {
				if _, ok := have[p]; !ok {
					makeable = false
					break
				}
			}
			if makeable {
				foundAny = true
				for _, t := range slices.Sorted(maps.Keys(proc.Makes)) {
					have[t] = struct{}{}
					pc.Steps = append(pc.Steps, ProductionStep{
						Target:  t,
						Process: &proc,
					})
				}
			}
		}
		if !foundAny {
			break
		}
	}
	return nil
}

func (pc *ProductionChain) String() string {
	sb := strings.Builder{}
	for _, step := range pc.Steps {
		sb.WriteString(step.String())
		sb.WriteString("\n")
	}
	return sb.String()
}

func (ps *ProductionStep) String() string {
	sb := strings.Builder{}
	sb.WriteString(fmt.Sprintf("%s: ", ps.Target))
	if ps.Process == nil {
		sb.WriteString("<unknown>")
	} else {
		var reqs []string
		for _, r := range slices.Sorted(maps.Keys(ps.Process.Consumes)) {
			reqs = append(reqs, r)
		}
		if len(reqs) > 0 {
			sb.WriteString(strings.Join(reqs, ", "))
		} else {
			sb.WriteString(fmt.Sprintf("<produced by %s>", strings.Join(ps.Process.Facility, " or ")))
		}
	}
	return sb.String()
}
