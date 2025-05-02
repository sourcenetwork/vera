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
	DefaultRpcAddr           = "localhost:26657"
)

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

var configHelp map[string]string = map[string]string{
	EnvMetricsPort:    "Port number from which metrics will be available. Default 9001",
	EnvCometRpcAddr:   "Address of the SourceHub to connect to. Must contain port to Comet RPC Api. Default: localhost:26657",
	EnvKafkaAddr:      "Address of Kafka node to connect to",
	EnvKafkaTopic:     "Name of the target Kafka topic",
	EnvChainID:        "Chain Identifier",
	EnvLogIncomingTxs: "Boolean flag indicating whether to log incoming txs. If set will log all incoming txs. Default not set",
}

var rootCmd = &cobra.Command{
	Use:   "msg-log-forwader",
	Short: "msg-log-forwader is an utility to forward SourceHub message logs.",
	Long: fmt.Sprintf(
		`msg-log-forwarder is a cli utility which connects to SourceHub's cometbft rpc connection
	and listens for Tx processing events.
	The received events are expanded and the Tx results are unmarshaled into the correct
	Msg response types.
	If a Tx failed, the error log is written to a Source managed kafka queue managed, which is part of
	the internal logging infrastructure.
	Additionally, the forwader can print the tx results to stdout.

	The following configuration options are available through environment vars:
	%v
	`, configHelp),
	Args: cobra.ExactArgs(0),
	Run:  entrypoint,
}

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
		return Config{}, fmt.Errorf("missing env var: %w", EnvKafkaAddr)
	}
	topic := os.Getenv(EnvKafkaTopic)
	if topic == "" {
		return Config{}, fmt.Errorf("missing env var: %w", EnvKafkaTopic)
	}

	chainID := os.Getenv(EnvChainID)
	if topic == "" {
		return Config{}, fmt.Errorf("missing env var: %w", EnvChainID)
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

type metrics struct {
	errorsCounter *prometheus.CounterVec
	txCounter     *prometheus.CounterVec
	msgCounter    *prometheus.CounterVec
	labels        prometheus.Labels
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

	w := &kafka.Writer{
		Addr:         kafka.TCP(config.KafkaAddr),
		Topic:        config.KafkaTopic,
		MaxAttempts:  5,
		BatchTimeout: time.Millisecond * 500,
		Async:        true,
		Completion: func(messages []kafka.Message, err error) {
			if err != nil {
				log.Printf("failed writing messages: %v", err)
				m.errorsCounter.With(m.labels).Inc()
			} else {
				m.msgCounter.With(m.labels).Add(float64(len(messages)))
				log.Printf("wrriten %v msgs", len(messages))
			}
		},
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

type event struct {
	TxHash    string `json:"tx_hash"`
	Height    int64  `json:"height"`
	MsgIndex  uint32 `json:"msg_index"`
	Codespace string `json:"code_space"`
	Code      uint32 `json:"code"`
	Log       string `json:"log"`
	GasWanted int64  `json:"gas_wanted"`
	GasUsed   int64  `json:"gas_used"`
}

func writeMsg(ctx context.Context, m *metrics, w *kafka.Writer, tx sdk.Event) {
	var key []byte = []byte("") // key define the partition key - not a unique key as in kv stores
	hasher := sha256.New()
	hasher.Write(tx.Tx)
	hash := hex.EncodeToString(hasher.Sum(nil))
	ev := event{
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
		Key:   key,
		Value: bz,
	}

	w.WriteMessages(ctx, msg) // no error since async
}

func newMetrics(config Config, reg *prometheus.Registry) (*metrics, error) {
	host, err := os.Hostname()
	if err != nil {
		return nil, err
	}

	labels := []string{
		"host",
		"chain_id",
	}

	errorsCounter := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "sourcehub",
		Subsystem: "tx_log_fwd",
		Name:      "errors_total",
		Help:      "Total number of errors encountered while processing a tx result",
	}, labels)

	txCounter := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "sourcehub",
		Subsystem: "tx_log_fwd",
		Name:      "tx_total",
		Help:      "Total number of txs received",
	}, labels)

	msgCounter := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "sourcehub",
		Subsystem: "tx_log_fwd",
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
			"host":     host,
			"chain_id": config.ChainID,
		},
	}
	return &m, nil
}
