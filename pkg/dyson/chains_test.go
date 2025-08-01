package dyson

import (
	"strings"
	"testing"
)

// Test data for chain testing
var chainTestYAMLData = `
facilities:
  smelter:
    Arc Smelter: 1
  assembler:
    Assembling Machine Mk. I: 1
  mine:
    Mining Machine: 1

processes:
  - makes:
      Iron Ore: 1
    time: 2
    facility: [ mine ]

  - makes:
      Copper Ore: 1
    time: 2
    facility: [ mine ]

  - makes:
      Iron Ingot: 1
    consumes:
      Iron Ore: 1
    time: 1
    facility: [ smelter ]

  - makes:
      Copper Ingot: 1
    consumes:
      Copper Ore: 1
    time: 1
    facility: [ smelter ]

  - makes:
      Gear: 1
    consumes:
      Iron Ingot: 1
    time: 1
    facility: [ assembler ]

  - makes:
      Circuit Board: 2
    consumes:
      Iron Ingot: 2
      Copper Ingot: 1
    time: 1
    facility: [ assembler ]

  - makes:
      Electric Motor: 1
    consumes:
      Iron Ore: 2
      Gear: 1
    time: 2
    facility: [ assembler ]

  - makes:
      Special Item: 1
    consumes:
      Iron Ingot: 1
    time: 1
    facility: [ assembler ]
    special: true
`

func getTestDataFile(t *testing.T) *DataFile {
	df, err := LoadData([]byte(chainTestYAMLData))
	if err != nil {
		t.Fatalf("Failed to load test data: %v", err)
	}
	return df
}

func TestDataFile_NewChain(t *testing.T) {
	df := getTestDataFile(t)

	tests := []struct {
		name string
		reqs []string
		want int // expected number of steps
	}{
		{
			name: "single requirement",
			reqs: []string{"Iron Ingot"},
			want: 1,
		},
		{
			name: "multiple requirements",
			reqs: []string{"Iron Ingot", "Copper Ingot"},
			want: 2,
		},
		{
			name: "empty requirements",
			reqs: []string{},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pc := df.NewChain(tt.reqs)
			if pc == nil {
				t.Error("NewChain() returned nil")
				return
			}
			if pc.df != df {
				t.Error("NewChain() did not set DataFile reference correctly")
			}
			if len(pc.Steps) != tt.want {
				t.Errorf("NewChain() created %d steps, want %d", len(pc.Steps), tt.want)
			}
			for i, req := range tt.reqs {
				if pc.Steps[i].Target != req {
					t.Errorf("NewChain() step %d target = %q, want %q", i, pc.Steps[i].Target, req)
				}
				if pc.Steps[i].Process != nil {
					t.Errorf("NewChain() step %d process should be nil initially", i)
				}
			}
		})
	}
}

func TestProductionChain_fillOneChain(t *testing.T) {
	df := getTestDataFile(t)

	tests := []struct {
		name    string
		target  string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid target with process",
			target:  "Iron Ingot",
			wantErr: false,
		},
		{
			name:    "target with no non-special processes",
			target:  "Iron Ore", // only has mining process
			wantErr: false,
		},
		{
			name:    "non-existent target",
			target:  "Nonexistent Item",
			wantErr: true,
			errMsg:  "no processes found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pc := df.NewChain([]string{tt.target})
			err := pc.fillOneChain(0)

			if (err != nil) != tt.wantErr {
				t.Errorf("fillOneChain() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errMsg != "" {
				if err == nil || !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("fillOneChain() error = %v, want error containing %q", err, tt.errMsg)
				}
			}

			if !tt.wantErr {
				if pc.Steps[0].Process == nil {
					t.Error("fillOneChain() did not set process for valid target")
				}
			}
		})
	}
}

