package docker

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"os"
	"path"
	"testing"
	"time"

	"github.com/docker/docker/client"
	"github.com/sourcenetwork/vera/sdk"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tclog "github.com/testcontainers/testcontainers-go/log"
)

var (
	consensusKeyFileName = "priv_validator_key.json"
	p2pKeyFileName       = "node_key.json"
	genesisFileName      = "genesis.json"
	mnemonicFileName     = "mnemonic"
	containerBaseDir     = "/home/node"
	veraImage            = "ghcr.io/sourcenetwork/vera:dev"
)

var (
	//go:embed files/node_key.json
	p2pKeyData string
	//go:embed files/priv_validator_key.json
	consensusKeyData string
	//go:embed files/genesis.json
	genesisData string
	//go:embed files/mnemonic
	mnemonicData string
)

// Test Docker containers correctly executes and produces blocks
// The docker image should be previously built with `make docker“
func Test_DockerContainer_Starts(t *testing.T) {
	ctx := context.Background()
	requireLocalImage(t, ctx, veraImage)

	// write the require state files
	// (consensus key, p2p key, acc key, genesis)
	tmpDir := t.TempDir()
	fileMap := map[string]string{
		p2pKeyFileName:       p2pKeyData,
		consensusKeyFileName: consensusKeyData,
		genesisFileName:      genesisData,
		mnemonicFileName:     mnemonicData,
	}
	for fileName, data := range fileMap {
		path := path.Join(tmpDir, fileName)
		t.Logf("writting file to tmp dir: %v", path)
		err := os.WriteFile(path, []byte(data), 0x777)
		require.NoError(t, err)
	}

	// prepare container opts
	var containerFiles []testcontainers.ContainerFile
	for fileName := range fileMap {
		f := testcontainers.ContainerFile{
			HostFilePath:      path.Join(tmpDir, fileName),
			ContainerFilePath: path.Join(containerBaseDir, fileName),
			FileMode:          0x777,
		}
		containerFiles = append(containerFiles, f)
	}

	// create test container for the default config
	testLogger := tclog.TestLogger(t)
	container, err := testcontainers.Run(
		ctx,
		veraImage,
		testcontainers.WithFiles(containerFiles...),
		testcontainers.WithEnv(map[string]string{
			"CHAIN_ID":            "test",
			"MONIKER":             "moniker",
			"GENESIS_PATH":        path.Join(containerBaseDir, genesisFileName),
			"MNEMONIC_PATH":       path.Join(containerBaseDir, mnemonicFileName),
			"CONSENSUS_KEY_PATH":  path.Join(containerBaseDir, consensusKeyFileName),
			"COMET_NODE_KEY_PATH": path.Join(containerBaseDir, p2pKeyFileName),
		}),
		testcontainers.WithExposedPorts("26657/tcp"),
		testcontainers.WithExposedPorts("9090/tcp"),
		testcontainers.WithLogger(testLogger),
	)
	require.NoError(t, err)

	t.Cleanup(func() {
		t.Helper()
		logs, err := container.Logs(ctx)
		if err != nil {
			t.Logf("could not read container logs")
		} else {
			buf := bytes.Buffer{}
			// errors during cleanup don't affect anything
			buf.ReadFrom(logs)
			t.Logf("container logs: %v", buf.String())
			logs.Close()
		}
		testcontainers.TerminateContainer(container)
	})

	// probe the container until it is producing blocks
	grpcEndpoint, err := container.PortEndpoint(ctx, "9090", "")
	require.NoError(t, err)
	rpcEndpoint, err := container.PortEndpoint(ctx, "26657", "tcp")
	require.NoError(t, err)
	t.Logf("vera endpoints: grpc=%v, rpc=%v", grpcEndpoint, rpcEndpoint)
	err = waitForChain(t, grpcEndpoint, rpcEndpoint)
	require.NoError(t, err)
}

func requireLocalImage(t testing.TB, ctx context.Context, image string) {
	t.Helper()

	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		t.Skipf("docker is unavailable: %v", err)
	}
	defer dockerClient.Close()

	if _, err := dockerClient.ImageInspect(ctx, image); err != nil {
		t.Skipf("%s is not available locally; run `make docker` before this smoke test: %v", image, err)
	}
}

func waitForChain(t testing.TB, grpcEndpoint, cometRpcEndpoint string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	i := 1
	startTs := time.Now()
	for {
		// use an exponential backoff timer to adjust polling
		timer := time.After(time.Duration(i) * (10 * time.Millisecond))
		i++
		select {
		case <-ctx.Done():
			t.Logf("time out waiting for chain to start")
			return fmt.Errorf("error setting up chain: connection not ready after deadline")
		case <-timer:
			ok := probeChain(t, ctx, grpcEndpoint, cometRpcEndpoint)
			if ok {
				elapsed := time.Since(startTs)
				t.Logf("chain ready to receive connections: after %v", elapsed)
				return nil
			}
		}
	}
}

// probeChain is a readiness probe which tries to connect to Vera's
// RPC endpoint to determine if it is ready to receive connections.
// Returns true if the probe succeeded.
func probeChain(t testing.TB, ctx context.Context, grpcAddr, cometRpcAddr string) bool {
	client, err := sdk.NewClient(sdk.WithGRPCAddr(grpcAddr), sdk.WithCometRPCAddr(cometRpcAddr))
	if err != nil {
		t.Logf("chain probe failed: %v", err)
		return false
	}
	defer client.Close()

	// probe rpc service
	height := int64(1)
	_, err = client.CometBFTRPCClient().Block(ctx, &height)
	if err != nil {
		t.Logf("chain probe failed: %v", err)
		return false
	}
	return true
}
