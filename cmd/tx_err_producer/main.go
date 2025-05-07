package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/segmentio/kafka-go"
	"github.com/spf13/cobra"

	"github.com/sourcenetwork/sourcehub/sdk"
)

const (
	EnvMetricsPort    string = "METRICS_PORT"
	EnvCometRpcAddr          = "COMET_RPC_ADDR"
	EnvKafkaAddr             = "KAFKA_ADDR"
	EnvKafkaTopic            = "KAFKA_TOPIC"
	EnvChainID               = "CHAIN_ID"
	EnvLogIncomingTxs        = "LOG_TXS"
	DefaultPort              = "9001"
	DefaultRpcAddr           = "tcp://localhost:26657"
)

const (
	metricNamespace string = "sourcehub"
	metricSubsystem        = "tx_err_log"
	hostLabel              = "host"
	chainIdLabel           = "chain_id"
)

var labels = []string{hostLabel, chainIdLabel}

var kafkaPartitionKey []byte = nil

// Config models the set of parameters
// the system uses
type Config struct {
	MetricsPort  string
	CometRPCAddr string
	// kafka address to connect to
	KafkaAddr string
	// Name of Kafka topic to listen to
	KafkaTopic string
	ChainID    string
	LogTxs     bool
}

// configHelp maps an envrionment variable name to its description
var configHelp map[string]string = map[string]string{
	EnvMetricsPort:    "Port number from which metrics will be available. Default 9001",
	EnvCometRpcAddr:   "Address of the SourceHub to connect to. Must contain port to Comet RPC Api. Default: tcp://localhost:26657",
	EnvKafkaAddr:      "Address of Kafka node",
	EnvKafkaTopic:     "Name of the target Kafka topic",
	EnvChainID:        "Chain Identifier",
	EnvLogIncomingTxs: "Boolean flag indicating whether to log incoming txs. If set will log all incoming txs. Default not set",
}

func fmtConfigHelp() string {
	builder := strings.Builder{}
	for key, value := range configHelp {
		builder.WriteString(key)
		builder.WriteString(": ")
		builder.WriteString(value)
		builder.WriteString("\n")
	}
	return builder.String()
}

var rootCmd = &cobra.Command{
	Use:   "tx_err_producer",
	Short: "tx_err_producer listens to SourceHub Tx events and pushes it to Source's event infrastructre.",
	Long: fmt.Sprintf(
		`tx_err_producer is a cli utility which connects to SourceHub's cometbft rpc connection and listens for Tx processing events.
The received events are expanded and the Tx results are unmarshaled into the correct Msg response types.
If a Tx failed, the error log is written to a Source managed kafka queue managed, which is part of the internal logging infrastructure.
Additionally, the forwader can print the tx results to stdout.

The following configuration options are available through environment vars:

%v
	`, fmtConfigHelp()),
	Args: cobra.ExactArgs(0),
	Run:  entrypoint,
}

// newConfigFromEnv creates a Config object by fetching values from the environment.
// Returns an error if required variables aren't set
func newConfigFromEnv() (Config, error) {
	port := os.Getenv(EnvMetricsPort)
	if port == "" {
		port = DefaultPort
	}
	cometAddr := os.Getenv(EnvCometRpcAddr)
	if cometAddr == "" {
		cometAddr = DefaultRpcAddr
	}
	kafkaAddr := os.Getenv(EnvKafkaAddr)
	if kafkaAddr == "" {
		return Config{}, fmt.Errorf("missing env var: %v", EnvKafkaAddr)
	}
	topic := os.Getenv(EnvKafkaTopic)
	if topic == "" {
		return Config{}, fmt.Errorf("missing env var: %v", EnvKafkaTopic)
	}

	chainID := os.Getenv(EnvChainID)
	if topic == "" {
		return Config{}, fmt.Errorf("missing env var: %v", EnvChainID)
	}

	log := os.Getenv(EnvLogIncomingTxs) != ""

	return Config{
		MetricsPort:  port,
		CometRPCAddr: cometAddr,
		KafkaAddr:    kafkaAddr,
		KafkaTopic:   topic,
		ChainID:      chainID,
		LogTxs:       log,
	}, nil
}

// metrics models set of metrics collected by the application
type metrics struct {
	// number of errors encountered while processing msgs
	errorsCounter *prometheus.CounterVec
	// total number of tx events received
	txCounter *prometheus.CounterVec
	// msgCounter is the number of messages written to Kafka
	msgCounter *prometheus.CounterVec
	// set of static labels appended to all measurements
	labels prometheus.Labels
}

func main() {
	rootCmd.Execute()
}

