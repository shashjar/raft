package main

import (
	"encoding/json"
	"io"
	"net/http"
)

// Helper method to read HTTP request body and unmarshal it into a struct
func parseRequestBody(r *http.Request, v interface{}) error {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	defer r.Body.Close()

	return json.Unmarshal(body, v)
}

// Helper method to marshal data into JSON and send it as a response
func sendJSONResponse(w http.ResponseWriter, data interface{}) {
	responseBytes, err := json.Marshal(data)
	if err != nil {
		http.Error(w, "Failed to marshal response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(responseBytes)
}
