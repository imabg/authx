package httpx

import (
	"encoding/json"
	"net/http"
)

type ErrorBody struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

func WriteJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if body == nil {
		return
	}
	_ = json.NewEncoder(w).Encode(body)
}

func WriteError(w http.ResponseWriter, status int, code, message string) {
	WriteJSON(w, status, ErrorBody{Error: message, Code: code})
}

func DecodeJSON(r *http.Request, dest any) error {
	defer r.Body.Close()
	dec := json.NewDecoder(r.Body)
	return dec.Decode(dest)
}
