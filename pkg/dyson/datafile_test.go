package dyson

import (
	"strings"
	"testing"
)

// Test data for consistent testing
var testYAMLData = `
facilities:
  smelter:
    Arc Smelter: 1
    Plane Smelter: 2
  assembler:
    Assembling Machine Mk. I: 0.75
    Assembling Machine Mk. II: 1
  mine:
    Mining Machine: 1

processes:
  - makes:
      Iron Ore: 1
    time: 2
    facility: [ mine ]

  - makes:
      Iron Ingot: 1
    consumes:
      Iron Ore: 1
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
      Copper Ore: 1
    time: 2
    facility: [ mine ]

  - makes:
      Copper Ingot: 1
    consumes:
      Copper Ore: 1
    time: 1
    facility: [ smelter ]
`

var invalidYAMLData = `
invalid yaml content [
`

func TestLoadData(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		wantErr bool
	}{
		{
			name:    "valid YAML data",
			data:    []byte(testYAMLData),
			wantErr: false,
		},
		{
			name:    "invalid YAML data",
			data:    []byte(invalidYAMLData),
			wantErr: true,
		},
		{
			name:    "empty data",
			data:    []byte(""),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			df, err := LoadData(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadData() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && df == nil {
				t.Error("LoadData() returned nil DataFile without error")
			}
			if !tt.wantErr && df != nil {
				// Verify procsByTarget is initialized
				if df.procsByTarget == nil {
					t.Error("LoadData() did not initialize procsByTarget")
				}
				// Verify some processes are indexed correctly
				if len(df.Processes) > 0 {
					found := false
					for _, proc := range df.Processes {
						for item := range proc.Makes {
							if procs, exists := df.procsByTarget[item]; exists && len(procs) > 0 {
								found = true
								break
							}
						}
						if found {
							break
						}
					}
					if !found && len(df.Processes) > 0 {
						t.Error("LoadData() did not properly index processes by target")
					}
				}
			}
		})
	}
}

func TestDataFile_Makeable(t *testing.T) {
	df, err := LoadData([]byte(testYAMLData))
	if err != nil {
		t.Fatalf("Failed to load test data: %v", err)
	}

	tests := []struct {
		name string
		item string
		want bool
	}{
		{
			name: "basic resource (Iron Ore)",
			item: "Iron Ore",
			want: true,
		},
		{
			name: "processed item (Iron Ingot)",
			item: "Iron Ingot",
			want: true,
		},
		{
			name: "complex item (Gear)",
			item: "Gear",
			want: true,
		},
		{
			name: "item requiring multiple inputs (Circuit Board)",
			item: "Circuit Board",
			want: true,
		},
		{
			name: "non-existent item",
			item: "Nonexistent Item",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := df.Makeable(tt.item)
			if got != tt.want {
				t.Errorf("DataFile.Makeable(%q) = %v, want %v", tt.item, got, tt.want)
			}
		})
	}
}

func TestDataFile_Validate(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid data",
			data:    testYAMLData,
			wantErr: false,
		},
		{
			name: "zero facility rate",
			data: `
facilities:
  smelter:
    Arc Smelter: 0
processes: []
`,
			wantErr: true,
			errMsg:  "facility rate is zero",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			df, err := LoadData([]byte(tt.data))
			if err != nil {
				t.Fatalf("Failed to load test data: %v", err)
			}

			err = df.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("DataFile.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errMsg != "" {
				if err == nil || !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("DataFile.Validate() expected error containing %q, got %v", tt.errMsg, err)
				}
			}
		})
	}
}

// Test duplicate facility validation by creating DataFile directly
func TestDataFile_ValidateDuplicates(t *testing.T) {
	// Test duplicate facility type by manually creating DataFile
	df := &DataFile{
		Facilities: map[string]map[string]float32{
			"smelter": {
				"Arc Smelter": 1,
			},
		},
		Processes:     []Process{},
		procsByTarget: make(map[string][]Process),
	}

	// Manually add duplicate facility type (this simulates what would happen
	// if YAML allowed duplicates)
	df.Facilities["smelter2"] = map[string]float32{
		"Plane Smelter": 2,
	}

	err := df.Validate()
	if err != nil {
		t.Errorf("DataFile.Validate() should not error on different facility types, got: %v", err)
	}

	// Test zero facility rate
	df.Facilities["smelter"]["Zero Rate Smelter"] = 0
	err = df.Validate()
	if err == nil {
		t.Error("DataFile.Validate() should error on zero facility rate")
	}
	if !strings.Contains(err.Error(), "facility rate is zero") {
		t.Errorf("DataFile.Validate() error should mention zero rate, got: %v", err)
	}
}

func TestDataFile_ValidateUnmakeableItem(t *testing.T) {
	// Test case where an item cannot be made
	unmakeableData := `
facilities:
  assembler:
    Assembling Machine Mk. I: 1
processes:
  - makes:
      Complex Item: 1
    consumes:
      Unmakeable Resource: 1
    time: 1
    facility: [ assembler ]
`

	df, err := LoadData([]byte(unmakeableData))
	if err != nil {
		t.Fatalf("Failed to load test data: %v", err)
	}

	err = df.Validate()
	if err == nil {
		t.Error("DataFile.Validate() expected error for unmakeable item, got nil")
	}
}
