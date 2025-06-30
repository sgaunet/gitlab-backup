// Package config provides configuration management for GitLab backup application.
package config

import (
	"fmt"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/sgaunet/gitlab-backup/pkg/hooks"
	"gopkg.in/yaml.v3"
)

// S3Config holds the configuration for S3 storage backend.
type S3Config struct {
	Endpoint   string `env:"S3ENDPOINT"            env-default:""   yaml:"endpoint"`
	BucketName string `env:"S3BUCKETNAME"          env-default:""   yaml:"bucketName"`
	BucketPath string `env:"S3BUCKETPATH"          env-default:""   yaml:"bucketPath"`
	Region     string `env:"S3REGION"              env-default:""   yaml:"region"`
	AccessKey  string `env:"AWS_ACCESS_KEY_ID"     yaml:"accessKey"`
	SecretKey  string `env:"AWS_SECRET_ACCESS_KEY" yaml:"secretKey"`
}

// Config holds the application configuration.
type Config struct {
	GitlabGroupID   int         `env:"GITLABGROUPID"   env-default:"0"                  yaml:"gitlabGroupID"`
	GitlabProjectID int         `env:"GITLABPROJECTID" env-default:"0"                  yaml:"gitlabProjectID"`
	GitlabToken     string      `env:"GITLAB_TOKEN"    env-required:"true"              yaml:"gitlabToken"`
	GitlabURI       string      `env:"GITLAB_URI"      env-default:"https://gitlab.com" yaml:"gitlabURI"`
	LocalPath       string      `env:"LOCALPATH"       env-default:""                   yaml:"localpath"`
	TmpDir          string      `env:"TMPDIR"          env-default:"/tmp"               yaml:"tmpdir"`
	Hooks           hooks.Hooks `yaml:"hooks"`
	S3cfg           S3Config    `yaml:"s3cfg"`
	NoLogTime       bool        `env:"NOLOGTIME"       env-default:"false"              yaml:"noLogTime"`
}

// NewConfigFromFile returns a new Config struct from the given file.
func NewConfigFromFile(filePath string) (*Config, error) {
	var cfg Config
	err := cleanenv.ReadConfig(filePath, &cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to read config from file %s: %w", filePath, err)
	}
	return &cfg, nil
}

// NewConfigFromEnv returns a new Config struct from the environment variables.
func NewConfigFromEnv() (*Config, error) {
	var cfg Config
	err := cleanenv.ReadEnv(&cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to read config from environment: %w", err)
	}
	return &cfg, nil
}

// IsS3ConfigValid returns true if the S3 config is valid.
func (c *Config) IsS3ConfigValid() bool {
	return len(c.S3cfg.BucketPath) > 0 && len(c.S3cfg.Region) > 0
}

// IsLocalConfigValid returns true if the local config is valid.
func (c *Config) IsLocalConfigValid() bool {
	return len(c.LocalPath) > 0
}

// IsConfigValid returns true if the config is valid.
func (c *Config) IsConfigValid() bool {
	valid := c.GitlabGroupID > 0 || c.GitlabProjectID > 0
	return (c.IsS3ConfigValid() || c.IsLocalConfigValid()) && valid && len(c.GitlabToken) > 0
}

func (c *Config) String() string {
	cyaml, err := yaml.Marshal(c)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
	}
	return string(cyaml)
}

// Usage prints the usage of the config.
func (c *Config) Usage() {
	f := cleanenv.Usage(c, nil)
	f()
}
