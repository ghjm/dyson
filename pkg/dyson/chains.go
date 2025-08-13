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
	Rate    float32
}

type StringOptions struct {
	converterFunc StringUnitConverterFunc
}

type StringOption func(*StringOptions)

type StringUnitConverterFunc func(item string, rate float32) (bool, float32, string)

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

func (pc *ProductionChain) SetRate(item string, rate float32) error {
	for i := range pc.Steps {
		if pc.Steps[i].Target == item {
			pc.Steps[i].Rate = rate
			return nil
		}
	}
	return fmt.Errorf("item not found in chain: %s", item)
}

func (pc *ProductionChain) fillOneChain(n int) error {
	return pc.fillOneChainExcluding(n, nil)
}

func (pc *ProductionChain) fillOneChainExcluding(n int, excluded map[string]struct{}) error {
	ps := &pc.Steps[n]
	if ps.Process != nil {
		return fmt.Errorf("chain already filled")
	}
	found := false
	for _, proc := range pc.df.procsByTarget[ps.Target] {
		if !proc.Special {
			found = true
			ps.Process = &proc

			var runsPerSecond float32
			itemsPerRun := float32(proc.Makes[ps.Target])
			if itemsPerRun > 0 {
				runsPerSecond = ps.Rate / itemsPerRun
			}

			for _, con := range slices.Sorted(maps.Keys(proc.Consumes)) {
				// Skip excluded items
				if excluded != nil {
					if _, isExcluded := excluded[con]; isExcluded {
						continue
					}
				}

				consumedAmount := float32(proc.Consumes[con])
				requiredRate := runsPerSecond * consumedAmount

				alreadyHave := false
				for i := range pc.Steps {
					if pc.Steps[i].Target == con {
						// Add to existing rate (accumulate demand from multiple consumers)
						pc.Steps[i].Rate += requiredRate
						alreadyHave = true
						break
					}
				}
				if !alreadyHave {
					pc.Steps = append(pc.Steps, ProductionStep{
						Target: con,
						Rate:   requiredRate,
					})
				}
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
	return pc.FillChainExcluding(nil)
}

func (pc *ProductionChain) FillChainExcluding(exclusions []string) error {
	excluded := make(map[string]struct{})
	for _, ex := range exclusions {
		excluded[ex] = struct{}{}
	}

	var complete bool
	for !complete {
		complete = true
		for i, step := range pc.Steps {
			if step.Process == nil {
				complete = false
				err := pc.fillOneChainExcluding(i, excluded)
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
	return pc.StringWithOpts()
}

func (pc *ProductionChain) StringWithOpts(opts ...StringOption) string {
	sb := strings.Builder{}
	for _, step := range pc.Steps {
		sb.WriteString(step.StringWithOpts(opts...))
		sb.WriteString("\n")
	}
	return sb.String()
}

func (ps *ProductionStep) String() string {
	return ps.StringWithOpts()
}

func WithUnitConverter(conv StringUnitConverterFunc) func(options *StringOptions) {
	return func(options *StringOptions) {
		options.converterFunc = conv
	}
}

func (ps *ProductionStep) StringWithOpts(opts ...StringOption) string {
	so := StringOptions{}
	for _, opt := range opts {
		opt(&so)
	}
	sb := strings.Builder{}
	rr := ""
	if ps.Rate > 0 {
		rate := ps.Rate
		suffix := "/s"
		if so.converterFunc != nil {
			convert, newRate, newSuffix := so.converterFunc(ps.Target, rate)
			if convert {
				rate, suffix = newRate, newSuffix
			}
		}
		// Format without scientific notation and remove trailing zeros
		rr = fmt.Sprintf(" (%s%s)", strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.3f", rate), "0"), "."), suffix)
	}

	sb.WriteString(fmt.Sprintf("%s%s: ", ps.Target, rr))

	if ps.Process == nil {
		sb.WriteString("<unknown>")
	} else {
		reqs := slices.Sorted(maps.Keys(ps.Process.Consumes))
		if len(reqs) > 0 {
			sb.WriteString(strings.Join(reqs, ", "))
		} else {
			sb.WriteString(fmt.Sprintf("<produced by %s>", strings.Join(ps.Process.Facility, " or ")))
		}
	}
	return sb.String()
}
