package main

import (
	"log"
	"logger-service/data"
	"net/http"
)

type JSONPayload struct {
	Name string `json:"name"`
	Data string `json:"data"`
}

func (app *Config) WriteLog(w http.ResponseWriter, r *http.Request) {

	log.Println("WriteLog() called")

	// Read json into var
	var requestPayload JSONPayload
	err := app.readJSON(w, r, &requestPayload)
	if err != nil {
		log.Println("Error reading JSON: ", err)
		app.errorJSON(w, err)
		return
	}

	// Insert into database
	event := data.LogEntry{
		Name: requestPayload.Name,
		Data: requestPayload.Data,
	}
	err = app.Models.LogEntry.Insert(event)
	if err != nil {
		log.Println("Error inserting into database: ", err)
		app.errorJSON(w, err)
		return
	}

	// Return success
	res := jsonResponse{
		Error:   false,
		Message: "Log entry created",
	}

	log.Println("WriteLog() returning")
	app.writeJSON(w, http.StatusAccepted, res, nil)
}
