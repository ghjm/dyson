package dyson

import (
	"fmt"
	"gopkg.in/yaml.v3"
)

type DataFile struct {
	Facilities    map[string]map[string]float32 `yaml:"facilities"`
	Processes     []Process                     `yaml:"processes"`
	procsByTarget map[string][]Process          `yaml:"-"`
}

type Process struct {
	Makes    map[string]int `yaml:"makes"`
	Consumes map[string]int `yaml:"consumes"`
	Time     float32        `yaml:"time"`
	Facility []string       `yaml:"facility"`
	Special  bool           `yaml:"special"`
}

func LoadData(data []byte) (*DataFile, error) {
	var df DataFile
	err := yaml.Unmarshal(data, &df)
	if err != nil {
		return nil, fmt.Errorf("could not parse data: %w", err)
	}
	df.procsByTarget = make(map[string][]Process)
	for _, proc := range df.Processes {
		for m := range proc.Makes {
			df.procsByTarget[m] = append(df.procsByTarget[m], proc)
		}
	}
	return &df, nil
}

func (df *DataFile) Makeable(item string) bool {
	for _, proc := range df.procsByTarget[item] {
		result := true
		for c := range proc.Consumes {
			if !df.Makeable(c) {
				result = false
				break
			}
		}
		if result {
			return true
		}
	}
	return false
}

func (df *DataFile) Validate() error {

	// Check for duplicate facilities / facility types
	facTypeMap := make(map[string]struct{})
	for facType, facs := range df.Facilities {
		_, ok := facTypeMap[facType]
		if ok {
			return fmt.Errorf("duplicate facility type: %s", facType)
		}
		facTypeMap[facType] = struct{}{}
		facMap := make(map[string]struct{})
		for fac, rate := range facs {
			_, ok := facMap[fac]
			if ok {
				return fmt.Errorf("duplicate facility: %s", fac)
			}
			if rate == 0 {
				return fmt.Errorf("facility rate is zero: %s", fac)
			}
			facMap[fac] = struct{}{}
		}
	}

	// Make sure every mentioned item is either a resource or makeable
	items := make(map[string]struct{})
	for _, process := range df.Processes {
		for m := range process.Makes {
			items[m] = struct{}{}
		}
		for c := range process.Consumes {
			items[c] = struct{}{}
		}
	}
	for item := range items {
		if !df.Makeable(item) {
			return fmt.Errorf("item cannot be made: %s", item)
		}
	}
	return nil
}
