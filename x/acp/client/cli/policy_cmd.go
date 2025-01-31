package cli

import (
	"encoding/hex"
	"fmt"
	"strconv"

	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/gogoproto/jsonpb"
	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
	"github.com/spf13/cobra"
)

type dispatcher = func(*cobra.Command, string, *types.PolicyCmd) error

func CmdSetRelationship(dispatcher dispatcher) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-relationship policy-id resource objectId relation [subject resource] subjectId [subjRel]",
		Short: "Issue a SetRelationship PolicyCmd",
		Long:  ``,
		Args:  cobra.MinimumNArgs(5),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			polId, polCmd, err := parseSetRelationshipArgs(args)
			if err != nil {
				return err
			}

			err = dispatcher(cmd, polId, polCmd)
			if err != nil {
				return err
			}
			return nil
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func CmdDeleteRelationship(dispatcher dispatcher) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete-relationship policy-id resource objectId relation [subject resource] subjectId [subjRel]",
		Short: "Issues a DeleteRelationship PolicyCmd",
		Long:  ``,
		Args:  cobra.MinimumNArgs(5),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			polId, polCmd, err := parseDeleteRelationshipArgs(args)
			if err != nil {
				return err
			}

			err = dispatcher(cmd, polId, polCmd)
			if err != nil {
				return err
			}
			return nil
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func CmdRegisterObject(dispatcher dispatcher) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "register-object policy-id resource objectId",
		Short: "Issue RegisterObject PolicyCmd",
		Long:  ``,
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			polId, polCmd, err := parseRegisterObjectArgs(args)
			if err != nil {
				return err
			}

			err = dispatcher(cmd, polId, polCmd)
			if err != nil {
				return err
			}
			return nil
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func CmdArchiveObject(dispatcher dispatcher) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "archive-object policy-id resource objectId",
		Short: "Issue ArchiveObject PolicyCmd",
		Long:  ``,
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			polId, polCmd, err := parseArchiveObjectArgs(args)
			if err != nil {
				return err
			}
			err = dispatcher(cmd, polId, polCmd)
			if err != nil {
				return err
			}
			return nil
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func CmdRevealRegistration(dispatcher dispatcher) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reveal-registration commitment-id json-proof",
		Short: "Reveal an Object Registration for a Commitment",
		Long:  ``,
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			commitId, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid commitId: %w", err)
			}
			proofJson := args[1]

			proof := &types.RegistrationProof{}
			err = jsonpb.UnmarshalString(proofJson, proof)
			if err != nil {
				return fmt.Errorf("unmarshaling proof: %v", err)
			}

			polCmd := types.NewRevealRegistrationCmd(commitId, proof)

			err = dispatcher(cmd, "", polCmd)
			if err != nil {
				return err
			}
			return nil
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func CmdCreateCommitment(dispatcher dispatcher) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create-commitment policy-id hex-commitment",
		Short: "Create a new Registration Commitment on SourceHub",
		Long:  ``,
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			polId := args[0]
			commitment, err := hex.DecodeString(args[1])
			if err != nil {
				return fmt.Errorf("decoding commitment: %v", err)
			}
			polCmd := types.NewCommitRegistrationCmd(commitment)

			err = dispatcher(cmd, polId, polCmd)
			if err != nil {
				return err
			}
			return nil
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func CmdFlagHijack(dispatcher dispatcher) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "flag-hijack policy-id event-id",
		Short: "Issue UnregisterObject PolicyCmd",
		Long:  ``,
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			polId := args[0]
			eventId, err := strconv.ParseUint(args[1], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid event id: %w", err)
			}

			polCmd := types.NewFlagHijackAttemptCmd(eventId)

			err = dispatcher(cmd, polId, polCmd)
			if err != nil {
				return err
			}
			return nil
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func CmdUnarchiveObject(dispatcher dispatcher) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unarchive-object policy-id resource objectId",
		Short: "Issue UnarchiveObject PolicyCmd",
		Long:  ``,
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			polId := args[0]
			obj := coretypes.NewObject(args[1], args[2])
			polCmd := types.NewUnarchiveObjectCmd(obj)
			err = dispatcher(cmd, polId, polCmd)
			if err != nil {
				return err
			}
			return nil
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// parseSetRelationshipArgs parses a SetRelationship from cli args
// format: policy-id resource objectId relation [subject resource] subjectId [subjRel]
func parseSetRelationshipArgs(args []string) (string, *types.PolicyCmd, error) {
	polId := args[0]
	resource := args[1]
	objId := args[2]
	relation := args[3]
	var relationship *coretypes.Relationship
	if len(args) == 5 {
		subjectId := args[4]
		relationship = coretypes.NewActorRelationship(resource, objId, relation, subjectId)
	} else if len(args) == 6 {
		subjResource := args[4]
		subjectId := args[5]
		relationship = coretypes.NewRelationship(resource, objId, relation, subjResource, subjectId)
	} else if len(args) == 7 {
		subjResource := args[4]
		subjectId := args[5]
		subjRel := args[6]
		relationship = coretypes.NewActorSetRelationship(resource, objId, relation, subjResource, subjectId, subjRel)
	} else {
		return "", nil, fmt.Errorf("invalid number of arguments: Set Relationship expects [5,7] cli args")
	}
	cmd := types.NewSetRelationshipCmd(relationship)
	return polId, cmd, nil
}

// parseDeleteRelationshipArgs parses a DeleteRelationship from cli args
// format: policy-id resource objectId relation [subject resource] subjectId [subjRel]
func parseDeleteRelationshipArgs(args []string) (string, *types.PolicyCmd, error) {
	polId := args[0]
	resource := args[1]
	objId := args[2]
	relation := args[3]
	var relationship *coretypes.Relationship
	if len(args) == 5 {
		subjectId := args[4]
		relationship = coretypes.NewActorRelationship(resource, objId, relation, subjectId)
	} else if len(args) == 6 {
		subjResource := args[4]
		subjectId := args[5]
		relationship = coretypes.NewRelationship(resource, objId, relation, subjResource, subjectId)
	} else if len(args) == 7 {
		subjResource := args[4]
		subjectId := args[5]
		subjRel := args[6]
		relationship = coretypes.NewActorSetRelationship(resource, objId, relation, subjResource, subjectId, subjRel)
	} else {
		return "", nil, fmt.Errorf("invalid number of arguments: Delete Relationship expects [5,7] cli args")
	}
	cmd := types.NewDeleteRelationshipCmd(relationship)
	return polId, cmd, nil
}

// parseRegisterObjectArgs parses a RegisterObject from cli args
// format: policy-id resource objectId
func parseRegisterObjectArgs(args []string) (string, *types.PolicyCmd, error) {
	if len(args) != 3 {
		return "", nil, fmt.Errorf("RegisterObject: invalid number of arguments: policy-id resource objectId")
	}
	polId := args[0]
	resource := args[1]
	objId := args[2]
	return polId, types.NewRegisterObjectCmd(coretypes.NewObject(resource, objId)), nil
}

func parseArchiveObjectArgs(args []string) (string, *types.PolicyCmd, error) {
	if len(args) != 3 {
		return "", nil, fmt.Errorf("ArchiveObject: invalid number of arguments: policy-id resource objectId")
	}
	polId := args[0]
	resource := args[1]
	objId := args[2]
	return polId, types.NewArchiveObjectCmd(coretypes.NewObject(resource, objId)), nil
}
