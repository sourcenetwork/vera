package signed_policy_cmd

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// reject jws with critical header: If any of the listed extension Header Parameters are not understood
//and supported by the recipient, then the JWS is invalid
// reject jws with jose kid, x5c, x5u, kid, jwk, jku

// test jws containing kid and signed with kids key is rejected

// test did no verification method is rejected

func TestComputePayloadID_ValidJWS(t *testing.T) {
	jws1 := `{"payload":"eyJhY3RvciI6ImRpZDprZXk6ejZNa3JNZGthUFFlWGZCUGJLelkyYlhNNkE1UktxUHF1cDgxeDE1UEdnQVJZRVVaIiwiaXNzdWVkSGVpZ2h0IjoiMSIsImlzc3VlZEF0IjoiMjAyNS0wOC0xOVQxMjoxODo1NS4yOTI3NDdaIiwiZXhwaXJhdGlvbkRlbHRhIjoiNDMyMDAiLCJwb2xpY3lJZCI6ImRhN2JlNjUwMjc2NjQ3MDg1NTFmOTcxOTdiYTVmNTk5M2FhOTliYzdiNTcwNTVkZjk3NjY0MjZkYzZkYTk2MDUiLCJjbWQiOnsicmVnaXN0ZXJPYmplY3RDbWQiOnsib2JqZWN0Ijp7InJlc291cmNlIjoiZmlsZSIsImlkIjoiZm9vIn19fX0","protected":"eyJhbGciOiJFZERTQSJ9","signature":"XJjL-DwobYDKmvwWDovTu1TbGbdr355hTzHJsXaOYnfim1wdlVyggGkAr1y10xB_zRf_k7CHZSLrh98lZhuyCw"}`

	jws2 := `
	{"payload":"eyJhY3RvciI6ImRpZDprZXk6ejZNa2lvRTR6ejcydjk1SERFRjgxTkFrQjVwbllyalVzYUFhSzhCUDdhQWFlaEZuIiwiaXNzdWVkSGVpZ2h0IjoiMSIsImlzc3VlZEF0IjoiMjAyNS0wOC0xOVQxMjo1NzoxMS4wNzUxNzhaIiwiZXhwaXJhdGlvbkRlbHRhIjoiNDMyMDAiLCJwb2xpY3lJZCI6IjgxZDUwNTU5YzUyZjM5YWNiOTc4NGRkY2I3ZmM1MTY2ZWRjZDczMmQ3NzU4Y2QyMjQ2NWE4NjhjNGI5NmJiNmIiLCJjbWQiOnsicmVnaXN0ZXJPYmplY3RDbWQiOnsib2JqZWN0Ijp7InJlc291cmNlIjoiZmlsZSIsImlkIjoiZm9vIn19fX0","protected":"eyJhbGciOiJFZERTQSJ9","signature":"5WKgw-80H6HTH5-CVBWykyFKHMjz1tEeIJNbuWJOHNlEM9nd8FfEpOuR0Ha6iHwsyviqcpfUXF8_Jqy2nou-Dg"}`

	id1 := ComputePayloadID(jws1)
	require.NotEmpty(t, id1)

	id2 := ComputePayloadID(jws2)
	require.NotEmpty(t, id2)

	id3 := ComputePayloadID(jws1)
	require.NotEmpty(t, id3)

	require.Equal(t, id1, id3)
	require.NotEqual(t, id1, id2)
}

func TestComputePayloadID_EmptyInput(t *testing.T) {
	id := ComputePayloadID("")
	require.NotEmpty(t, id)
}

func TestComputePayloadID_InvalidJWS(t *testing.T) {
	invalidJSON := `{"payload": eyJhY3RvciI6ImRp}`
	id := ComputePayloadID(invalidJSON)
	require.NotEmpty(t, id)
}

func TestComputePayloadID_MalformedJWS(t *testing.T) {
	malformedJWS := `{"payload": "test"}`
	id := ComputePayloadID(malformedJWS)
	require.NotEmpty(t, id)
}
