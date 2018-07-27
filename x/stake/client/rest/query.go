package rest

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/gorilla/mux"

	"github.com/cosmos/cosmos-sdk/client/context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/wire"

	"github.com/cosmos/cosmos-sdk/x/stake"
	"github.com/cosmos/cosmos-sdk/x/stake/types"
)

const storeName = "stake"

func contains(stringSlice []string, txType string) bool {
	for _, word := range stringSlice {
		if word == txType {
			return true
		}
	}
	return false
}

func registerQueryRoutes(ctx context.CoreContext, r *mux.Router, cdc *wire.Codec) {

	// GET /delegators/{addr}/txs // Get all staking txs from a delegator

	// GET /delegators/{addr}/txs/{id} // Get a specific staking tx from a delegator
	// GET /delegators/{addr}/validators // Query all validators that a delegator is bonded to
	// GET /delegators/{addr}/validators/{addr} // Query a validator info that a delegator is bonded to

	// GET /delegators/{addr}/validators/{addr}/txs // Get all txs to a validator performed by a delegator
	// GET /delegators/{addr}/validators/{addr}/txs?type=bond // Get all bonding txs to a validator performed by a delegator
	// GET /delegators/{addr}/validators/{addr}/txs?type=unbond // Get all unbonding txs to a validator performed by a delegator
	// GET /delegators/{addr}/validators/{addr}/txs?type=redelegate // Get all redelegation txs to a validator performed by a delegator
	r.HandleFunc(
		"/delegators/{delegatorAddr}/validators/{validatorAddr}/txs",
		stakingTxsHandlerFn(ctx, cdc),
	).Queries("type", "{type}").Methods("GET")

	// GET /validators/
	r.HandleFunc(
		"/validators",
		validatorsHandlerFn(ctx, cdc),
	).Methods("GET")

	// GET /validators/{addr}
	r.HandleFunc(
		"/validators/{addr}",
		validatorHandlerFn(ctx, cdc),
	).Methods("GET")

	// GET /validators/{addr}/delegators
	// r.HandleFunc(
	// 	"/validators/{addr}/delegators",
	// 	validatorDelegatorsHandlerFn(ctx, cdc),
	// ).Methods("GET")
}

func stakingTxsHandlerFn(ctx context.CoreContext, cdc *wire.Codec) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// read parameters
		vars := mux.Vars(r)
		bech32delegator := vars["delegatorAddr"]
		bech32validator := vars["validatorAddr"]

		delegatorAddr, err := sdk.AccAddressFromBech32(bech32delegator)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return
		}

		validatorAddr, err := sdk.AccAddressFromBech32(bech32validator)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return
		}

		var typesQuerySlice []string
		var output []byte
		typesQuery := r.URL.Query().Get("type")
		typesQuerySlice = strings.Split(strings.TrimSpace(typesQuery), " ")

		noQuery := len(typesQuerySlice) == 0

		// delegation
		if noQuery || contains(typesQuerySlice, "bond") {
			key := stake.GetDelegationKey(delegatorAddr, validatorAddr)

			res, err := ctx.QueryStore(key, storeName)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(fmt.Sprintf("couldn't query delegation. Error: %s", err.Error())))
				return
			}

			// the query will return empty if there is no data for this record
			if len(res) == 0 {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			delegation, err := types.UnmarshalDelegation(cdc, key, res)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(err.Error()))
				return
			}

			outputBond, err := cdc.MarshalJSON(delegation)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(err.Error()))
				return
			}
			output = append(output, outputBond...)
		}
		if noQuery || contains(typesQuerySlice, "unbond") {

			key := stake.GetUBDKey(delegatorAddr, validatorAddr)

			res, err := ctx.QueryStore(key, storeName)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(fmt.Sprintf("couldn't query unbonding-delegation. Error: %s", err.Error())))
				return
			}

			// the query will return empty if there is no data for this record
			if len(res) == 0 {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			ubd, err := types.UnmarshalUBD(cdc, key, res)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(fmt.Sprintf("couldn't query unbonding-delegation. Error: %s", err.Error())))
				return
			}

			outputUnbond, err := cdc.MarshalJSON(ubd)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(err.Error()))
				return
			}
			output = append(output, outputUnbond...)
		}
		if noQuery || contains(typesQuerySlice, "redelegate") {
			// keyValidatorFrom := stake.GetREDsByDelFromValSrcIndexKey(delegatorAddr, validatorAddr)
			keyValidatorTo := stake.GetREDsByDelToValDstIndexKey(delegatorAddr, validatorAddr)

			// All redelegations from source validator (i.e unbonding)
			res, err := ctx.QueryStore(keyValidatorFrom, storeName)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(fmt.Sprintf("couldn't query redelegation. Error: %s", err.Error())))
				return
			}

			// the query will return empty if there is no data for this record
			if len(res) == 0 {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			red, err := types.UnmarshalRED(cdc, keyValidatorFrom, res)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(fmt.Sprintf("couldn't query unbonding-delegation. Error: %s", err.Error())))
				return
			}

			outputFrom, err := cdc.MarshalJSON(red)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(err.Error()))
				return
			}

			// All redelegations to destination validator (i.e bonding)
			res, err = ctx.QueryStore(keyValidatorTo, storeName)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(fmt.Sprintf("couldn't query redelegation. Error: %s", err.Error())))
				return
			}

			// the query will return empty if there is no data for this record
			if len(res) == 0 {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			red, err = types.UnmarshalRED(cdc, keyValidatorTo, res)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(fmt.Sprintf("couldn't query unbonding-delegation. Error: %s", err.Error())))
				return
			}

			outputTo, err := cdc.MarshalJSON(red)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(err.Error()))
				return
			}

			// All redelegations From or To the given validator address
			output := append(outputFrom, outputTo...)
		}
		w.Write(output)
	}
}