func entrypoint(cmd *cobra.Command, args []string) {
	config, err := newConfigFromEnv()
	if err != nil {
		log.Fatal(err)
	}

	reg := prometheus.NewRegistry()
	m, err := newMetrics(config, reg)

	if err != nil {
		log.Fatalf("initializing metrics: %v", err)
	}

	http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{Registry: reg}))
	metricsAddr := fmt.Sprintf("0.0.0.0:%v", config.MetricsPort)
	go http.ListenAndServe(metricsAddr, nil)
	log.Printf("served metrics in /metrics: %v", metricsAddr)

	// Writer automatically batches writes by a set timeout
	w := &kafka.Writer{
		Addr:         kafka.TCP(config.KafkaAddr),
		Topic:        config.KafkaTopic,
		MaxAttempts:  5,
		BatchTimeout: time.Millisecond * 500,
		Async:        false,
	}

	opts := []sdk.Opt{sdk.WithCometRPCAddr(config.CometRPCAddr)}
	client, err := sdk.NewClient(opts...)
	if err != nil {
		log.Fatalf("error connecting to SourceHub node: %v", err)
	}

	listener := client.TxListener()

	ctx := context.Background()
	ch, errCh, err := listener.ListenTxs(ctx)
	defer listener.Close()
	if err != nil {
		log.Fatalf("initializing listener", err)
	}

	for {
		select {
		case result := <-ch:
			m.txCounter.With(m.labels).Inc()
			log.Print("received event")
			if result.Code != 0 {
				writeMsg(ctx, m, w, result)
			}
			if config.LogTxs {
				bytes, err := json.MarshalIndent(result, "", "  ")
				if err != nil {
					continue
				}
				log.Print(string(bytes))
			}
		case err := <-errCh:
			m.errorsCounter.With(m.labels).Inc()
			log.Printf("ERROR in Tx: %v", err)
		case <-listener.Done():
			log.Printf("Client terminated")
			return
		case <-ctx.Done():
			log.Printf("Ctx terminated")
			return
		}
	}
}

// errorLog models a tx error
type errorLog struct {
	TxHash    string `json:"tx_hash"`
	Height    int64  `json:"height"`
	MsgIndex  uint32 `json:"msg_index"`
	Codespace string `json:"code_space"`
	Code      uint32 `json:"code"`
	Log       string `json:"log"`
	GasWanted int64  `json:"gas_wanted"`
	GasUsed   int64  `json:"gas_used"`
}

// writeMsgs converts a tx event into an instance of error log,
// mrshals it into a Json and writes it to kafka
func writeMsg(ctx context.Context, m *metrics, w *kafka.Writer, tx sdk.Event) {
	hasher := sha256.New()
	hasher.Write(tx.Tx)
	hash := hex.EncodeToString(hasher.Sum(nil))
	ev := errorLog{
		TxHash:    hash,
		Height:    tx.Height,
		MsgIndex:  tx.Index,
		Codespace: tx.Codespace,
		Code:      tx.Code,
		Log:       tx.Log,
		GasWanted: tx.GasWanted,
		GasUsed:   tx.GasUsed,
	}
	bz, err := json.Marshal(ev)
	if err != nil {
		log.Printf("Error marshaling tx %v: %v", hash, err)
	}
	msg := kafka.Message{
		Key:   kafkaPartitionKey,
		Value: bz,
	}

	err = w.WriteMessages(ctx, msg)
	if err != nil {
		log.Printf("failed writing messages: %v", err)
		m.errorsCounter.With(m.labels).Inc()
	} else {
		m.msgCounter.With(m.labels).Inc()
	}
}

// newMetrics creates a new instance the instrument set object
func newMetrics(config Config, reg *prometheus.Registry) (*metrics, error) {
	host, err := os.Hostname()
	if err != nil {
		return nil, err
	}

	errorsCounter := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metricNamespace,
		Subsystem: metricSubsystem,
		Name:      "errors_total",
		Help:      "Total number of errors encountered while processing a tx result",
	}, labels)

	txCounter := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metricNamespace,
		Subsystem: metricSubsystem,
		Name:      "tx_total",
		Help:      "Total number of txs received",
	}, labels)

	msgCounter := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metricNamespace,
		Subsystem: metricSubsystem,
		Name:      "msg_total",
		Help:      "Total number of msgs written to Kafka",
	}, labels)

	reg.MustRegister(errorsCounter)
	reg.MustRegister(txCounter)
	reg.MustRegister(msgCounter)

	m := metrics{
		errorsCounter: errorsCounter,
		txCounter:     txCounter,
		msgCounter:    msgCounter,
		labels: prometheus.Labels{
			hostLabel:    host,
			chainIdLabel: config.ChainID,
		},
	}
	return &m, nil
}
