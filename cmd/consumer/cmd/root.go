package cmd

import (
	"context"
	"log"
	"os"
	"sync"
	"time"

	"bitbucket.org/Amartha/go-fp-transaction/cmd/setup"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/graceful"
	"bitbucket.org/Amartha/go-fp-transaction/internal/deliveries/consumer"
	kafkaconsumer "bitbucket.org/Amartha/go-fp-transaction/internal/deliveries/consumer/kafka"

	xlog "bitbucket.org/Amartha/go-x/log"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "consumer",
	Short: "Consumer is a consumer application for handling message transaction or dlq",
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

func init() {
	rootCmd.AddCommand(runJobCmd)

	runJobCmd.Flags().StringP(runConsumerCmdName, "n", "", "job name")
	runJobCmd.MarkFlagRequired(runConsumerCmdName)
}

var (
	runJobCmd = &cobra.Command{
		Use:     "run",
		Short:   "Run consumer",
		Long:    `Run consumer for handling message transaction or dlq, available consumer type: transaction, transaction.dlq`,
		Example: "consumer run -n={consumer-type-name}",
		Run:     runConsumer,
	}
	runConsumerCmdName = "name"
)

func runConsumer(ccmd *cobra.Command, args []string) {
	var (
		ctx      = context.Background()
		starters []graceful.ProcessStarter
		stoppers []graceful.ProcessStopper
	)
	ctx, cancel := context.WithCancel(ctx)

	name, _ := ccmd.Flags().GetString(runConsumerCmdName)

	s, stopperContract, err := setup.Init("consumer-" + name)
	stoppers = append(stoppers, stopperContract...)
	if err != nil {
		timeout := 5 * time.Second
		if s != nil && s.Config.App.GracefulTimeout != 0 {
			timeout = s.Config.App.GracefulTimeout
		}
		graceful.StopProcess(timeout, stoppers...)
		log.Fatalf("failed to setup app: %v", err)
	}

	consumerProcess, consumerStopper, err := consumer.NewKafkaConsumer(ctx, name, s.Config, s.Service, s.RepoCache, s)
	stoppers = append(stoppers, consumerStopper...)
	if err != nil {
		graceful.StopProcess(s.Config.App.GracefulTimeout, stoppers...)
		xlog.Fatalf(ctx, "failed to setup consumer: %v", err)
	}

	healthCheckProcess := kafkaconsumer.NewHTTPServer(ctx, s.Config, s.Metrics)

	starters = append(starters, consumerProcess.Start(), healthCheckProcess.Start())
	stoppers = append(stoppers, consumerProcess.Stop(), healthCheckProcess.Stop())

	wg := new(sync.WaitGroup)
	wg.Add(1)
	go func() {
		graceful.StartProcessAtBackground(starters...)
		graceful.StopProcessAtBackground(s.Config.App.GracefulTimeout, stoppers...)
		wg.Done()
	}()

	wg.Wait()
	cancel()
	xlog.Info(ctx, "consumer server stopped!")
}
