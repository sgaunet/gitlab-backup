package config

import (
	"fmt"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/sgaunet/gitlab-backup/pkg/hooks"
	"gopkg.in/yaml.v3"
)

type S3Config struct {
	Endpoint   string `yaml:"endpoint" env:"S3ENDPOINT" env-default:""`
	BucketName string `yaml:"bucketName" env:"S3BUCKETNAME" env-default:""`
	BucketPath string `yaml:"bucketPath" env:"S3BUCKETPATH" env-default:""`
	Region     string `yaml:"region" env:"S3REGION" env-default:""`
	AccessKey  string `env:"AWS_ACCESS_KEY_ID"`
	SecretKey  string `env:"AWS_SECRET_ACCESS_KEY"`
}

type Config struct {
	GitlabGroupID   int         `yaml:"gitlabGroupID" env:"GITLABGROUPID" env-default:"0"`
	GitlabProjectID int         `yaml:"gitlabProjectID" env:"GITLABPROJECTID" env-default:"0"`
	GitlabToken     string      `env:"GITLAB_TOKEN" env-required:"true"`
	GitlabURI       string      `env:"GITLAB_URI" env-default:"https://gitlab.com"`
	LocalPath       string      `yaml:"localpath" env:"LOCALPATH" env-default:""`
	TmpDir          string      `yaml:"tmpdir" env:"TMPDIR" env-default:"/tmp"`
	Hooks           hooks.Hooks `yaml:"hooks"`
	S3cfg           S3Config    `yaml:"s3cfg"`
	NoLogTime       bool        `yaml:"noLogTime" env:"NOLOGTIME" env-default:"false"`
}

// NewConfigFromFile returns a new Config struct from the given file
func NewConfigFromFile(filePath string) (*Config, error) {
	var cfg Config
	err := cleanenv.ReadConfig(filePath, &cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

// NewConfigFromEnv returns a new Config struct from the environment variables
func NewConfigFromEnv() (*Config, error) {
	var cfg Config
	err := cleanenv.ReadEnv(&cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

// IsS3ConfigValid returns true if the S3 config is valid
func (c *Config) IsS3ConfigValid() bool {
	return len(c.S3cfg.BucketPath) > 0 && len(c.S3cfg.Region) > 0
}

// IsLocalConfigValid returns true if the local config is valid
func (c *Config) IsLocalConfigValid() bool {
	return len(c.LocalPath) > 0
}

// IsConfigValid returns true if the config is valid
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

// Usage prints the usage of the config
func (c *Config) Usage() {
	f := cleanenv.Usage(c, nil)
	f()
}
