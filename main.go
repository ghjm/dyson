package main

import (
	_ "embed"
	"fmt"
	"github.com/ghjm/dyson/pkg/dyson"
	"github.com/spf13/cobra"
	"os"
	"strconv"
	"strings"
)

//go:embed data.yml
var dataFileContent []byte

var gitCommit string

const gitRepo = "https://github.com/ghjm/dyson"

func main() {
	rootCmd := &cobra.Command{
		Use:          "dyson",
		Short:        "Dyson Sphere Program CLI calculator",
		SilenceUsage: true,
	}

	var dataFile string
	rootCmd.PersistentFlags().StringVar(&dataFile, "data", "", "path to data file")

	loadData := func() (*dyson.DataFile, error) {
		var data []byte
		if dataFile == "" {
			data = dataFileContent
		} else {
			var err error
			data, err = os.ReadFile(dataFile)
			if err != nil {
				return nil, fmt.Errorf("error reading data file: %w", err)
			}
		}
		df, err := dyson.LoadData(data)
		if err != nil {
			return nil, fmt.Errorf("error loading data: %w", err)
		}
		return df, nil
	}

	validateCmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate that the data file is correct",
		RunE: func(cmd *cobra.Command, args []string) error {
			df, err := loadData()
			if err != nil {
				return err
			}
			err = df.Validate()
			if err != nil {
				return fmt.Errorf("error validating data: %w", err)
			}
			fmt.Println("Validation successful!")
			return nil
		},
	}
	rootCmd.AddCommand(validateCmd)

	var haveItems []string
	var factoriesMode bool
	chainCmd := &cobra.Command{
		Use:   "chain",
		Short: "Calculate production chain for a given list of items.  Give item:rate to specify a target rate.",
		RunE: func(cmd *cobra.Command, args []string) error {
			df, err := loadData()
			if err != nil {
				return err
			}
			var reqs []string
			rates := make(map[string]float32)
			for _, arg := range args {
				if strings.Contains(arg, ":") {
					parts := strings.Split(arg, ":")
					if len(parts) != 2 {
						return fmt.Errorf("invalid argument: %s", arg)
					}
					pRate, err := strconv.ParseFloat(parts[1], 32)
					if err != nil {
						return fmt.Errorf("invalid rate: %s", parts[1])
					}
					rate := float32(pRate)
					if factoriesMode {
						rate, err = df.FactoriesToItemsPerSecond(parts[0], rate)
						if err != nil {
							return fmt.Errorf("error calculating rate: %w", err)
						}
					}
					reqs = append(reqs, parts[0])
					rates[parts[0]] = rate
				} else {
					reqs = append(reqs, arg)
				}
			}
			ch := df.NewChain(reqs)
			for item, rate := range rates {
				err = ch.SetRate(item, rate)
				if err != nil {
					return fmt.Errorf("error setting rate: %w", err)
				}
			}
			err = ch.FillChainExcluding(haveItems)
			if err != nil {
				return fmt.Errorf("error filling chain: %w", err)
			}
			var opts []dyson.StringOption
			if factoriesMode {
				opts = append(opts, dyson.WithUnitConverter(
					func(item string, rate float32) (bool, float32, string) {
						newRate, err := df.ItemsPerSecondToFactories(item, rate)
						if err != nil {
							return false, 0, ""
						} else {
							return true, newRate, " factories"
						}
					}))
			}
			fmt.Printf("%s", ch.StringWithOpts(opts...))
			return nil
		},
	}
	chainCmd.Flags().StringArrayVar(&haveItems, "have", []string{}, "Items you already have (excludes them from the chain)")
	chainCmd.Flags().BoolVar(&factoriesMode, "factories", false, "Interpret rates as number of factories instead of items per second")
	rootCmd.AddCommand(chainCmd)

	makesCmd := &cobra.Command{
		Use:   "makes",
		Short: "Calculate what can be produced from a given list of items",
		RunE: func(cmd *cobra.Command, args []string) error {
			df, err := loadData()
			if err != nil {
				return err
			}
			ch := df.NewChain(args)
			err = ch.GetAllProducible()
			if err != nil {
				return fmt.Errorf("error filling chain: %w", err)
			}
			fmt.Printf("%s", ch.String())
			return nil
		},
	}
	rootCmd.AddCommand(makesCmd)

	var oldItems []string
	var newItems []string
	var oldExcludes []string
	var newExcludes []string
	diffCmd := &cobra.Command{
		Use:   "diff",
		Short: "Calculate what additional items can be produced when adding a new resource",
		RunE: func(cmd *cobra.Command, args []string) error {
			df, err := loadData()
			if err != nil {
				return err
			}
			var reqs []string
			reqs = append(reqs, oldItems...)
			chOld := df.NewChain(reqs)
			err = chOld.GetAllProducibleExcluding(oldExcludes)
			if err != nil {
				return fmt.Errorf("error filling old chain: %w", err)
			}
			reqs = append(reqs, newItems...)
			chNew := df.NewChain(reqs)
			err = chNew.GetAllProducibleExcluding(newExcludes)
			if err != nil {
				return fmt.Errorf("error filling new chain: %w", err)
			}
			oldTargets := make(map[string]struct{})
			for _, s := range chOld.Steps {
				oldTargets[s.Target] = struct{}{}
			}
			for _, s := range chNew.Steps {
				_, ok := oldTargets[s.Target]
				if !ok {
					fmt.Println(s.String())
				}
			}
			return nil
		},
	}
	diffCmd.Flags().StringArrayVar(&oldItems, "old", []string{}, "old items")
	diffCmd.Flags().StringArrayVar(&newItems, "new", []string{}, "new items")
	diffCmd.Flags().StringArrayVar(&oldExcludes, "exclude-old", []string{}, "banned items")
	diffCmd.Flags().StringArrayVar(&newExcludes, "exclude-new", []string{}, "banned items")
	rootCmd.AddCommand(diffCmd)

	resourcesCmd := &cobra.Command{
		Use:   "resources",
		Short: "Lists items that can be directly mined, pumped, etc.",
		RunE: func(cmd *cobra.Command, args []string) error {
			df, err := loadData()
			if err != nil {
				return err
			}
			for _, proc := range df.Processes {
				if len(proc.Consumes) == 0 {
					for m := range proc.Makes {
						fmt.Println(m)
					}
				}
			}
			return nil
		},
	}
	rootCmd.AddCommand(resourcesCmd)

	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Shows the git commit this was built from",
		Run: func(cmd *cobra.Command, args []string) {
			if gitCommit == "" {
				fmt.Printf("This is a development build with no version information.\n")
			} else {
				fmt.Printf("This program is unversioned, but this copy was built from:\n")
				fmt.Printf("%s/commit/%s\n", gitRepo, gitCommit)
			}
			fmt.Printf("\n")
		},
	}
	rootCmd.AddCommand(versionCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
