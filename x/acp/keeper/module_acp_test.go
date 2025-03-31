package keeper

import "testing"

func Test_CreateModulePolicy_ModuleCanCreatePolicy(t *testing.T)    {}
func Test_CreateModulePolicy_ModuleCanEditTheirPolicy(t *testing.T) {}

func Test_CreateModulePolicy_ExternalModuleCannotEditOtherModulesPolicy(t *testing.T) {}
func Test_CreateModulePolicy_ModuleCanAddRelationshipsToTheirPolicy(t *testing.T)     {}

func Test_CreateModulePolicy_ModuleCannotUsePolicyWithoutClaimingCapability(t *testing.T)     {}
func Test_CreateModulePolicy_ExternalModuleCannotForgeCapabilityToModulePolicy(t *testing.T)  {}
func Test_CreateModulePolicy_ExternalModuleCannotForgeCapabilityToAnActorPolicy(t *testing.T) {}
