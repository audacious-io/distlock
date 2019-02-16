package httpserver

import (
	"encoding/json"
	"net/http"
	"strconv"
)

const (
	jsonContentType = "application/json; charset=utf-8"
)

// Respond with JSON data.
//
// Only returns an error if there was an error encoding the JSON data.
func respondJson(resp http.ResponseWriter, data interface{}, statusCode int) error {
	// Encode the response as JSON data.
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	// Set the response headers and write the response.
	resp.Header().Set("Content-Type", jsonContentType)
	resp.Header().Set("Content-Length", strconv.Itoa(len(jsonData)))

	resp.WriteHeader(statusCode)
	resp.Write(jsonData)
	return nil
}

// Respond with an error.
func respondError(resp http.ResponseWriter, code string, message string, statusCode int) error {
	return respondJson(resp, map[string]interface{}{
		"code":    code,
		"message": message,
	}, statusCode)
}

// Respond with a not found error.
func respondNotFound(resp http.ResponseWriter) error {
	return respondError(resp, "not_found", "Not found", 404)
}
