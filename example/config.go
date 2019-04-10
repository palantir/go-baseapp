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
	"io/ioutil"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"

	"github.com/palantir/go-baseapp/baseapp"
	"github.com/palantir/go-baseapp/baseapp/datadog"
)

type Config struct {
	Server  baseapp.HTTPConfig    `yaml:"server"`
	Datadog datadog.Config        `yaml:"datadog"`
	Logging baseapp.LoggingConfig `yaml:"logging"`

	App AppConfig `yaml:"app"`
}

type AppConfig struct {
	Message string `yaml:"message"`
}

func ReadConfig(path string) (Config, error) {
	var c Config

	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return c, errors.Wrapf(err, "failed reading server config file: %s", path)
	}

	if err := yaml.UnmarshalStrict(bytes, &c); err != nil {
		return c, errors.Wrap(err, "failed parsing configuration file")
	}

	return c, nil
}
