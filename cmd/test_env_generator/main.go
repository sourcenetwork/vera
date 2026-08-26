package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	test "github.com/sourcenetwork/vera/tests/integration/acp"
	"github.com/sourcenetwork/vera/utils"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "test_env_generator {permutation}",
	Short: "test_env_generator permutates through Vera's test suite environment variables",
	Long: `
	test_env_generator outputs the set of environment variables which should be set for each test permutation.
		   
	With no input, prints the amount of permutations available.
	Permutation numbering is 0 based (eg if there are permutations the allowed options arguments are 0, 1, 2)
	`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		environs := genEnvirons()

		if len(args) == 0 {
			fmt.Printf("%v\n", len(environs))
			return
		}

		if args[0] == "all" {
			for _, env := range environs {
				fmt.Println(env)
			}
			return
		}

		index, err := strconv.Atoi(args[0])
		if err != nil {
			log.Fatalf("%v is an invalid index", args[0])
		}
		if index < 0 || index > len(environs) {
			log.Fatalf("index must be within [0, %v]", len(environs)-1)
		}

		println(environs[index])
	},
}

func main() {
	rootCmd.Execute()
}

func writeKV(builder *strings.Builder, key, value string) {
	builder.WriteString("export ")
	builder.WriteString(key)
	builder.WriteRune('=')
	builder.WriteRune('"')
	builder.WriteString(value)
	builder.WriteRune('"')
	builder.WriteRune(' ')
	builder.WriteRune(';')
}

func genEnvirons() []string {
	combinations := len(test.ActorKeyMap) * len(test.ExecutorStrategyMap) * len(test.AuthenticationStrategyMap)
	environs := make([]string, 0, combinations)

	for actorKeyVar := range test.ActorKeyMap {
		for executorVar := range test.ExecutorStrategyMap {
			for authStratVar := range test.AuthenticationStrategyMap {
				// ED25519 key type is not valid for direct authentication
				// since ed25519 accounts cannot sign txs
				if actorKeyVar == "ED25519" && authStratVar == "DIRECT" {
					continue
				}

				builder := strings.Builder{}
				writeKV(&builder, test.VeraActorEnvVar, actorKeyVar)
				writeKV(&builder, test.VeraExecutorEnvVar, executorVar)
				writeKV(&builder, test.VeraAuthStratEnvVar, authStratVar)
				environ := builder.String()
				environs = append(environs, environ)
			}
		}
	}

	utils.SortSlice(environs)

	return environs
}
