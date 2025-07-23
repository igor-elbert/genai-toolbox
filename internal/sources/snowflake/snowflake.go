// Copyright 2025 Google LLC
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

package snowflake

import (
	"context"
	"fmt"
	"time"

	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/jmoiron/sqlx"
	"github.com/snowflakedb/gosnowflake"
	"go.opentelemetry.io/otel/trace"
)

const SourceKind string = "snowflake"

// validate interface
var _ sources.SourceConfig = &Config{}

func init() {
	if !sources.Register(SourceKind, newConfig) {
		panic(fmt.Sprintf("source kind %q already registered", SourceKind))
	}
}

func newConfig(ctx context.Context, name string, decoder *yaml.Decoder) (sources.SourceConfig, error) {
	actual := &Config{Name: name}
	if err := decoder.DecodeContext(ctx, actual); err != nil {
		return nil, err
	}
	return actual, nil
}

type Config struct {
	Name      string `yaml:"name" validate:"required"`
	Kind      string `yaml:"kind" validate:"required"`
	Account   string `yaml:"account" validate:"required"`
	User      string `yaml:"user" validate:"required"`
	Password  string `yaml:"password" validate:"required"`
	Database  string `yaml:"database" validate:"required"`
	Schema    string `yaml:"schema" validate:"required"`
	Warehouse string `yaml:"warehouse"`
	Role      string `yaml:"role"`
}

func (r *Config) SourceConfigKind() string {
	return SourceKind
}

func (r *Config) Initialize(ctx context.Context, tracer trace.Tracer) (sources.Source, error) {
	db, err := initSnowflakeConnection(ctx, tracer, r)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection: %w", err)
	}

	err = db.PingContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to connect successfully: %w", err)
	}

	s := &Source{
		Kind: SourceKind,
		DB:   db,
	}
	return s, nil
}

var _ sources.Source = &Source{}

type Source struct {
	Name string `yaml:"name"`
	Kind string `yaml:"kind"`
	DB   *sqlx.DB
}

func (s *Source) SourceKind() string {
	return SourceKind
}

func (s *Source) SnowflakeDB() *sqlx.DB {
	return s.DB
}

func initSnowflakeConnection(ctx context.Context, tracer trace.Tracer, cfg *Config) (*sqlx.DB, error) {
	_, span := sources.InitConnectionSpan(ctx, tracer, SourceKind, cfg.Name)
	defer span.End()

	// Set defaults for optional parameters
	if cfg.Warehouse == "" {
		cfg.Warehouse = "COMPUTE_WH"
	}
	if cfg.Role == "" {
		cfg.Role = "ACCOUNTADMIN"
	}

	// Use gosnowflake.Config for a more robust DSN construction
	dsnCfg := &gosnowflake.Config{
		Account:   cfg.Account,
		User:      cfg.User,
		Password:  cfg.Password,
		Database:  cfg.Database,
		Schema:    cfg.Schema,
		Warehouse: cfg.Warehouse,
		Role:      cfg.Role,
		Params: map[string]*string{
			"protocol": stringPtr("https"),
			"timeout":  stringPtr(fmt.Sprintf("%d", 60*time.Second)),
		},
	}

	dsn, err := gosnowflake.DSN(dsnCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to format Snowflake DSN: %w", err)
	}
	db, err := sqlx.Connect("snowflake", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to create Snowflake connection: %w", err)
	}

	return db, nil
}

func stringPtr(s string) *string {
	return &s
}
