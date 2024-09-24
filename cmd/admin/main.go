package main

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "proglv",
		Short: "Admin CLI tool for programme.lv",
	}

	var taskCmd = &cobra.Command{
		Use:   "task",
		Short: "Transform & upload tasks",
	}

	var src string
	var dst string
	var format string

	var taskTransformCmd = &cobra.Command{
		Use:   "transform",
		Short: "Transform task format to proglv format",
		Run: func(cmd *cobra.Command, args []string) {
			switch format {
			case "lio2023":
				importLio2023Task(src, dst)
			default:
				log.Fatalf("Unsupported format: %s\n", format)
			}
		},
	}

	// Define flags for the 'import' command
	taskTransformCmd.Flags().StringVarP(&src, "src", "s", "", "Source directory path (required)")
	taskTransformCmd.Flags().StringVarP(&dst, "dst", "d", "", "Destination directory path (required)")
	taskTransformCmd.Flags().StringVarP(&format, "format", "f", "", "Format of the import [lio2023, lio2024] (required)")

	// Mark 'src' and 'dst' as required flags
	taskTransformCmd.MarkFlagRequired("src")
	taskTransformCmd.MarkFlagRequired("dst")
	taskTransformCmd.MarkFlagRequired("format")

	// Build the command hierarchy
	rootCmd.AddCommand(taskCmd)
	taskCmd.AddCommand(taskTransformCmd)

	// Execute the root command
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func importLio2023Task(src string, dst string) error {
	fmt.Println("Transforming LIO 2023 task with the following parameters:")
	fmt.Printf("Source: %s\n", src)
	fmt.Printf("Destination: %s\n", dst)
	return nil
}