func TestProductionChain_fillOneChain_AlreadyFilled(t *testing.T) {
	df := getTestDataFile(t)
	pc := df.NewChain([]string{"Iron Ingot"})

	// Fill the chain once
	err := pc.fillOneChain(0)
	if err != nil {
		t.Fatalf("First fillOneChain() failed: %v", err)
	}

	// Try to fill again - should return error
	err = pc.fillOneChain(0)
	if err == nil {
		t.Error("fillOneChain() should return error when chain already filled")
	}
	if !strings.Contains(err.Error(), "already filled") {
		t.Errorf("fillOneChain() error = %v, want error containing 'already filled'", err)
	}
}

func TestProductionChain_FillChain(t *testing.T) {
	df := getTestDataFile(t)

	tests := []struct {
		name    string
		targets []string
		wantErr bool
	}{
		{
			name:    "simple chain",
			targets: []string{"Iron Ingot"},
			wantErr: false,
		},
		{
			name:    "complex chain",
			targets: []string{"Circuit Board"},
			wantErr: false,
		},
		{
			name:    "chain with dependencies",
			targets: []string{"Electric Motor"},
			wantErr: false,
		},
		{
			name:    "unmakeable item",
			targets: []string{"Nonexistent Item"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pc := df.NewChain(tt.targets)
			err := pc.FillChain()

			if (err != nil) != tt.wantErr {
				t.Errorf("FillChain() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Verify all steps have processes assigned
				for i, step := range pc.Steps {
					if step.Process == nil {
						t.Errorf("FillChain() step %d (%s) has no process assigned", i, step.Target)
					}
				}
			}
		})
	}
}

func TestProductionChain_GetAllProducible(t *testing.T) {
	df := getTestDataFile(t)
	pc := df.NewChain([]string{"Iron Ore", "Copper Ore"})

	err := pc.GetAllProducible()
	if err != nil {
		t.Errorf("GetAllProducible() error = %v", err)
		return
	}

	// Should have added more items beyond the initial ones
	if len(pc.Steps) <= 2 {
		t.Errorf("GetAllProducible() should have added more items, got %d steps", len(pc.Steps))
	}

	// Verify we have some expected items
	targets := make(map[string]bool)
	for _, step := range pc.Steps {
		targets[step.Target] = true
	}

	expectedItems := []string{"Iron Ore", "Copper Ore", "Iron Ingot", "Copper Ingot"}
	for _, item := range expectedItems {
		if !targets[item] {
			t.Errorf("GetAllProducible() missing expected item: %s", item)
		}
	}
}

func TestProductionChain_GetAllProducibleExcluding(t *testing.T) {
	df := getTestDataFile(t)
	pc := df.NewChain([]string{"Iron Ore"})

	exclusions := []string{"Gear"}
	err := pc.GetAllProducibleExcluding(exclusions)
	if err != nil {
		t.Errorf("GetAllProducibleExcluding() error = %v", err)
		return
	}

	// Verify excluded items are not present
	for _, step := range pc.Steps {
		for _, excluded := range exclusions {
			if step.Target == excluded {
				t.Errorf("GetAllProducibleExcluding() included excluded item: %s", excluded)
			}
		}
	}
}

func TestProductionChain_String(t *testing.T) {
	df := getTestDataFile(t)
	pc := df.NewChain([]string{"Iron Ingot"})

	// Test string representation before filling
	str := pc.String()
	if !strings.Contains(str, "Iron Ingot") {
		t.Error("ProductionChain.String() should contain target name")
	}
	if !strings.Contains(str, "<unknown>") {
		t.Error("ProductionChain.String() should show <unknown> for unfilled process")
	}

	// Fill the chain and test again
	err := pc.FillChain()
	if err != nil {
		t.Fatalf("FillChain() failed: %v", err)
	}

	str = pc.String()
	if !strings.Contains(str, "Iron Ingot") {
		t.Error("ProductionChain.String() should contain target name after filling")
	}
	if strings.Contains(str, "<unknown>") {
		t.Error("ProductionChain.String() should not show <unknown> after filling")
	}
}

func TestProductionStep_String(t *testing.T) {
	df := getTestDataFile(t)

	// Test unfilled step
	step := ProductionStep{Target: "Iron Ingot"}
	str := step.String()
	if !strings.Contains(str, "Iron Ingot") {
		t.Error("ProductionStep.String() should contain target name")
	}
	if !strings.Contains(str, "<unknown>") {
		t.Error("ProductionStep.String() should show <unknown> for nil process")
	}

	// Test filled step with requirements
	pc := df.NewChain([]string{"Iron Ingot"})
	err := pc.FillChain()
	if err != nil {
		t.Fatalf("FillChain() failed: %v", err)
	}

	if len(pc.Steps) > 0 && pc.Steps[0].Process != nil {
		str = pc.Steps[0].String()
		if !strings.Contains(str, "Iron Ingot") {
			t.Error("ProductionStep.String() should contain target name")
		}
	}

	// Test step with process that has no consumes (basic resource)
	pc2 := df.NewChain([]string{"Iron Ore"})
	err = pc2.FillChain()
	if err != nil {
		t.Fatalf("FillChain() failed: %v", err)
	}

	if len(pc2.Steps) > 0 && pc2.Steps[0].Process != nil {
		str = pc2.Steps[0].String()
		if !strings.Contains(str, "Iron Ore") {
			t.Error("ProductionStep.String() should contain target name")
		}
		if !strings.Contains(str, "<produced by") {
			t.Error("ProductionStep.String() should show facility info for basic resources")
		}
	}
}

func TestProductionChain_EdgeCases(t *testing.T) {
	df := getTestDataFile(t)

	// Test chain with special processes (should be skipped)
	pc := df.NewChain([]string{"Special Item"})
	err := pc.FillChain()
	if err == nil {
		t.Error("FillChain() should fail for items only available through special processes")
	}

	// Test empty chain operations
	emptyChain := df.NewChain([]string{})
	err = emptyChain.FillChain()
	if err != nil {
		t.Errorf("FillChain() should not error on empty chain: %v", err)
	}

	err = emptyChain.GetAllProducible()
	if err != nil {
		t.Errorf("GetAllProducible() should not error on empty chain: %v", err)
	}

	str := emptyChain.String()
	if str != "" {
		t.Errorf("String() should return empty string for empty chain, got: %q", str)
	}
}

func TestProductionChain_FillChain_StableOrder(t *testing.T) {
	df := getTestDataFile(t)

	// Test that FillChain produces stable ordering across multiple runs
	const numRuns = 10
	var results [][]string

	for i := 0; i < numRuns; i++ {
		pc := df.NewChain([]string{"Circuit Board"})
		err := pc.FillChain()
		if err != nil {
			t.Fatalf("FillChain() run %d failed: %v", i, err)
		}

		// Extract the targets in order
		targets := make([]string, len(pc.Steps))
		for j, step := range pc.Steps {
			targets[j] = step.Target
		}
		results = append(results, targets)
	}

	// Verify all runs produced the same order
	firstResult := results[0]
	for i := 1; i < numRuns; i++ {
		if len(results[i]) != len(firstResult) {
			t.Errorf("Run %d produced %d steps, expected %d", i, len(results[i]), len(firstResult))
			continue
		}

		for j := 0; j < len(firstResult); j++ {
			if results[i][j] != firstResult[j] {
				t.Errorf("Run %d step %d: got %q, expected %q", i, j, results[i][j], firstResult[j])
				t.Errorf("Run %d order: %v", i, results[i])
				t.Errorf("Expected order: %v", firstResult)
				break
			}
		}
	}

	t.Logf("FillChain stable order verified across %d runs with %d steps", numRuns, len(firstResult))
}

func TestProductionChain_GetAllProducible_StableOrder(t *testing.T) {
	df := getTestDataFile(t)

	// Test that GetAllProducible produces stable ordering across multiple runs
	const numRuns = 10
	var results [][]string

	for i := 0; i < numRuns; i++ {
		pc := df.NewChain([]string{"Iron Ore", "Copper Ore"})
		err := pc.GetAllProducible()
		if err != nil {
			t.Fatalf("GetAllProducible() run %d failed: %v", i, err)
		}

		// Extract the targets in order
		targets := make([]string, len(pc.Steps))
		for j, step := range pc.Steps {
			targets[j] = step.Target
		}
		results = append(results, targets)
	}

	// Verify all runs produced the same order
	firstResult := results[0]
	for i := 1; i < numRuns; i++ {
		if len(results[i]) != len(firstResult) {
			t.Errorf("Run %d produced %d steps, expected %d", i, len(results[i]), len(firstResult))
			continue
		}

		for j := 0; j < len(firstResult); j++ {
			if results[i][j] != firstResult[j] {
				t.Errorf("Run %d step %d: got %q, expected %q", i, j, results[i][j], firstResult[j])
				t.Errorf("Run %d order: %v", i, results[i])
				t.Errorf("Expected order: %v", firstResult)
				break
			}
		}
	}

	t.Logf("GetAllProducible stable order verified across %d runs with %d steps", numRuns, len(firstResult))
}

func TestProductionChain_GetAllProducibleExcluding_StableOrder(t *testing.T) {
	df := getTestDataFile(t)

	// Test that GetAllProducibleExcluding produces stable ordering across multiple runs
	const numRuns = 10
	var results [][]string
	exclusions := []string{"Gear"}

	for i := 0; i < numRuns; i++ {
		pc := df.NewChain([]string{"Iron Ore", "Copper Ore"})
		err := pc.GetAllProducibleExcluding(exclusions)
		if err != nil {
			t.Fatalf("GetAllProducibleExcluding() run %d failed: %v", i, err)
		}

		// Extract the targets in order
		targets := make([]string, len(pc.Steps))
		for j, step := range pc.Steps {
			targets[j] = step.Target
		}
		results = append(results, targets)
	}

	// Verify all runs produced the same order
	firstResult := results[0]
	for i := 1; i < numRuns; i++ {
		if len(results[i]) != len(firstResult) {
			t.Errorf("Run %d produced %d steps, expected %d", i, len(results[i]), len(firstResult))
			continue
		}

		for j := 0; j < len(firstResult); j++ {
			if results[i][j] != firstResult[j] {
				t.Errorf("Run %d step %d: got %q, expected %q", i, j, results[i][j], firstResult[j])
				t.Errorf("Run %d order: %v", i, results[i])
				t.Errorf("Expected order: %v", firstResult)
				break
			}
		}
	}

	// Verify exclusions are respected
	for _, step := range firstResult {
		for _, excluded := range exclusions {
			if step == excluded {
				t.Errorf("GetAllProducibleExcluding() included excluded item: %s", excluded)
			}
		}
	}

	t.Logf("GetAllProducibleExcluding stable order verified across %d runs with %d steps", numRuns, len(firstResult))
}

func TestProductionChain_StringOutput_StableOrder(t *testing.T) {
	df := getTestDataFile(t)

	// Test that String() output is stable across multiple runs
	const numRuns = 5
	var results []string

	for i := 0; i < numRuns; i++ {
		pc := df.NewChain([]string{"Circuit Board"})
		err := pc.FillChain()
		if err != nil {
			t.Fatalf("FillChain() run %d failed: %v", i, err)
		}

		str := pc.String()
		results = append(results, str)
	}

	// Verify all runs produced the same string output
	firstResult := results[0]
	for i := 1; i < numRuns; i++ {
		if results[i] != firstResult {
			t.Errorf("Run %d produced different string output", i)
			t.Errorf("Run %d output:\n%s", i, results[i])
			t.Errorf("Expected output:\n%s", firstResult)
		}
	}

	t.Logf("String output stable across %d runs", numRuns)
}
