package tags

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Governance tags
var (
	ActionProposalDropped  = "proposal-dropped"
	ActionProposalPassed   = "proposal-passed"
	ActionProposalRejected = "proposal-rejected"

	Action            = sdk.TagAction
	Sender            = sdk.TagSender
	ProposalID        = "proposal-id"
	VotingPeriodStart = "voting-period-start"
	ProposalResult    = "proposal-result"
)
