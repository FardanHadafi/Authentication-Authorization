package shared

import (
	"encoding/json/v2"
	"net/http"
)

type ErrorResponse struct {
	Error string `json:"error"`
}

func WriteJSON(w http.ResponseWriter, statusCode int, data any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	return json.MarshalWrite(w, data)
}

func WriteError(w http.ResponseWriter, statusCode int, message string) {
	_ = WriteJSON(w, statusCode, ErrorResponse{Error: message})
}
