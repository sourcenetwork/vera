package keeper_test

import (
	"os"
	"testing"

	"github.com/sourcenetwork/vera/app"
)

func TestMain(m *testing.M) {
	app.SetConfig(true)
	os.Exit(m.Run())
}
