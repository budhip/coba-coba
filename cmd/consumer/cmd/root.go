package cmd

import (
	"context"
	"log"
	"os"

	"bitbucket.org/Amartha/go-fp-transaction/cmd/setup"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/graceful"
	"bitbucket.org/Amartha/go-fp-transaction/internal/deliveries/consumer"
	kafkaconsumer "bitbucket.org/Amartha/go-fp-transaction/internal/deliveries/consumer/kafka"
	"bitbucket.org/Amartha/go-fp-transaction/internal/deliveries/http/health"

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
	if err := rootCmd.Execute(); err != nil {
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

	consumerName, _ := ccmd.Flags().GetString(runConsumerCmdName)
	xlog.Infof(ctx, "initializing consumer: %s", consumerName)

	// Step 1: Initialize setup
	s, stopperContract, err := setup.Init("consumer-" + consumerName)
	if err != nil {
		log.Fatalf("failed to setup app: %v", err)
	}

	// Step 2: Create Kafka consumer
	consumerProcess, consumerStopper, err := consumer.NewKafkaConsumer(ctx, consumerName, s.Config, s.Service, s.RepoCache, s)
	if err != nil {
		// Only stop setup resources, not consumer resources (they don't exist yet)
		xlog.Fatalf(ctx, "failed to setup consumer: %v", err)
	}

	// Step 3: Create health check server
	check := health.NewHealthCheck()
	healthCheckProcess := kafkaconsumer.NewHTTPServer(ctx, s.Config, s.Metrics, check)

	// Step 4: Collect all starters and stoppers
	starters = append(starters, consumerProcess.Start(), healthCheckProcess.Start())
	// Since graceful.StopProcess() calls slices.Reverse(), we append in OPPOSITE order:
	stoppers = append(stoppers, stopperContract...)        // Added FIRST → Will stop LAST (Kafka producers, DB, Cache)
	stoppers = append(stoppers, consumerStopper...)        // Added 2nd → Will stop 3rd (Consumer resources)
	stoppers = append(stoppers, consumerProcess.Stop())    // Added 3rd → Will stop 2nd (Kafka consumer)
	stoppers = append(stoppers, healthCheckProcess.Stop()) // Added LAST → Will stop FIRST (Health check HTTP)

	xlog.Info(ctx, "starting consumer services in background...")
	graceful.StartProcessAtBackground(starters...)

	xlog.Infof(ctx, "consumer %s started, waiting for shutdown signal...", consumerName)

	// Block until shutdown signal is received (includes 10 second sleep)
	graceful.StopProcessAtBackground(ctx)
	check.Shutdown()
	graceful.StopProcess(ctx, s.Config.App.GracefulTimeout, stoppers...)

	xlog.Infof(ctx, "consumer %s stopped successfully!", consumerName)
}
