package faucet

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sourcenetwork/sourcehub/testutil/network"
)

func TestFaucetRequest(t *testing.T) {
	net := network.NewWithOptions(t, network.NetworkOptions{EnableFaucet: true})

	_, err := net.WaitForHeight(1)
	require.NoError(t, err)

	t.Run("FaucetInfoBeforeRequest", func(t *testing.T) {
		httpAddr := network.TCPToHTTP(net.Validators[0].AppConfig.API.Address)
		resp, err := http.Get(fmt.Sprintf("%s/faucet/info", httpAddr))
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		require.NoError(t, err)
		defer resp.Body.Close()

		var info map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&info)
		require.NoError(t, err)

		assert.Contains(t, info, "address")
		assert.Contains(t, info, "balance")
		assert.Contains(t, info, "request_count")

		requestCount := info["request_count"].(float64)
		assert.Equal(t, float64(0), requestCount, "Initial request count should be 0")
		balance := info["balance"].(map[string]interface{})
		assert.Contains(t, balance, "amount")
		balanceAmount := balance["amount"].(string)
		balanceValue, err := strconv.ParseInt(balanceAmount, 10, 64)
		require.NoError(t, err)
		assert.Equal(t, balanceValue, int64(100000000000000), "Faucet balance should be 100000000000000")
	})

	t.Run("FaucetRequest", func(t *testing.T) {
		testAddress := "source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9"

		requestBody := map[string]string{
			"address": testAddress,
		}
		body, _ := json.Marshal(requestBody)

		httpAddr := network.TCPToHTTP(net.Validators[0].AppConfig.API.Address)
		resp, err := http.Post(
			fmt.Sprintf("%s/faucet/request", httpAddr),
			"application/json",
			bytes.NewBuffer(body),
		)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		require.NoError(t, err)
		defer resp.Body.Close()

		var response map[string]interface{}
		bodyBytes, _ := io.ReadAll(resp.Body)
		err = json.NewDecoder(bytes.NewReader(bodyBytes)).Decode(&response)
		require.NoError(t, err)

		assert.Contains(t, response, "txhash")
		assert.Contains(t, response, "code")
		assert.Contains(t, response, "address")
		assert.Contains(t, response, "amount")
		assert.Equal(t, float64(0), response["code"])

		_, err = net.WaitForHeight(3)
		require.NoError(t, err)

		balanceResp, err := http.Get(fmt.Sprintf("%s/cosmos/bank/v1beta1/balances/%s", httpAddr, testAddress))
		assert.Equal(t, http.StatusOK, balanceResp.StatusCode)
		require.NoError(t, err)
		defer balanceResp.Body.Close()

		var balanceResponse map[string]interface{}
		err = json.NewDecoder(balanceResp.Body).Decode(&balanceResponse)
		require.NoError(t, err)

		assert.Contains(t, balanceResponse, "balances")
		balances := balanceResponse["balances"].([]interface{})

		var uopenBalance string
		for _, balance := range balances {
			balanceMap := balance.(map[string]interface{})
			if balanceMap["denom"] == "uopen" {
				uopenBalance = balanceMap["amount"].(string)
				break
			}
		}

		assert.Equal(t, "1000000000", uopenBalance, "Account should have received exactly 1000000000uopen")
	})

	t.Run("FaucetInfoAfterRequest", func(t *testing.T) {
		httpAddr := network.TCPToHTTP(net.Validators[0].AppConfig.API.Address)
		resp, err := http.Get(fmt.Sprintf("%s/faucet/info", httpAddr))
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		require.NoError(t, err)
		defer resp.Body.Close()

		var info map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&info)
		require.NoError(t, err)

		assert.Contains(t, info, "address")
		assert.Contains(t, info, "balance")
		assert.Contains(t, info, "request_count")

		requestCount := info["request_count"].(float64)
		assert.Equal(t, float64(1), requestCount, "Request count should be 1")

		balance := info["balance"].(map[string]interface{})
		assert.Contains(t, balance, "amount")
		balanceAmount := balance["amount"].(string)
		balanceValue, err := strconv.ParseInt(balanceAmount, 10, 64)
		require.NoError(t, err)
		assert.Equal(t, balanceValue, int64(99999000000000), "Faucet balance should be 99999000000000")
	})
}

