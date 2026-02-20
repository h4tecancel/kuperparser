package respond

import (
	"encoding/json"
	"net/http"
)

type ErrorBody struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func WriteInternalError(w http.ResponseWriter) {
	WriteError(w, http.StatusInternalServerError, "internal_error", "internal error")
}

func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func WriteError(w http.ResponseWriter, status int, code, msg string) {
	var b ErrorBody
	b.Error.Code = code
	b.Error.Message = msg
	WriteJSON(w, status, b)
}
