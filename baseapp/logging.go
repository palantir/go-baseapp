// Copyright 2019 Palantir Technologies, Inc.
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

package baseapp

import (
	"io"
	"os"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

// ConfigureDefaultLogger returns a zerolog logger based on the conventions in a LoggingConfig
func ConfigureDefaultLogger(c LoggingConfig) (zerolog.Logger, error) {
	out := io.Writer(os.Stdout)
	if c.Text {
		out = zerolog.ConsoleWriter{Out: out}
	}

	logger := zerolog.New(out).With().Timestamp().Logger()
	if c.Level == "" {
		return logger, nil
	}

	level, err := zerolog.ParseLevel(c.Level)
	if err != nil {
		return logger, errors.Wrap(err, "failed to parse log level")
	}

	return logger.Level(level), nil
}
