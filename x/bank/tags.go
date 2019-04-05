package bank

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Tag keys and values
var (
	TagActionUndelegateCoins = []byte("undelegateCoins")
	TagActionDelegateCoins   = []byte("delegateCoins")

	TagKeyRecipient = "recipient"
	TagSender       = sdk.TagSender
)
