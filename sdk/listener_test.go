package sdk

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_channelMapper_Success(t *testing.T) {
	ch := make(chan int, 10)

	mapper := func(i int) (string, error) {
		return fmt.Sprintf("%v", i), nil
	}
	vals, errs, _ := channelMapper(ch, mapper)

	ch <- 1
	got := <-vals
	require.Equal(t, "1", got)

	select {
	case <-errs:
		t.Fatalf("unnexpected error in errors channel")
	default:
	}
}

func Test_channelMapper_Err(t *testing.T) {
	ch := make(chan int, 10)

	err := errors.New("err")
	mapper := func(_ int) (string, error) {
		return "", err
	}
	vals, errs, _ := channelMapper(ch, mapper)

	ch <- 1

	gotErr := <-errs
	require.Equal(t, err, gotErr)

	select {
	case <-vals:
		t.Fatalf("unnexpected value in values channel")
	default:
	}
}
