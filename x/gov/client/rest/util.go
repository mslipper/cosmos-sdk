package rest

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/cosmos/cosmos-sdk/client/context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/wire"
	authctx "github.com/cosmos/cosmos-sdk/x/auth/client/context"

	"github.com/pkg/errors"
)

type baseReq struct {
	Name          string `json:"name"`
	Password      string `json:"password"`
	ChainID       string `json:"chain_id"`
	AccountNumber int64  `json:"account_number"`
	Sequence      int64  `json:"sequence"`
	Gas           int64  `json:"gas"`
}

func buildReq(w http.ResponseWriter, r *http.Request, cdc *wire.Codec, req interface{}) error {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		writeErr(&w, http.StatusBadRequest, err.Error())
		return err
	}
	err = cdc.UnmarshalJSON(body, req)
	if err != nil {
		writeErr(&w, http.StatusBadRequest, err.Error())
		return err
	}
	return nil
}

func (req baseReq) baseReqValidate(w http.ResponseWriter) bool {
	if len(req.Name) == 0 {
		writeErr(&w, http.StatusUnauthorized, "Name required but not specified")
		return false
	}

	if len(req.Password) == 0 {
		writeErr(&w, http.StatusUnauthorized, "Password required but not specified")
		return false
	}

	if len(req.ChainID) == 0 {
		writeErr(&w, http.StatusUnauthorized, "ChainID required but not specified")
		return false
	}

	if req.AccountNumber < 0 {
		writeErr(&w, http.StatusUnauthorized, "Account Number required but not specified")
		return false
	}

	if req.Sequence < 0 {
		writeErr(&w, http.StatusUnauthorized, "Sequence required but not specified")
		return false
	}
	return true
}

func writeErr(w *http.ResponseWriter, status int, msg string) {
	(*w).WriteHeader(status)
	err := errors.New(msg)
	(*w).Write([]byte(err.Error()))
}

// TODO: Build this function out into a more generic base-request
// (probably should live in client/lcd).
func signAndBuild(w http.ResponseWriter, cliCtx context.CLIContext, baseReq baseReq, msg sdk.Msg, cdc *wire.Codec) {
	txCtx := authctx.TxContext{
		Codec:         cdc,
		AccountNumber: baseReq.AccountNumber,
		Sequence:      baseReq.Sequence,
		ChainID:       baseReq.ChainID,
		Gas:           baseReq.Gas,
	}

	txBytes, err := txCtx.BuildAndSign(baseReq.Name, baseReq.Password, []sdk.Msg{msg})
	if err != nil {
		writeErr(&w, http.StatusUnauthorized, err.Error())
		return
	}

	res, err := cliCtx.BroadcastTx(txBytes)
	if err != nil {
		writeErr(&w, http.StatusInternalServerError, err.Error())
		return
	}

	output, err := wire.MarshalJSONIndent(cdc, res)
	if err != nil {
		writeErr(&w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Write(output)
}

func parseInt64OrReturnBadRequest(s string, w http.ResponseWriter) (n int64, ok bool) {
	var err error
	n, err = strconv.ParseInt(s, 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		err := fmt.Errorf("'%s' is not a valid int64", s)
		w.Write([]byte(err.Error()))
		return 0, false
	}
	return n, true
}
