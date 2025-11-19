package metrics

import (
	"os"

	"github.com/hashicorp/go-metrics"
)

// commonLabels models a set of labels which should be added
// to every Sourcehub meter
var commonLabels []metrics.Label

// AddCommonLabel appends the given name, value pair
// to the list of labels which are added to all Sourcehub meters
func AddCommonLabel(name, value string) {
	label := metrics.Label{
		Name:  name,
		Value: value,
	}
	commonLabels = append(commonLabels, label)
}

func init() {
	name, err := os.Hostname()
	if err != nil {
		panic("could not recover hostname")
	}
	AddCommonLabel(HostnameLabel, name)

	chainIDVal := os.Getenv(ChainIDEnvVar)
	if chainIDVal != "" {
		AddCommonLabel(ChainIDLabel, chainIDVal)
	}
}
