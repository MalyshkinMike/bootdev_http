package main

import (
	"encoding/json"
	"net/http"
	"log"
)

func respondWithError(w http.ResponseWriter, code int, msg string, err error) {
	type errorResponse struct {
		Message string `json: "error"`
	}
	if err != nil {
		log.Printf("Error occured, %s", err)
	}
	resp := errorResponse{ Message: msg }
	respondWithJson(w, code, resp)
}

func respondWithJson(w http.ResponseWriter, code int, payload interface{}) {

	body, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(body)
}
