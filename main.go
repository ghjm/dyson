package main

import (
	_ "embed"
	"fmt"
	"github.com/ghjm/dyson/pkg/dyson"
	"github.com/spf13/cobra"
	"os"
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

	chainCmd := &cobra.Command{
		Use:   "chain",
		Short: "Calculate production chain for a given list of items",
		RunE: func(cmd *cobra.Command, args []string) error {
			df, err := loadData()
			if err != nil {
				return err
			}
			ch := df.NewChain(args)
			err = ch.FillChain()
			if err != nil {
				return fmt.Errorf("error filling chain: %w", err)
			}
			fmt.Printf("%s", ch.String())
			return nil
		},
	}
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
