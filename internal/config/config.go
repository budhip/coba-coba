package config

import (
	"time"
)

type (
	Config struct {
		App                 App      `json:"app"`
		Postgres            Postgres `json:"postgres"`
		Redis               Redis    `json:"redis"`
		SecretKey           string   `json:"secret_key"`
		GcloudProjectID     string   `json:"gcloud_project_id"`
		NewRelicLicenseKey  string   `json:"new_relic_license_key"`
		HostGoFPTransaction string   `json:"host_go_fp_transaction"`

		FeatureFlag                 FeatureFlag                 `json:"feature_flag"`
		TransactionConfig           TransactionConfig           `json:"transaction_config"`
		AccountConfig               AccountConfig               `json:"account_config"`
		MessageBroker               MessageBroker               `json:"message_broker"`
		CloudStorageConfig          CloudStorageConfig          `json:"cloud_storage"`
		MasterData                  MasterDataConfig            `json:"master_data"`
		ExponentialBackoff          ExponentialBackOffConfig    `json:"exponential_backoff"`
		ReconEngine                 ReconEngineConfig           `json:"recon_engine"`
		TransactionValidationConfig TransactionValidationConfig `json:"transaction_validation_config"`
		AccountFeatureConfig        map[string]FeatureConfig    `json:"account_feature_config"`

		GoQueueUnicorn       HTTPConfiguration     `json:"go_queue_unicorn"`
		GoAccounting         HTTPConfiguration     `json:"go_accounting"`
		DDDNotification      DDDNotificationConfig `json:"ddd_notification"`
		FeatureFlagSDKConfig FeatureFlagSDKConfig  `json:"feature_flag_sdk"`

		FeatureFlagKeyLookup FeatureFlagKeyLookup `json:"feature_flag_key_lookup"`

		AcuanLibConfig AcuanLibConfig `json:"go_acuan_lib"`
	}

	App struct {
		Env             string        `json:"env"`
		HTTPPort        int           `json:"http_port"`
		HTTPTimeout     time.Duration `json:"http_timeout"`
		GracefulTimeout time.Duration `json:"graceful_timeout"`
		Name            string        `json:"name"`
		LogOption       string        `json:"log_option"`
		LogLevel        string        `json:"log_level"`
	}

	Postgres struct {
		Write Database `json:"write"`
		Read  Database `json:"read"`
	}

	Database struct {
		DbHost            string `json:"db_host"`
		DbPort            string `json:"db_port"`
		DbUser            string `json:"db_user"`
		DbPass            string `json:"db_pass"`
		DbName            string `json:"db_name"`
		DbSchema          string `json:"db_schema"`
		MaxOpenConnection int    `json:"maxOpenConnections"`
		MaxIdleConnection int    `json:"maxIdleConnections"`
		ConnMaxLifetime   int    `json:"connMaxLifetime"`
	}

	Redis struct {
		Host     string `json:"host"`
		Port     string `json:"port"`
		Password string `json:"password"`
		Db       int    `json:"db"`
	}

	FeatureFlag struct {
		EnableCheckAccountTransaction          bool `json:"enable_check_account_transaction"`
		EnableConsumerValidationReject         bool `json:"enable_consumer_validation_reject"`
		EnablePublishTransactionNotification   bool `json:"enable_publish_transaction_notification"`
		EnablePreventSameAccountMutationActing bool `json:"enable_prevent_same_account_mutation_acting"`
		EnableDelayBalanceUpdateOnHVTAccount   bool `json:"enable_delay_balance_update_on_hvt_account"`
		EnablePublishHvtBalanceDLQ             bool `json:"enable_publish_hvt_balance_dlq"`
	}

	// TransactionValidationConfig is used to configure validation when creating transaction
	// this config used in consumer side and api side
	TransactionValidationConfig struct {
		// AcceptedOrderType is the list of accepted order type
		// main purposes is for accepting legacy order type, since we already have validation
		// using payment master data from GCS. This also apply to AcceptedTransactionType
		AcceptedOrderType       []string `json:"accepted_order_type"`
		AcceptedTransactionType []string `json:"accepted_transaction_type"`

		SkipBalanceCheckAccountNumber []string `json:"skip_balance_check_account_number"`

		ValidateTUPEPCustomerNum bool `json:"validate_tupep_customer_num"`
		ValidateTUPEPLoanType    bool `json:"validate_tupep_loan_type"`

		ValidatePAYMDLoanAccountNumber bool `json:"validate_paymd_loan_account_number"`
		ValidatePAYMDLoanIDS           bool `json:"validate_paymd_loan_ids"`
	}

	TransactionConfig struct {
		BatchSize                          int           `json:"batch_size"`
		BalanceTTL                         time.Duration `json:"balance_ttl"`
		HandlerTimeoutWalletTransaction    time.Duration `json:"handler_timeout_wallet_transaction"`
		TransactionTimeUploadMaxWindowDays int           `json:"transaction_time_upload_max_window_days"`
		ReversalTimeRangeDays              int           `json:"reversal_time_range_days"`
		AsyncWalletTransactionForClients   []string      `json:"async_wallet_transaction_for_clients"`
	}

	AccountConfig struct {
		// TODO: remove this config, if we already fully migrated to go-accounting
		AccountNumberPadWidth int64 `json:"account_number_pad_width"`

		BPE                                  string `json:"bpe"`
		SplitEscrow                          string `json:"split_escrow"`
		SystemAccountNumber                  string `json:"system_account_number"`
		BRIEscrowAFAAccountNumber            string `json:"bri_escrow_afa_account_number"`
		BRI9539AccountNumber                 string `json:"bri_9539_account_number"`
		DepositCampaignBudget                string `json:"deposit_campaign_budget"`
		OperationalPayableGeneralCashDeposit string `json:"operational_payable_general_cash_deposit"`
		OperationalReceivableVoucher         string `json:"operational_receivable_voucher"`
		OperationalReceivableDigiasia        string `json:"operational_receivable_digiasia"`
		OperationalPayableGeneralCashback    string `json:"operational_payable_general_cashback"`
		DepositDiscountPPOB                  string `json:"deposit_discount_ppob"`
		DepositVoucher                       string `json:"deposit_voucher"`
		RemittanceAdminFeeBudget             string `json:"remittance_admin_fee_budget"`
		RemittanceAdminFeeRevenue            string `json:"remittance_admin_fee_revenue"`
		OtherReceivableRepayment             string `json:"other_receivable_repayment"`

		// OperationalReceivableAccountNumberByEntity is the account number by entity for "Piutang Ops Administrasi"
		OperationalReceivableAccountNumberByEntity map[string]string `json:"operational_receivable_account_number_by_entity"`

		// WHT2326Loan is the account number by loan for "Titipan Lender PPH for Loan"
		WHT2326Loan map[string]string `json:"wht_23_26_loan"`

		// AmarthaRevenueLoan is the account number by loan for "Utang Operasional Imbal Jasa for Loan"
		AmarthaRevenueLoan map[string]string `json:"amartha_revenue_loan"`

		// WHT2326ByEntityCode is the account number by loan by entity for "Titipan Lender PPH"
		WHT2326ByEntityCode map[string]map[string]string `json:"wht_23_26_by_entity"`

		// AmarthaRevenueByEntityCode is the account number by loan by entity for "Utang Operasional Imbal Jasa"
		AmarthaRevenueByEntityCode map[string]map[string]string `json:"amartha_revenue_by_entity"`

		// VATOutByEntityCode is the account number by loan for "Amartha PPN for Modal Loan"
		VATOutByEntityCode map[string]map[string]string `json:"vat_out_loan_by_entity"`

		// VATOutLoan is the account number by loan for "Amartha PPN for Loan"
		VATOutLoan map[string]string `json:"vat_out_loan"`

		// AmarthaRevenuePlatformFee is the account number by loan for "Utang Operasional Imbal Jasa for Platform Fee"
		AmarthaRevenuePlatformFee map[string]string `json:"amartha_revenue_platform_fee"`

		// AmarthaRevenueModalLoanPlatformFeeByEntityCode is the account number by loan by entity for "Utang Operasional Imbal Jasa for Modal Loan Platform Fee"
		AmarthaRevenueModalLoanPlatformFeeByEntityCode map[string]string `json:"amartha_revenue_modal_loan_platform_fee_by_entity"`

		// VATOutPlatformFee is the account number by loan for "Amartha PPN for Platform Fee"
		VATOutPlatformFee map[string]string `json:"vat_out_platform_fee"`

		PPOBCogsAccountNumber    map[string]string `json:"ppob_cogs_account_number"`
		PPOBRevenueAccountNumber map[string]string `json:"ppob_revenue_account_number"`

		// HVTSubCategoryCodes is list sub categories for account that have high volume transaction
		HVTSubCategoryCodes []string `json:"hvt_sub_category_codes"`

		// ExcludedBalanceUpdateAccountNumbers is list of account numbers that will be excluded from balance update
		// usually it's used for system account that we don't want to update the balance
		ExcludedBalanceUpdateAccountNumbers []string `json:"excluded_balance_update_account_numbers"`

		// GeneralCashOutHoldingAccountNumber is "Akun Tampungan General Cash Out"
		GeneralCashOutHoldingAccountNumber string `json:"general_cash_out_holding_account_number"`

		MapAccountEntity map[string]string `json:"mapping_account_entity"`

		AmarthaRevenueAdminFeeAFA                string `json:"amartha_revenue_admin_fee_afa"`
		OperationalReceivableDiscountAdminFeeAFA string `json:"operational_receivable_discount_admin_fee_afa"`

		AccountNumberBankTUPVIForADMFE         map[string]string `json:"account_number_bank_tupvi_for_admfe"`
		AccountNumberBankCOTLRForADMFEByEntity map[string]string `json:"account_number_bank_cotlr_for_admfe_by_entity"`
	}

	MessageBroker struct {
		HTTPPort      int            `json:"http_port"`
		KafkaConsumer ConsumerConfig `json:"kafka_consumer"`
	}

	ConsumerConfig struct {
		Brokers                               []string `json:"brokers"`
		ConsumerGroup                         string   `json:"consumer_group"`
		ConsumerGroupDailyRecon               string   `json:"consumer_group_daily_recon"`
		ConsumerGroupDLQ                      string   `json:"consumer_group_dlq"`
		ConsumerGroupAccountMutation          string   `json:"consumer_group_account_mutation"`
		ConsumerGroupTaskQueueRecon           string   `json:"consumer_group_task_queue_recon"`
		ConsumerGroupDLQRetrier               string   `json:"consumer_group_dlq_retrier"`
		ConsumerGroupBalanceHvt               string   `json:"consumer_group_balance_hvt"`
		ConsumerGroupProcessWalletTransaction string   `json:"consumer_group_process_wallet_transaction"`
		Topic                                 string   `json:"topic"`
		TopicDLQ                              string   `json:"topic_dlq"`
		TopicAccountMutation                  string   `json:"topic_account_mutation"`
		TopicAccountMutationDLQ               string   `json:"topic_account_mutation_dlq"`
		TopicRecon                            string   `json:"topic_recon"`
		TopicAccountingJournal                string   `json:"topic_accounting_journal"`
		TopicTransactionNotification          string   `json:"topic_transaction_notification"`
		TopicBalanceLogs                      string   `json:"topic_balance_logs"`
		TopicBalanceHVT                       string   `json:"topic_balance_hvt"`
		TopicBalanceHvtDLQ                    string   `json:"topic_balance_hvt_dlq"`
		TopicProcessWalletTransaction         string   `json:"topic_process_wallet_transaction"`
		TopicProcessWalletTransactionDLQ      string   `json:"topic_process_wallet_transaction_dlq"`
		Assignor                              string   `json:"assignor"`
		IsOldest                              bool     `json:"is_oldest"`
		IsVerbose                             bool     `json:"is_verbose"`
	}

	CloudStorageConfig struct {
		BaseURL    string `json:"base_url"`
		BucketName string `json:"bucket_name"`
	}

	ExponentialBackOffConfig struct {
		MaxRetries        uint64        `json:"max_retries"`
		MaxBackoffTime    time.Duration `json:"max_backoff_time"`
		BackoffMultiplier float64       `json:"backoff_multiplier"`
	}

	DDDNotificationConfig struct {
		BaseUrl       string `json:"base_url"`
		RetryCount    int    `json:"retry_count"`
		RetryWaitTime int    `json:"retry_wait_time"`
	}
	MasterDataConfig struct {
		BucketName         string `json:"bucket_name"`
		OrderTypeFilePath  string `json:"order_type_file_path"`
		VatRevenueFilePath string `json:"vat_revenue_file_path"`
	}

	ReconEngineConfig struct {
		// ResultURLExpiryTime is the expiry time of the result URL in minutes
		ResultURLExpiryTime int `json:"result_url_expiry_time"`
	}

	HTTPConfiguration struct {
		BaseURL       string        `json:"base_url"`
		SecretKey     string        `json:"secret_key"`
		RetryCount    int           `json:"retry_count"`
		RetryWaitTime int           `json:"retry_wait_time"`
		Timeout       time.Duration `json:"timeout"`
	}

	FeatureConfig struct {
		BalanceRangeMin        float64  `json:"balance_range_min"`
		BalanceRangeMax        float64  `json:"balance_range_max"`
		AllowedNegativeTrxType []string `json:"allowed_negative_trx_type"`
		AllowedTrxType         []string `json:"allowed_trx_type"`
		NegativeBalanceAllowed bool     `json:"negative_balance_allowed"`
		NegativeLimit          float64  `json:"negative_limit"`
	}

	TransactionMigrationConfig struct {
		AllowedAcuanTransactionTypes []string `json:"allowed_acuan_transaction_types"`
	}

	FeatureFlagSDKConfig struct {
		URL             string        `json:"url"`
		Token           string        `json:"token"`
		Env             string        `json:"env"`
		RefreshInterval time.Duration `json:"refresh_interval"`
	}

	FeatureFlagKeyLookup struct {
		IgnoredBalanceCheckAccountNumbers                string `json:"ignored_balance_check_account_numbers"`
		ExcludeConsumeTransactionFromSpecificSubCategory string `json:"exclude_consume_transaction_from_specific_sub_category"`
		AutoCreateAccountIfNotExists                     string `json:"auto_create_account_if_not_exists"`
		ShowOnlyAMFTransactionList                       string `json:"show_only_amf_transaction_list"`
		UseAccountConfigFromExternal                     string `json:"use_account_config_from_external"`
		LceRollout                                       string `json:"lce_rollout"`
		BalanceLimitToggle                               string `json:"balance_limit_toggle"`
	}

	AcuanLibKafkaConfig struct {
		BrokerList        string `json:"broker_list"`
		PartitionStrategy string `json:"partition_strategy"`
	}

	AcuanLibConfig struct {
		Kafka                 AcuanLibKafkaConfig `json:"kafka"`
		SourceSystem          string              `json:"source_system"`
		Topic                 string              `json:"topic"`
		TopicAccounting       string              `json:"topic_accounting"`
		TopUpKey              string              `json:"topup_key"`
		InvestmentKey         string              `json:"investment_key"`
		CashoutKey            string              `json:"cashout_key"`
		DisbursementKey       string              `json:"disbursement_key"`
		DisbursementFailedKey string              `json:"disbursement_failed_key"`
		RepaymentKey          string              `json:"repayment_key"`
		RefundKey             string              `json:"refund_key"`
	}
)
