package faucet

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	faucettypes "github.com/sourcenetwork/sourcehub/app/faucet/types"
	appparams "github.com/sourcenetwork/sourcehub/app/params"
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

		var info faucettypes.FaucetInfoResponse
		err = json.NewDecoder(resp.Body).Decode(&info)
		require.NoError(t, err)

		assert.NotEmpty(t, info.Address)
		assert.NotEmpty(t, info.Balance.Amount)
		assert.NotEmpty(t, info.Balance.Denom)
		assert.Equal(t, int32(0), info.RequestCount, "Initial request count should be 0")
		assert.Equal(t, int64(100000000000000), info.Balance.Amount.Int64(), "Faucet balance should be 100000000000000")
	})

	t.Run("FaucetRequest", func(t *testing.T) {
		testAddress := "source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9"

		request := &faucettypes.FaucetRequest{
			Address: testAddress,
		}
		body, _ := json.Marshal(request)

		httpAddr := network.TCPToHTTP(net.Validators[0].AppConfig.API.Address)
		resp, err := http.Post(
			fmt.Sprintf("%s/faucet/request", httpAddr),
			"application/json",
			bytes.NewBuffer(body),
		)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		require.NoError(t, err)
		defer resp.Body.Close()

		var response faucettypes.FaucetResponse
		bodyBytes, _ := io.ReadAll(resp.Body)
		err = json.NewDecoder(bytes.NewReader(bodyBytes)).Decode(&response)
		require.NoError(t, err)

		assert.NotEmpty(t, response.Txhash)
		assert.Equal(t, uint32(0), response.Code)
		assert.NotEmpty(t, response.Address)
		assert.NotEmpty(t, response.Amount.Amount)

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
			if balanceMap["denom"] == appparams.MicroOpenDenom {
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

		var info faucettypes.FaucetInfoResponse
		err = json.NewDecoder(resp.Body).Decode(&info)
		require.NoError(t, err)

		assert.NotEmpty(t, info.Address)
		assert.NotEmpty(t, info.Balance.Amount)
		assert.NotEmpty(t, info.Balance.Denom)
		assert.Equal(t, int32(1), info.RequestCount, "Request count should be 1")
		assert.Equal(t, int64(99999000000000), info.Balance.Amount.Int64(), "Faucet balance should be 99999000000000")
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

		var info faucettypes.FaucetInfoResponse
		err = json.NewDecoder(resp.Body).Decode(&info)
		require.NoError(t, err)

		assert.NotEmpty(t, info.Address)
		assert.NotEmpty(t, info.Balance.Amount)
		assert.NotEmpty(t, info.Balance.Denom)
		assert.Equal(t, int32(0), info.RequestCount, "Initial request count should be 0")
		assert.Equal(t, int64(100000000000000), info.Balance.Amount.Int64(), "Faucet balance should be 100000000000000")
	})

	t.Run("FaucetInitAccount", func(t *testing.T) {
		testAddress := "source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et"

		request := &faucettypes.InitAccountRequest{
			Address: testAddress,
		}
		body, _ := json.Marshal(request)

		httpAddr := network.TCPToHTTP(net.Validators[0].AppConfig.API.Address)
		resp, err := http.Post(
			fmt.Sprintf("%s/faucet/init-account", httpAddr),
			"application/json",
			bytes.NewBuffer(body),
		)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		require.NoError(t, err)
		defer resp.Body.Close()

		var response faucettypes.InitAccountResponse
		bodyBytes, _ := io.ReadAll(resp.Body)
		err = json.NewDecoder(bytes.NewReader(bodyBytes)).Decode(&response)
		require.NoError(t, err)

		assert.NotEmpty(t, response.Message)
		assert.NotEmpty(t, response.Txhash)
		assert.Equal(t, uint32(0), response.Code)
		assert.NotEmpty(t, response.Address)
		assert.NotEmpty(t, response.Amount.Amount)
		assert.False(t, response.Exists)

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
			if balanceMap["denom"] == appparams.MicroOpenDenom {
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

		var info faucettypes.FaucetInfoResponse
		err = json.NewDecoder(resp.Body).Decode(&info)
		require.NoError(t, err)

		assert.NotEmpty(t, info.Address)
		assert.NotEmpty(t, info.Balance.Amount)
		assert.NotEmpty(t, info.Balance.Denom)
		assert.Equal(t, int32(0), info.RequestCount, "Request count should be 0")
		assert.Equal(t, int64(99999999999999), info.Balance.Amount.Int64(), "Faucet balance should be 99999999999999")
	})
}

func TestFaucetGrantAllowance(t *testing.T) {
	net := network.NewWithOptions(t, network.NetworkOptions{EnableFaucet: true})

	_, err := net.WaitForHeight(1)
	require.NoError(t, err)

	t.Run("FaucetInfoBeforeGrantAllowance", func(t *testing.T) {
		httpAddr := network.TCPToHTTP(net.Validators[0].AppConfig.API.Address)
		resp, err := http.Get(fmt.Sprintf("%s/faucet/info", httpAddr))
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		require.NoError(t, err)
		defer resp.Body.Close()

		var info faucettypes.FaucetInfoResponse
		err = json.NewDecoder(resp.Body).Decode(&info)
		require.NoError(t, err)

		assert.NotEmpty(t, info.Address)
		assert.NotEmpty(t, info.Balance.Amount)
		assert.NotEmpty(t, info.Balance.Denom)
		assert.Equal(t, int64(100000000000000), info.Balance.Amount.Int64(), "Faucet balance should be 100000000000000")
	})

	t.Run("FaucetGrantAllowance", func(t *testing.T) {
		testAddress := "source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et"

		request := &faucettypes.GrantAllowanceRequest{
			Address: testAddress,
		}
		body, _ := json.Marshal(request)

		httpAddr := network.TCPToHTTP(net.Validators[0].AppConfig.API.Address)
		resp, err := http.Post(
			fmt.Sprintf("%s/faucet/grant-allowance", httpAddr),
			"application/json",
			bytes.NewBuffer(body),
		)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		require.NoError(t, err)
		defer resp.Body.Close()

		var response faucettypes.GrantAllowanceResponse
		bodyBytes, _ := io.ReadAll(resp.Body)
		err = json.NewDecoder(bytes.NewReader(bodyBytes)).Decode(&response)
		require.NoError(t, err)

		assert.NotEmpty(t, response.Message)
		assert.NotEmpty(t, response.Txhash)
		assert.Equal(t, uint32(0), response.Code)
		assert.NotEmpty(t, response.Granter)
		assert.NotEmpty(t, response.Grantee)
		assert.NotEmpty(t, response.AmountLimit.Amount)

		_, err = net.WaitForHeight(3)
		require.NoError(t, err)

		// Check feegrant allowances instead of balance
		allowanceResp, err := http.Get(fmt.Sprintf("%s/cosmos/feegrant/v1beta1/allowances/%s", httpAddr, testAddress))
		assert.Equal(t, http.StatusOK, allowanceResp.StatusCode)
		require.NoError(t, err)
		defer allowanceResp.Body.Close()

		var allowanceResponse map[string]interface{}
		err = json.NewDecoder(allowanceResp.Body).Decode(&allowanceResponse)
		require.NoError(t, err)

		assert.Contains(t, allowanceResponse, "allowances")
		allowances := allowanceResponse["allowances"].([]interface{})
		assert.NotEmpty(t, allowances, "Should have at least one allowance")

		allowance := allowances[0].(map[string]interface{})
		assert.Contains(t, allowance, "granter")
		assert.Contains(t, allowance, "grantee")
		assert.Contains(t, allowance, "allowance")

		allowanceData := allowance["allowance"].(map[string]interface{})
		assert.Contains(t, allowanceData, "@type")
		assert.Contains(t, allowanceData, "spend_limit")

		spendLimit := allowanceData["spend_limit"].([]interface{})
		assert.NotEmpty(t, spendLimit, "Should have spend limit")

		spendLimitCoin := spendLimit[0].(map[string]interface{})
		assert.Equal(t, appparams.MicroOpenDenom, spendLimitCoin["denom"])
		assert.Equal(t, "10000000000", spendLimitCoin["amount"], "Allowance should have 10,000 OPEN spend limit")
	})

	t.Run("FaucetInfoAfterGrantAllowance", func(t *testing.T) {
		httpAddr := network.TCPToHTTP(net.Validators[0].AppConfig.API.Address)
		resp, err := http.Get(fmt.Sprintf("%s/faucet/info", httpAddr))
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		require.NoError(t, err)
		defer resp.Body.Close()

		var info faucettypes.FaucetInfoResponse
		err = json.NewDecoder(resp.Body).Decode(&info)
		require.NoError(t, err)

		assert.NotEmpty(t, info.Address)
		assert.NotEmpty(t, info.Balance.Amount)
		assert.NotEmpty(t, info.Balance.Denom)
		assert.Equal(t, int64(100000000000000), info.Balance.Amount.Int64(), "Faucet balance should still be 100000000000000")
	})
}
