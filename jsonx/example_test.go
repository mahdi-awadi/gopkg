package jsonx_test

import (
	"net/http"

	"github.com/mahdi-awadi/gopkg/jsonx"
)

type CreateUserReq struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type CreateUserResp struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

func Example() {
	http.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
		var req CreateUserReq
		if err := jsonx.Decode(r, &req, jsonx.DecodeOptions{DisallowUnknownFields: true}); err != nil {
			jsonx.Error(w, http.StatusBadRequest, err.Error())
			return
		}

		// ... create user, produce resp ...
		resp := CreateUserResp{ID: "u_42", Name: req.Name, Email: req.Email}

		_ = jsonx.Write(w, http.StatusCreated, resp)
	})
}
