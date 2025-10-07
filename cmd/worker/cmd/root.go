/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"fmt"
	"os"

	"bitbucket.org/Amartha/go-fp-transaction/cmd/setup"
	helperFlag "bitbucket.org/Amartha/go-fp-transaction/internal/common/flag"
	"bitbucket.org/Amartha/go-fp-transaction/internal/deliveries/job"
	xlog "bitbucket.org/Amartha/go-x/log"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "worker",
	Short: "Worker application to configuring and running a job",
	Long:  ``,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

var (
	j *job.Job
)

func init() {
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(runJobCmd)

	runJobCmd.Flags().StringP(runJobCmdName, "n", "", "job name")
	runJobCmd.MarkFlagRequired(runJobCmdName)
	runJobCmd.Flags().StringP(runJobCmdVersion, "v", "", "job version")
	runJobCmd.MarkFlagRequired(runJobCmdVersion)
	runJobCmd.Flags().StringP(runJobCmdDate, "d", "", "job running date")
	runJobCmd.Flags().StringP(runJobCmdFileName, "f", "", "file name")
	runJobCmd.Flags().StringP(runJobCmdBucketName, "b", "", "bucket name")
	runJobCmd.Flags().BoolP(runJobCmdFlagPublish, "p", false, "flag publish")
}

var (
	listCmd = &cobra.Command{
		Use:   "list",
		Short: "List job name and version",
		Long:  ``,
		Run:   list,
	}
)

func list(ccmd *cobra.Command, args []string) {
	for version, l := range j.Routes {
		for name := range l {
			list := fmt.Sprintf("version=%s, name=%s", version, name)
			fmt.Println(list)
		}
	}
}

var (
	runJobCmd = &cobra.Command{
		Use:     "run",
		Short:   "Run execution job",
		Long:    ``,
		Example: "worker run -n={job-name} -v={job-version} -d={job-date}",
		Run:     runJob,
	}
	runJobCmdName        = "name"
	runJobCmdVersion     = "version"
	runJobCmdDate        = "date"
	runJobCmdFileName    = "file"
	runJobCmdBucketName  = "bucket"
	runJobCmdFlagPublish = "publishToAcuanNotif"
)

func runJob(ccmd *cobra.Command, args []string) {
	var (
		ctx = context.Background()
	)

	name, _ := ccmd.Flags().GetString(runJobCmdName)
	version, _ := ccmd.Flags().GetString(runJobCmdVersion)
	date, _ := ccmd.Flags().GetString(runJobCmdDate)
	fileName, _ := ccmd.Flags().GetString(runJobCmdFileName)
	bucketName, _ := ccmd.Flags().GetString(runJobCmdBucketName)
	flagPublishAcuan, _ := ccmd.Flags().GetBool(runJobCmdFlagPublish)

	s, _, err := setup.Init("job")
	if err != nil {
		xlog.Fatalf(ctx, "failed to setup app: %v", err)
	}

	defer func() {
		s.WriteDB.Close()
		s.ReadDB.Close()
		s.Cache.Close()
		s.RepoCloudStorage.Close()
	}()

	j = job.New(s.Config, s.Service)
	j.Start(ctx, helperFlag.Job{
		JobName:          name,
		Version:          version,
		Date:             date,
		FileName:         fileName,
		BucketName:       bucketName,
		FlagPublishAcuan: flagPublishAcuan,
	})
	xlog.Info(ctx, "job server stopped!")
}
