package cli

import (
	"strings"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"

	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

func CmdQueryFilterRelationships() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "filter-relationships [policy-id] [object] [relation] [subject]",
		Short: "Filters through relationships in a policy",
		Long: `Filters thourgh all relationships in a Policy. 
                Performs a lookup using the object, relation and subject filters.
                Uses a mini grammar as describe:
                object := resource:id | *
                relation := name | *
                subject := id | *
                Returns`,
		Args:    cobra.ExactArgs(4),
		Aliases: []string{"relationships"},
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			polId := args[0]
			object := args[1]
			relation := args[2]
			subject := args[3]

			queryClient := types.NewQueryClient(clientCtx)
			req := types.QueryFilterRelationshipsRequest{
				PolicyId: polId,
				Selector: buildSelector(object, relation, subject),
			}

			res, err := queryClient.FilterRelationships(cmd.Context(), &req)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

func buildSelector(object, relation, subject string) *coretypes.RelationshipSelector {
	objSelector := &coretypes.ObjectSelector{}
	relSelector := &coretypes.RelationSelector{}
	subjSelector := &coretypes.SubjectSelector{}

	if object == "*" {
		objSelector.Selector = &coretypes.ObjectSelector_Wildcard{
			Wildcard: &coretypes.WildcardSelector{},
		}
	} else {
		res, id, _ := strings.Cut(object, ":")
		objSelector.Selector = &coretypes.ObjectSelector_Object{
			Object: &coretypes.Object{
				Resource: res,
				Id:       id,
			},
		}
	}

	if relation == "*" {
		relSelector.Selector = &coretypes.RelationSelector_Wildcard{
			Wildcard: &coretypes.WildcardSelector{},
		}
	} else {
		relSelector.Selector = &coretypes.RelationSelector_Relation{
			Relation: relation,
		}
	}

	if subject == "*" {
		subjSelector.Selector = &coretypes.SubjectSelector_Wildcard{
			Wildcard: &coretypes.WildcardSelector{},
		}
	} else {
		subjSelector.Selector = &coretypes.SubjectSelector_Subject{
			Subject: &coretypes.Subject{
				Subject: &coretypes.Subject_Actor{
					Actor: &coretypes.Actor{
						Id: subject,
					},
				},
			},
		}
	}

	return &coretypes.RelationshipSelector{
		ObjectSelector:   objSelector,
		RelationSelector: relSelector,
		SubjectSelector:  subjSelector,
	}
}
