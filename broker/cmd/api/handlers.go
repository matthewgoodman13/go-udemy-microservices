package main

import (
	"broker/event"
	"broker/logs"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/rpc"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type RequestPayload struct {
	Action string      `json:"action"`
	Auth   AuthPayload `json:"auth,omitempty"`
	Log    LogPayload  `json:"log,omitempty"`
	Mail   MailPayload `json:"mail,omitempty"`
}

type AuthPayload struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LogPayload struct {
	Name string `json:"name"`
	Data string `json:"data"`
}

type MailPayload struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	Message string   `json:"message"`
}

func (app *Config) Broker(w http.ResponseWriter, r *http.Request) {
	payload := jsonResponse{
		Error:   false,
		Message: "Reached Broker Service",
	}

	_ = app.writeJSON(w, http.StatusOK, payload)
}

func (app *Config) HandleSubmission(w http.ResponseWriter, r *http.Request) {
	var requestPayload RequestPayload

	err := app.readJSON(w, r, &requestPayload)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	switch requestPayload.Action {
	case "auth":
		app.authenticate(w, requestPayload.Auth)
	case "log":
		app.logItem(w, requestPayload.Log)
	case "log_rabbit":
		app.logEventViaRabbit(w, requestPayload.Log)
	case "log_rpc":
		app.logEventViaRPC(w, requestPayload.Log)
	case "log_grpc":
		app.logEventViaGRPC(w, requestPayload.Log)
	case "mail":
		app.sendMail(w, requestPayload.Mail)
	default:
		app.errorJSON(w, errors.New("unknown action"), http.StatusBadRequest)
	}
}

func (app *Config) authenticate(w http.ResponseWriter, a AuthPayload) {
	// Create auth data to send to auth microservice
	jsonData, err := json.MarshalIndent(a, "", "\t")
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	// Call auth microservice
	request, err := http.NewRequest("POST", "http://auth-service/authenticate", bytes.NewBuffer(jsonData))
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	request.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		app.errorJSON(w, err)
		return
	}
	defer response.Body.Close()

	// Verify response from auth microservice
	if response.StatusCode == http.StatusUnauthorized {
		app.errorJSON(w, errors.New("invalid credentials"), http.StatusUnauthorized)
		return
	} else if response.StatusCode != http.StatusAccepted {
		app.errorJSON(w, errors.New("error calling authentication service"), http.StatusInternalServerError)
		return
	}

	// Read response from auth microservice
	var jsonFromAuthService jsonResponse
	err = json.NewDecoder(response.Body).Decode(&jsonFromAuthService)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	if jsonFromAuthService.Error {
		app.errorJSON(w, errors.New(jsonFromAuthService.Message), http.StatusUnauthorized)
		return
	}

	// Create response to send back to client
	var payload jsonResponse
	payload.Error = false
	payload.Message = "Successfully authenticated"
	payload.Data = jsonFromAuthService.Data

	_ = app.writeJSON(w, http.StatusAccepted, payload)
}

func (app *Config) logItem(w http.ResponseWriter, l LogPayload) {
	// Create log data to send to log microservice
	jsonData, err := json.MarshalIndent(l, "", "\t")
	if err != nil {
		app.errorJSON(w, err)
		return
	}
	// Call log microservice
	request, err := http.NewRequest("POST", "http://logger-service/log", bytes.NewBuffer(jsonData))
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	request.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		app.errorJSON(w, err)
		return
	}
	defer response.Body.Close()

	// Verify response from log microservice
	if response.StatusCode != http.StatusAccepted {
		app.errorJSON(w, errors.New("error calling log service"), http.StatusInternalServerError)
		return
	}

	// Create response to send back to client
	var payload jsonResponse
	payload.Error = false
	payload.Message = "Successfully logged"

	_ = app.writeJSON(w, http.StatusAccepted, payload)
}

func (app *Config) sendMail(w http.ResponseWriter, m MailPayload) {
	// Create mail data to send to mail microservice
	jsonData, err := json.MarshalIndent(m, "", "\t")
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	// Call mail microservice
	request, err := http.NewRequest("POST", "http://mailer-service/send", bytes.NewBuffer(jsonData))
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	request.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		app.errorJSON(w, err)
		return
	}
	defer response.Body.Close()

	// Verify response from mail microservice
	if response.StatusCode != http.StatusAccepted {
		app.errorJSON(w, errors.New("error calling mail service"), http.StatusInternalServerError)
		return
	}

	// Create response to send back to client
	var payload jsonResponse
	payload.Error = false
	payload.Message = "Successfully sent mail"

	_ = app.writeJSON(w, http.StatusAccepted, payload)
}

func (app *Config) logEventViaRabbit(w http.ResponseWriter, l LogPayload) {
	err := app.pushToQueue(l.Name, l.Data)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	var payload jsonResponse
	payload.Error = false
	payload.Message = "Successfully logged via RabbitMQ"

	_ = app.writeJSON(w, http.StatusAccepted, payload)
}

func (app *Config) pushToQueue(name, msg string) error {
	producer, err := event.NewEventProducer(app.Rabbit)
	if err != nil {
		return err
	}

	payload := LogPayload{
		Name: name,
		Data: msg,
	}

	jsonData, err := json.MarshalIndent(payload, "", "\t")
	if err != nil {
		return err
	}

	err = producer.Publish(string(jsonData), "log.INFO")
	if err != nil {
		return err
	}

	return nil
}

type RPCPayload struct {
	Name string `json:"name"`
	Data string `json:"data"`
}

func (app *Config) logEventViaRPC(w http.ResponseWriter, l LogPayload) {
	client, err := rpc.Dial("tcp", "logger-service:5001")
	if err != nil {
		app.errorJSON(w, err)
		return
	}
	defer client.Close()

	rpcPayload := RPCPayload{
		Name: l.Name,
		Data: l.Data,
	}

	var result string
	err = client.Call("RPCServer.LogInfo", rpcPayload, &result)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	payload := jsonResponse{
		Error:   false,
		Message: result,
	}

	_ = app.writeJSON(w, http.StatusAccepted, payload)
}

func (app *Config) logEventViaGRPC(w http.ResponseWriter, l LogPayload) {
	conn, err := grpc.Dial("logger-service:50001", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		app.errorJSON(w, err)
		return
	}
	defer conn.Close()

	client := logs.NewLogServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, err = client.WriteLog(ctx, &logs.LogRequest{
		LogEntry: &logs.Log{
			Name: l.Name,
			Data: l.Data,
		},
	})
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	payload := jsonResponse{
		Error:   false,
		Message: "Successfully logged via gRPC",
	}

	_ = app.writeJSON(w, http.StatusAccepted, payload)
}
