package keeper

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"cosmossdk.io/math"

	"github.com/sourcenetwork/sourcehub/x/tier/types"
)

func TestCalculateCredit(t *testing.T) {
	rateList := []types.Rate{
		{Amount: math.NewInt(300), Rate: 150},
		{Amount: math.NewInt(200), Rate: 120},
		{Amount: math.NewInt(100), Rate: 110},
		{Amount: math.NewInt(0), Rate: 100},
	}

	tests := []struct {
		lockedAmt  int64
		lockingAmt int64
		want       int64
	}{
		{
			lockedAmt:  100,
			lockingAmt: 0,
			want:       0,
		},
		{
			lockedAmt:  250,
			lockingAmt: 0,
			want:       0,
		},
		{
			lockedAmt:  0,
			lockingAmt: 100,
			want:       100,
		},
		{
			lockedAmt:  0,
			lockingAmt: 200,
			want:       (100 * 1.0) + (100 * 1.1),
		},
		{
			lockedAmt:  0,
			lockingAmt: 250,
			want:       (100 * 1.0) + (100 * 1.1) + (50 * 1.2),
		},
		{
			lockedAmt:  0,
			lockingAmt: 300,
			want:       (100 * 1.0) + (100 * 1.1) + (100 * 1.2),
		},
		{
			lockedAmt:  0,
			lockingAmt: 350,
			want:       (100 * 1.0) + (100 * 1.1) + (100 * 1.2) + (50 * 1.5),
		},
		{
			lockedAmt:  0,
			lockingAmt: 600,
			want:       (100 * 1.0) + (100 * 1.1) + (100 * 1.2) + (300 * 1.5),
		},
		{
			lockedAmt:  100,
			lockingAmt: 100,
			want:       (100 * 1.1),
		},
		{
			lockedAmt:  200,
			lockingAmt: 100,
			want:       (100 * 1.2),
		},
		{
			lockedAmt:  150,
			lockingAmt: 150,
			want:       (50 * 1.1) + (100 * 1.2),
		},
		{
			lockedAmt:  50,
			lockingAmt: 400,
			want:       (50 * 1.0) + (100 * 1.1) + (100 * 1.2) + (150 * 1.5),
		},
	}
	for _, tt := range tests {
		name := fmt.Sprintf("%d adds %d", tt.lockedAmt, tt.lockingAmt)
		oldLock := math.NewInt(tt.lockedAmt)
		newLock := math.NewInt(tt.lockingAmt)
		want := math.NewInt(tt.want)

		t.Run(name, func(t *testing.T) {
			if got := calculateCredit(rateList, oldLock, newLock); !reflect.DeepEqual(got, want) {
				t.Errorf("calculateCredit() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCalculateProratedCredit(t *testing.T) {
	baseTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name           string
		lockedAmt      int64
		lockingAmt     int64
		epochStartTime time.Time
		epochDuration  time.Duration
		expectedCredit int64
	}{
		{
			/*
				Default rate for locking 300+ tokens is 150, but since the reward rates are tiered:
				- 0-100 tokens are prorated at a rate of 1.0 (100%).
				- 100-200 tokens are prorated at a rate of 1.1 (110%).
				- 200-300 tokens are prorated at a rate of 1.2 (120%).
				- 300+ tokens are prorated at a rate of 1.5 (150%).
				For a locking amount of 10,000,000, the total is 14,999,550 + 100 + 110 + 120 = 14,999,880.
				After 2 minutes in a 5-minute epoch, the expected credit amount should be:
				14,999,880 * 120,000 ms / 300,000 ms = 14,999,880 * 0.4 = 5,999,952.
			*/
			name:           "Large amount in the middle of the epoch",
			lockedAmt:      0,
			lockingAmt:     10_000_000,
			epochStartTime: baseTime.Add(-2 * time.Minute),
			epochDuration:  time.Minute * 5,
			expectedCredit: 5_999_952,
		},
		{
			name:           "Large amount at the first second of the epoch",
			lockedAmt:      0,
			lockingAmt:     10_000_000,
			epochStartTime: baseTime.Add(-5 * time.Minute).Add(1 * time.Second),
			epochDuration:  time.Minute * 5,
			expectedCredit: 14_949_880,
		},
		{
			name:           "Large amount at the last second of the epoch",
			lockedAmt:      0,
			lockingAmt:     10_000_000,
			epochStartTime: baseTime.Add(-1 * time.Second),
			epochDuration:  time.Minute * 5,
			expectedCredit: 49_999,
		},
		{
			name:           "At the beginning of an epoch",
			lockedAmt:      0,
			lockingAmt:     100,
			epochStartTime: baseTime.Add(-5 * time.Minute),
			epochDuration:  time.Minute * 5,
			expectedCredit: 100,
		},
		{
			name:           "In the middle of an epoch",
			lockedAmt:      0,
			lockingAmt:     100,
			epochStartTime: baseTime.Add(-2 * time.Minute),
			epochDuration:  time.Minute * 5,
			expectedCredit: 40,
		},
		{
			name:           "At the end of an epoch",
			lockedAmt:      0,
			lockingAmt:     100,
			epochStartTime: baseTime,
			epochDuration:  time.Minute * 5,
			expectedCredit: 0,
		},
		{
			name:           "Negative locking amount",
			lockedAmt:      0,
			lockingAmt:     -100,
			epochStartTime: baseTime.Add(-2 * time.Minute),
			epochDuration:  time.Minute * 5,
			expectedCredit: 0,
		},
		{
			name:           "Zero locking amount",
			lockedAmt:      0,
			lockingAmt:     0,
			epochStartTime: baseTime.Add(-2 * time.Minute),
			epochDuration:  time.Minute * 5,
			expectedCredit: 0,
		},
		{
			name:           "Locking amount with 1.1 rate",
			lockedAmt:      0,
			lockingAmt:     100,
			epochStartTime: baseTime.Add(-4 * time.Minute),
			epochDuration:  time.Minute * 5,
			expectedCredit: 80,
		},
		{
			name:           "Locking amount with 1.2 rate",
			lockedAmt:      0,
			lockingAmt:     200,
			epochStartTime: baseTime.Add(-3 * time.Minute),
			epochDuration:  time.Minute * 6,
			expectedCredit: 105,
		},
		{
			name:           "Locking amount with 1.5 rate",
			lockedAmt:      0,
			lockingAmt:     300,
			epochStartTime: baseTime.Add(-2 * time.Minute),
			epochDuration:  time.Minute * 5,
			expectedCredit: 132,
		},
		{
			name:           "Zero epoch duration",
			lockedAmt:      0,
			lockingAmt:     100,
			epochStartTime: baseTime,
			epochDuration:  0,
			expectedCredit: 0,
		},
		{
			name:           "Short epoch duration",
			lockedAmt:      0,
			lockingAmt:     100,
			epochStartTime: baseTime.Add(-30 * time.Second),
			epochDuration:  time.Minute * 1,
			expectedCredit: 50,
		},
		{
			name:           "Long epoch duration",
			lockedAmt:      0,
			lockingAmt:     100,
			epochStartTime: baseTime.Add(-15 * 24 * time.Hour),
			epochDuration:  time.Hour * 24 * 30,
			expectedCredit: 50,
		},
		{
			name:           "Epoch duration less than sinceCurrentEpoch",
			lockedAmt:      0,
			lockingAmt:     100,
			epochStartTime: baseTime.Add(-10 * time.Minute),
			epochDuration:  time.Minute * 5,
			expectedCredit: 100,
		},
		{
			name:           "Small locking amount and short epoch",
			lockedAmt:      0,
			lockingAmt:     9,
			epochStartTime: baseTime.Add(-3 * time.Second),
			epochDuration:  time.Second * 10,
			expectedCredit: 2,
		},
		{
			name:           "Small locking amount and long epoch",
			lockedAmt:      0,
			lockingAmt:     11,
			epochStartTime: baseTime.Add(-10 * time.Minute),
			epochDuration:  time.Hour * 2,
			expectedCredit: 0,
		},
		{
			name:           "Large locking amount and short epoch",
			lockedAmt:      0,
			lockingAmt:     1_000_003,
			epochStartTime: baseTime.Add(-1 * time.Second),
			epochDuration:  time.Second * 5,
			expectedCredit: 299_976,
		},
		{
			name:           "Large locking amount and long epoch",
			lockedAmt:      0,
			lockingAmt:     1_000_003,
			epochStartTime: baseTime.Add(-1 * time.Hour),
			epochDuration:  time.Hour * 5,
			expectedCredit: 299_976,
		},
		{
			name:           "Locking amount causing uneven division",
			lockedAmt:      0,
			lockingAmt:     7,
			epochStartTime: baseTime.Add(-10 * time.Second),
			epochDuration:  time.Second * 33,
			expectedCredit: 2,
		},
		{
			name:           "Epoch duration causing uneven division",
			lockedAmt:      0,
			lockingAmt:     1234,
			epochStartTime: baseTime.Add(-4 * time.Second),
			epochDuration:  time.Second * 9,
			expectedCredit: 769,
		},
		{
			name:           "Locking amount with 1.0 rate and previously locked amount",
			lockedAmt:      40,
			lockingAmt:     40,
			epochStartTime: baseTime.Add(-2 * time.Minute),
			epochDuration:  time.Minute * 5,
			expectedCredit: 16, // 40 * 1.0 * 2/5
		},
		{
			name:           "Locking amount with 1.1 rate and previously locked amount",
			lockedAmt:      100,
			lockingAmt:     100,
			epochStartTime: baseTime.Add(-2 * time.Minute),
			epochDuration:  time.Minute * 5,
			expectedCredit: 44, // 100 * 1.1 * 2/5
		},
		{
			name:           "Locking amount with 1.5 rate and previously locked amount",
			lockedAmt:      300,
			lockingAmt:     300,
			epochStartTime: baseTime.Add(-2 * time.Minute),
			epochDuration:  time.Minute * 5,
			expectedCredit: 180, // 300 * 1.5 * 2/5
		},
		{
			name:           "Locking amount with 1.2 + 1.1 rate and previously locked amount",
			lockedAmt:      150,
			lockingAmt:     150,
			epochStartTime: baseTime.Add(-2 * time.Minute),
			epochDuration:  time.Minute * 5,
			expectedCredit: 70, // ((100 * 1.2) + (50 * 1.1)) * 2/5
		},
		{
			name:           "Locking amount with 1.5 + 1.2 rate and previously locked amount",
			lockedAmt:      200,
			lockingAmt:     200,
			epochStartTime: baseTime.Add(-2 * time.Minute),
			epochDuration:  time.Minute * 5,
			expectedCredit: 108, // ((100 * 1.5) + (100 * 1.2)) * 2/5
		},
		{
			name:           "Locking amount with 1.5 + 1.2 + 1.1 + 1.0 rate and previously locked amount",
			lockedAmt:      50,
			lockingAmt:     350,
			epochStartTime: baseTime.Add(-2 * time.Minute),
			epochDuration:  time.Minute * 5,
			expectedCredit: 172, // ((100 * 1.5) + (100 * 1.2) + (100 * 1.1) + (50 * 1.0)) * 2/5
		},
		{
			name:           "Locking small amount with large previously locked amount",
			lockedAmt:      1_000_000,
			lockingAmt:     100,
			epochStartTime: baseTime.Add(-2 * time.Minute),
			epochDuration:  time.Minute * 5,
			expectedCredit: 60, // 100 * 1.5 * 2/5
		},
		{
			name:           "Locking large amount with large previously locked amount",
			lockedAmt:      1_000_000,
			lockingAmt:     10_000_000,
			epochStartTime: baseTime.Add(-2 * time.Minute),
			epochDuration:  time.Minute * 5,
			expectedCredit: 6_000_000, // 10,000,000 * 1.5 * 2/5
		},
	}

	rates := []types.Rate{
		{Amount: math.NewInt(300), Rate: 150},
		{Amount: math.NewInt(200), Rate: 120},
		{Amount: math.NewInt(100), Rate: 110},
		{Amount: math.NewInt(0), Rate: 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lockedAmt := math.NewInt(tt.lockedAmt)
			lockingAmt := math.NewInt(tt.lockingAmt)

			credit := calculateProratedCredit(
				rates,
				lockedAmt,
				lockingAmt,
				tt.epochStartTime,
				baseTime,
				tt.epochDuration,
			)

			if !credit.Equal(math.NewInt(tt.expectedCredit)) {
				t.Errorf("calculateProratedCredit() = %v, expected %v", credit, tt.expectedCredit)
			}
		})
	}
}
