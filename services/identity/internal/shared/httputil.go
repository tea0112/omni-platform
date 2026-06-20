package shared

import (
	"encoding/json"
	"net/http"
)

func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func WriteErr(w http.ResponseWriter, err error) {
	status, _, body := MapError(err)
	WriteJSON(w, status, body)
}