func TestFaucetInitAccount(t *testing.T) {
	net := network.NewWithOptions(t, network.NetworkOptions{EnableFaucet: true})

	_, err := net.WaitForHeight(1)
	require.NoError(t, err)

	t.Run("FaucetInfoBeforeInitAccount", func(t *testing.T) {
		httpAddr := network.TCPToHTTP(net.Validators[0].AppConfig.API.Address)
		resp, err := http.Get(fmt.Sprintf("%s/faucet/info", httpAddr))
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		require.NoError(t, err)
		defer resp.Body.Close()

		var info map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&info)
		require.NoError(t, err)

		assert.Contains(t, info, "address")
		assert.Contains(t, info, "balance")
		assert.Contains(t, info, "request_count")

		requestCount := info["request_count"].(float64)
		assert.Equal(t, float64(0), requestCount, "Initial request count should be 0")

		balance := info["balance"].(map[string]interface{})
		assert.Contains(t, balance, "amount")
		balanceAmount := balance["amount"].(string)
		balanceValue, err := strconv.ParseInt(balanceAmount, 10, 64)
		require.NoError(t, err)
		assert.Equal(t, balanceValue, int64(100000000000000), "Faucet balance should be 100000000000000")
	})

	t.Run("FaucetInitAccount", func(t *testing.T) {
		testAddress := "source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et"

		requestBody := map[string]string{
			"address": testAddress,
		}
		body, _ := json.Marshal(requestBody)

		httpAddr := network.TCPToHTTP(net.Validators[0].AppConfig.API.Address)
		resp, err := http.Post(
			fmt.Sprintf("%s/faucet/init-account", httpAddr),
			"application/json",
			bytes.NewBuffer(body),
		)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		require.NoError(t, err)
		defer resp.Body.Close()

		var response map[string]interface{}
		bodyBytes, _ := io.ReadAll(resp.Body)
		err = json.NewDecoder(bytes.NewReader(bodyBytes)).Decode(&response)
		require.NoError(t, err)

		assert.Contains(t, response, "message")
		assert.Contains(t, response, "txhash")
		assert.Contains(t, response, "code")
		assert.Contains(t, response, "address")
		assert.Contains(t, response, "amount")
		assert.Contains(t, response, "exists")
		assert.Equal(t, float64(0), response["code"])

		_, err = net.WaitForHeight(3)
		require.NoError(t, err)

		balanceResp, err := http.Get(fmt.Sprintf("%s/cosmos/bank/v1beta1/balances/%s", httpAddr, testAddress))
		assert.Equal(t, http.StatusOK, balanceResp.StatusCode)
		require.NoError(t, err)
		defer balanceResp.Body.Close()

		var balanceResponse map[string]interface{}
		err = json.NewDecoder(balanceResp.Body).Decode(&balanceResponse)
		require.NoError(t, err)

		assert.Contains(t, balanceResponse, "balances")
		balances := balanceResponse["balances"].([]interface{})

		var uopenBalance string
		for _, balance := range balances {
			balanceMap := balance.(map[string]interface{})
			if balanceMap["denom"] == "uopen" {
				uopenBalance = balanceMap["amount"].(string)
				break
			}
		}

		assert.Equal(t, "1", uopenBalance, "Account should have received exactly 1uopen")
	})

	t.Run("FaucetInfoAfterInitAccount", func(t *testing.T) {
		httpAddr := network.TCPToHTTP(net.Validators[0].AppConfig.API.Address)
		resp, err := http.Get(fmt.Sprintf("%s/faucet/info", httpAddr))
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		require.NoError(t, err)
		defer resp.Body.Close()

		var info map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&info)
		require.NoError(t, err)

		assert.Contains(t, info, "address")
		assert.Contains(t, info, "balance")
		assert.Contains(t, info, "request_count")

		requestCount := info["request_count"].(float64)
		assert.Equal(t, float64(0), requestCount, "Request count should be 0")

		balance := info["balance"].(map[string]interface{})
		assert.Contains(t, balance, "amount")
		balanceAmount := balance["amount"].(string)
		balanceValue, err := strconv.ParseInt(balanceAmount, 10, 64)
		require.NoError(t, err)
		assert.Equal(t, balanceValue, int64(99999999999999), "Faucet balance should be 99999999999999")
	})
}
