package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
)

// Helper function to read HTTP request body and unmarshal it into a struct
func parseIncomingRequest(r *http.Request, v interface{}) error {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	defer r.Body.Close()

	if err := json.Unmarshal(body, v); err != nil {
		return err
	}

	return nil
}

// Helper function to marshal data into JSON and send it as a response
func sendOutgoingResponse(w http.ResponseWriter, data interface{}) error {
	responseBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(responseBytes)

	return nil
}

// Helper function to make an HTTP request to a server and unmarshal the response into a struct
func makeFullRequest(serverAddr string, path string, args interface{}, results interface{}) error {
	reqBody, err := json.Marshal(args)
	if err != nil {
		return err
	}

	resp, err := http.Post(serverAddr+path, "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(respBody, &results); err != nil {
		return err
	}

	return nil
}
