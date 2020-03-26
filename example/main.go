// Copyright 2018 Palantir Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"net/http"

	"github.com/palantir/go-baseapp/baseapp"
	"github.com/palantir/go-baseapp/baseapp/datadog"
	"github.com/rs/zerolog"
	"goji.io/pat"
)

type MessageHandler struct {
	Message string
}

func (h *MessageHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := zerolog.Ctx(ctx)

	// Logging example
	logger.Info().Str("user-agent", r.Header.Get("User-Agent")).Msg("Received request")

	baseapp.WriteJSON(w, http.StatusOK, map[string]string{
		"message": h.Message,
	})
}

// type assertion
var _ http.Handler = &MessageHandler{}

func main() {
	// Load your configuration from a file
	config, err := ReadConfig("example/config.yml")
	if err != nil {
		panic(err)
	}

	// Configure a root logger for everything to use
	logger := baseapp.NewLogger(config.Logging)

	// Create a server using the default options
	serverParams := baseapp.DefaultParams(logger, "example.")
	server, err := baseapp.NewServer(config.Server, serverParams...)
	if err != nil {
		panic(err)
	}

	// Register your routes with the server
	messageHandler := &MessageHandler{Message: config.App.Message}
	server.Mux().Handle(pat.Get("/api/message"), messageHandler)

	// Start a goroutine to emit server metrics to Datadog
	if err := datadog.StartEmitter(server, config.Datadog); err != nil {
		panic(err)
	}

	// Start the server (blocking)
	_ = server.Start()
}
