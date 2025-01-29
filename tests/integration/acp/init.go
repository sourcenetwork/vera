package test

import (
	"github.com/sourcenetwork/sourcehub/app"
)

var initialized bool = false

func initTest() {
	if !initialized {
		app.SetConfig(false)
		initialized = true
	}
}