// TODO bech32
// http request handler to query list of validators
func validatorsHandlerFn(ctx context.CoreContext, cdc *wire.Codec) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		kvs, err := ctx.QuerySubspace(cdc, stake.ValidatorsKey, storeName)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(fmt.Sprintf("couldn't query validators. Error: %s", err.Error())))
			return
		}

		// the query will return empty if there are no validators
		if len(kvs) == 0 {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		// parse out the validators
		validators := make([]types.BechValidator, len(kvs))
		for i, kv := range kvs {

			addr := kv.Key[1:]
			validator, err := types.UnmarshalValidator(cdc, addr, kv.Value)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(fmt.Sprintf("Error: %s", err.Error())))
				return
			}

			bech32Validator, err := validator.Bech32Validator()
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(err.Error()))
				return
			}
			validators[i] = bech32Validator
		}

		output, err := cdc.MarshalJSON(validators)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

		w.Write(output)
	}
}

// HTTP request handler to query the validator information from a given validator address
func validatorHandlerFn(ctx context.CoreContext, cdc *wire.Codec) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// read parameters
		vars := mux.Vars(r)
		bech32validatorAddr := vars["addr"]
		valAddress, err := sdk.AccAddressFromBech32(bech32validatorAddr)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(fmt.Sprintf("Error: %s", err.Error())))
			return
		}

		kvs, err := ctx.QuerySubspace(cdc, stake.ValidatorsKey, storeName)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(fmt.Sprintf("Error: %s", err.Error())))
			return
		}

		// the query will return empty if there are no validators
		if len(kvs) == 0 {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		var output []byte

		// parse out the validators
		for i, kv := range kvs {
			addr := kv.Key[1:]
			validator, err := types.UnmarshalValidator(cdc, addr, kv.Value)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(fmt.Sprintf("couldn't query unbonding-delegation. Error: %s", err.Error())))
				return
			}

			ownerAddress := validator.GetOwner()
			if reflect.DeepEqual(ownerAddress.Bytes(), valAddress.Bytes()) {
				output, err = cdc.MarshalJSON(validator)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte(err.Error()))
					return
				}
				break
			}
		}

		if output == nil {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.Write(output)
	}
}

func validatorDelegatorsHandlerFn(ctx context.CoreContext, cdc *wire.Codec) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// read parameters
		vars := mux.Vars(r)
		bech32validatorAddr := vars["addr"]
		valAddress, err := sdk.AccAddressFromBech32(bech32validatorAddr)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(fmt.Sprintf("Error: %s", err.Error())))
			return
		}

		kvs, err := ctx.QuerySubspace(cdc, stake.ValidatorsKey, storeName)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(fmt.Sprintf("Error: %s", err.Error())))
			return
		}

		// the query will return empty if there are no validators
		if len(kvs) == 0 {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		var output []byte

		// parse out the validators
		for i, kv := range kvs {
			addr := kv.Key[1:]
			validator, err := types.UnmarshalValidator(cdc, addr, kv.Value)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(fmt.Sprintf("couldn't query unbonding-delegation. Error: %s", err.Error())))
				return
			}

			ownerAddress := validator.GetOwner()
			if reflect.DeepEqual(ownerAddress.Bytes(), valAddress.Bytes()) {
				ctx.QuerySubspace(cdc, stake.DelegationKey, storeName)
			}
		}
	}
}
