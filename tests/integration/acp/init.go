package test

import (
	"github.com/sourcenetwork/vera/app"
)

var initialized bool = false

func initTest() {
	if !initialized {
		app.SetConfig(false)
		initialized = true
	}
}
