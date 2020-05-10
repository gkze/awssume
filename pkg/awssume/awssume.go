package awssume

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"sort"
	"strings"
	"syscall"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/naoina/toml"
	"github.com/spf13/afero"
	"gopkg.in/yaml.v3"
)

var (
	// ErrCommandMissing is returned when there is no command passed
	ErrCommandMissing error = errors.New("command was not passed")

	// ErrNoArgs is returned when there are no arguments passed to the command
	ErrNoArgs error = errors.New("no arguments passed")

	// ErrNoShell is returned when a shell cannot be found
	ErrNoShell error = errors.New("no shell found")

	// ErrUnsupportedConfigFormat is returned when an unsupported configuration
	// file format is specified
	ErrUnsupportedConfigFormat error = errors.New("unsupported config file format")

	// ErrMultipleConfigs is returns when multiple configuration fils are
	// detected
	ErrMultipleConfigs error = errors.New("multiple configuration files detected")

	// ErrUnexpected is returned when an unexpected error occurs
	ErrUnexpected error = errors.New("unexpected error occurred")
)

// errors
const (
	// ErrCheckFileExists is returned when error is encountered while checking
	// for the existence of the specified format configuration file on the
	// filesystem
	ErrCheckFileExists string = "error occurred while checking for %s configuration file existence: %w"

	// ErrCreatingFile is returned when an error is encountered while creating
	// the specified file
	ErrCreatingFile string = "error creating file: %w"

	// ErrExecCmd is returned when an error is encountered while executing a
	// passed command
	ErrExecCmd string = "error executing command %s (args %s): %w"

	// ErrExeNotFound is returned when the specified command executable cannot
	// be located in $PATH
	ErrExeNotFound string = "executable %s not found: %w"

	// ErrGetRoleByAlias is returned when the Role cannot be retrieved given its
	// alias
	ErrGetRoleByAlias string = "error getting Role for alias %s: %w"

	// ErrLoadAWSConfig is returned when AWS configuration cannot be loaded
	ErrLoadAWSConfig string = "error loading AWS config: %w"

	// ErrMarshal is returned when an error is encountered during serialization
	ErrMarshal string = "error serializing: %w"

	// ErrNewConfig is returned when a new aws.Config struct cannot be created
	ErrNewConfig string = "error creating config: %w"

	// ErrReadingFile is returned when an error reading a specified file is
	// encountered
	ErrReadingFile string = "error reading file %s: %w"

	// ErrReadingFromByteBuf is returned when an error is encountered reading
	// from a byte buffer
	ErrReadingFromByteBuf string = "error reading from byte buffer: %w"

	// ErrRoleExists is returned when a specified Role already exists in the
	// configuration
	ErrRoleExists string = "role %s already exists: %w"

	// ErrRoleNotFound is returned when the specified file cannot be found
	ErrRoleNotFound string = "no role with alias %s found"

	// ErrSTSAssumeRole is returned when an error is encountered while
	// performing the sts:AssumeRole operation
	ErrSTSAssumeRole string = "error assuming Role %s: %w"

	// ErrUnmarshal is returned when an error is encountered during
	// deserialization
	ErrUnmarshal string = "error deserializing: %w"

	// ErrUnmarshalARN is returned when an error is encountered during
	// deserialization of an ARN
	ErrUnmarshalARN string = "error deserializing ARN: %w"

	// ErrWritingToFile is returned when an error is encountered while writing
	// to a file
	ErrWritingToFile string = "error writing to file %s: %w"
)

// SDK defaults
const (
	// AWS Access Key ID environment variable name
	AWSAccessKeyIDEnvVar string = "AWS_ACCESS_KEY_ID"

	// AWS Secret Access Key environment variable name
	AWSSecreteAccessKeyEnvVar string = "AWS_SECRET_ACCESS_KEY"

	// AWS Security Token (otherwise known as Session Token) environment
	// variable name
	AWSSecurityTokenEnvVar string = "AWS_SECURITY_TOKEN"

	// AWS Session Token is the STS Session Token received as part of an
	// sts:AssumeRole API call
	AWSSessionTokenEnvVar string = "AWS_SESSION_TOKEN"
)

// ConfigFormat describes the various supported configuration file formats
type ConfigFormat int

const (
	// JSON //
	JSON ConfigFormat = iota

	// TOML //
	TOML

	// YAML //
	YAML

	// Unknown //
	Unknown
)

// FromExt creates a ConfigFormat from its associated well-known extension
func (cf *ConfigFormat) FromExt(ext string) {
	switch strings.TrimPrefix(ext, ".") {
	case "json":
		*cf = JSON
	case "toml":
		*cf = TOML
	case "yaml", "yml":
		*cf = YAML
	default:
		*cf = Unknown
	}
}

// String returns the string file extension for the config format
func (cf ConfigFormat) String() string {
	switch cf {
	case JSON:
		return "json"
	case TOML:
		return "toml"
	case YAML:
		return "yaml"
	case Unknown:
		return "UNKNOWN"
	default:
		return ""
	}
}

const (
	// DefaultConfigFilePath is the default filesystem path where the configuration
	// file is located
	DefaultConfigFilePath string = ".config/awssume"

	// DefaultIndent is the default indentation to use when serializing into
	// various formats
	DefaultIndent int = 2
)

// ARN is a wrapper around github.com/aws/aws-sdk-go-v2/aws/arn.ARN to allow
// custom serdes
type ARN struct{ *arn.ARN }

// ParseARN ...
func ParseARN(s string) (ARN, error) {
	a, err := arn.Parse(s)
	return ARN{&a}, err
}

func (a *ARN) String() string { return a.ARN.String() }

// MarshalYAML seriaalizes to YAML by stringifying the ARN
func (a *ARN) MarshalYAML() (interface{}, error) { return a.String(), nil }

// UnmarshalYAML deserializes the ARN from JSON by parsing it
func (a *ARN) UnmarshalYAML(n *yaml.Node) error {
	var err error
	*a, err = ParseARN(n.Value)
	return err
}

// MarshalJSON serializes to JSON by stringifying the ARN
func (a *ARN) MarshalJSON() ([]byte, error) { return []byte(a.String()), nil }

// UnmarshalJSON deserializes the ARN from JSON by parsing it
func (a *ARN) UnmarshalJSON(bs []byte) error {
	intermediate := struct {
		Arn string `json:"arn"`
	}{}
	err := json.Unmarshal(bs, &intermediate)
	*a, err = ParseARN(intermediate.Arn)
	return err
}

// Compile-time interface-implementation compatibility checks
var (
	_ json.Marshaler   = (*ARN)(nil)
	_ json.Unmarshaler = (*ARN)(nil)
	_ yaml.Marshaler   = (*ARN)(nil)
	_ yaml.Unmarshaler = (*ARN)(nil)
)

// IRole interface describes operations against IAM Roles
type IRole interface {
	// GetAlias gets the Role's human friendly name
	GetAlias() string

	// SetAlias sets the Role's human friendly name
	SetAlias(string)

	// GetARN retrieves the Role's ARN (Amazon Resource Name)
	GetARN() *ARN

	// SetARN sets the Role's ARN (Amazon Resource Name)
	SetARN(*ARN)

	// GetSessionName gets the Role's Session name
	GetSessionName() string

	// SetRoleSessionName sets the Role's session name
	SetSessionName(string)
}

// IConfig interface describes operations against configuration source(s) for
// Role(s).
type IConfig interface {
	// GetPath returns the configuration filesystem path.
	GetPath() string

	// SetPath sets the configuration filesystem path.
	SetPath(string)

	// GetFormat returns the configuration format
	GetFormat() ConfigFormat

	// SetFormat sets the configuraion format
	SetFormat(ConfigFormat)

	// Save serializes the configuration to the filesystem
	Save() error

	// ListRoles returns a list of all configured Roles
	GetRoles() []IRole

	// GetRoleByAlias returns a Role by its configured alias
	GetRoleByAlias(string) (IRole, error)

	// RemoveRoleByAlias removes a specified Role by its configured alias
	RemoveRoleByAlias(string) error

	// AddRole configures a specified Role
	AddRole(IRole) error

	// UpdateRoleByAlias updates the specified Role by its alias and the updated Role
	UpdateRoleByAlias(string, IRole) error

	// ExecRole allows executing subprocesses by assuming the target Role
	// through STS and providing the resulting credentials as environment
	// variables
	ExecRole(alias string, sessionDuration int64, command string, args []string) error
}

// Role struct implements the Role interface
type Role struct {
	// alias refers to a human-friendly Role identifier
	Alias string `json:"alias" toml:"alias" yaml:"alias"`

	// rn is the Amazon Resource Name
	ARN *ARN `json:"arn" toml:"arn" yaml:"arn"`

	// sessionName is the the string to use for the STS Session when assuming
	// the target Role
	SessionName string `json:"session_name" toml:"session_name" yaml:"session_name"`
}

// GetAlias returns the Role's alias
func (r *Role) GetAlias() string { return r.Alias }

// SetAlias sets the Role's alias
func (r *Role) SetAlias(alias string) { r.Alias = alias }

// GetARN returns the Role's Amazon Resource Name
func (r *Role) GetARN() *ARN { return r.ARN }

// SetARN sets the Role's Amazon Resource Name
func (r *Role) SetARN(a *ARN) { r.ARN = a }

// GetSessionName gets the Role's STS Session Name
func (r *Role) GetSessionName() string { return r.SessionName }

// SetSessionName sets the Role's STS Session Name
func (r *Role) SetSessionName(sname string) { r.SessionName = sname }

// Compile-time interface-implementation compatibility check
var _ IRole = (*Role)(nil)

// Config represents the configured Roles
type Config struct {
	// Path represents the filesystem path where the configuration is located
	Path string `json:"-" toml:"-" yaml:"-"`

	// Format describes the current configuration format
	Format ConfigFormat `json:"-" toml:"-" yaml:"-"`

	// Roles holds all of the configured Roles
	Roles []*Role `json:"roles" toml:"roles" yaml:"roles"`

	// fs is an afero.Fs for filesystem operations
	fs afero.Fs
}

// GetPath returns the configuration filesystem path
func (c *Config) GetPath() string { return c.Path }

// SetPath sets the configuration filesystem path
func (c *Config) SetPath(p string) { c.Path = p }

// GetFormat returns the configuration format
func (c *Config) GetFormat() ConfigFormat { return c.Format }

// SetFormat sets the configuraion format
func (c *Config) SetFormat(format ConfigFormat) { c.Format = format }

// Save serializes the configuration to the filesystem
func (c *Config) Save() error {
	marshalFn := func(v interface{}) ([]byte, error) { return nil, nil }
	switch c.GetFormat() {
	case JSON:
		marshalFn = json.Marshal
	case TOML:
		marshalFn = toml.Marshal
	case YAML:
		marshalFn = yaml.Marshal
	}

	bytes, err := marshalFn(c)
	if err != nil {
		return fmt.Errorf(ErrMarshal, err)
	}

	return afero.WriteFile(
		c.fs,
		strings.Join([]string{c.GetPath(), c.GetFormat().String()}, "."),
		bytes, os.FileMode(0644),
	)
}

// GetRoles returns a list of all configured Roles
func (c *Config) GetRoles() []IRole {
	roles := make([]IRole, len(c.Roles))

	for i, r := range c.Roles {
		roles[i] = r
	}

	return roles
}

// GetRoleByAlias returns a Role by its configured alias
func (c *Config) GetRoleByAlias(alias string) (IRole, error) {
	for _, role := range c.Roles {
		if role.GetAlias() == alias {
			return role, nil
		}
	}

	return nil, fmt.Errorf(ErrRoleNotFound, alias)
}

// RemoveRoleByAlias removes a specified Role by its configured alias
func (c *Config) RemoveRoleByAlias(alias string) error {
	for i, r := range c.GetRoles() {
		if r.GetAlias() == alias {
			// https://github.com/golang/go/wiki/SliceTricks
			c.Roles = c.Roles[:i+copy(c.Roles[i:], c.Roles[i+1:])]
			return nil
		}
	}

	return fmt.Errorf(ErrRoleNotFound, alias)
}

// AddRole configures a specified Role
func (c *Config) AddRole(r IRole) error {
	existingRole, err := c.GetRoleByAlias(r.GetAlias())
	if err == nil && existingRole != nil {
		return fmt.Errorf(ErrRoleExists, r.GetAlias(), err)
	}

	roles := append(c.Roles, r.(*Role))
	sort.Slice(roles, func(i, j int) bool {
		return roles[i].GetAlias() > roles[j].GetAlias()
	})

	c.Roles = roles

	return nil
}

// UpdateRoleByAlias updates the specified Role by its alias and the updated
// Role
func (c *Config) UpdateRoleByAlias(alias string, r IRole) error {
	err := c.RemoveRoleByAlias(alias)
	if err != nil {
		return err
	}

	c.Roles = append(c.Roles, r.(*Role))

	return nil
}

// ExecRole allows executing subprocesses by assuming the target Role
// through STS and providing the resulting credentials as environment
// variables
func (c *Config) ExecRole(
	alias string,
	sessionDuration int64,
	command string,
	arguments []string,
) error {
	awsCfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		return fmt.Errorf(ErrLoadAWSConfig, err)
	}

	resRole, err := c.GetRoleByAlias(alias)
	if err != nil {
		return fmt.Errorf(ErrGetRoleByAlias, alias, err)
	}

	res, err := sts.New(awsCfg).AssumeRoleRequest(&sts.AssumeRoleInput{
		DurationSeconds: aws.Int64(sessionDuration),
		RoleArn:         aws.String(resRole.GetARN().String()),
		RoleSessionName: aws.String(resRole.GetSessionName()),
	}).Send(context.Background())
	if err != nil {
		return fmt.Errorf(ErrSTSAssumeRole, resRole.GetARN(), err)
	}

	cmdToRun := exec.Command(command, arguments...)
	cmdToRun.Stdin = os.Stdin
	cmdToRun.Stdout = os.Stdout
	cmdToRun.Stderr = os.Stderr
	cmdToRun.Env = append(os.Environ(), (*NewEnvMap(map[string]string{
		AWSAccessKeyIDEnvVar:      *res.Credentials.AccessKeyId,
		AWSSecreteAccessKeyEnvVar: *res.Credentials.SecretAccessKey,
		AWSSecurityTokenEnvVar:    *res.Credentials.SessionToken,
		AWSSessionTokenEnvVar:     *res.Credentials.SessionToken,
	})).StringSlice()...)

	// Forward SIGINT, SIGTERM, SIGKILL to the child command
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT, syscall.SIGKILL)

	go func() {
		sig := <-sigChan
		if cmdToRun.Process != nil {
			cmdToRun.Process.Signal(sig)
		}
	}()

	var waitStatus syscall.WaitStatus
	if err := cmdToRun.Run(); err != nil {
		if err != nil {
			return err
		}
		if exitError, ok := err.(*exec.ExitError); ok {
			waitStatus = exitError.Sys().(syscall.WaitStatus)
			os.Exit(waitStatus.ExitStatus())
		}
	}
	return nil
}

var _ IConfig = (*Config)(nil)

// NewConfigOpts is an option set passed to the config constructors
type NewConfigOpts struct {
	Fs   afero.Fs
	Path string
}

// NewConfig parses a config object from a specified path
func NewConfig(opts *NewConfigOpts) (*Config, error) {
	cfg := &Config{
		Format: Unknown,
		Path:   strings.TrimSuffix(opts.Path, path.Ext(opts.Path)),
		fs:     opts.Fs,
	}

	JSONFilePath := strings.Join([]string{cfg.GetPath(), JSON.String()}, ".")
	JSONFileExists, JSONFileExistsErr := afero.Exists(cfg.fs, JSONFilePath)
	if JSONFileExistsErr != nil {
		return nil, fmt.Errorf(ErrCheckFileExists, JSONFilePath, JSONFileExistsErr)
	}

	YAMLFilePath := strings.Join([]string{cfg.GetPath(), YAML.String()}, ".")
	YAMLFileExists, YAMLFileExistsErr := afero.Exists(cfg.fs, YAMLFilePath)
	if YAMLFileExistsErr != nil {
		return nil, fmt.Errorf(ErrCheckFileExists, YAMLFilePath, YAMLFileExistsErr)
	}

	TOMLFilePath := strings.Join([]string{cfg.GetPath(), TOML.String()}, ".")
	TOMLFileExists, TOMLFileExistsErr := afero.Exists(cfg.fs, TOMLFilePath)
	if TOMLFileExistsErr != nil {
		return nil, fmt.Errorf(ErrCheckFileExists, TOMLFilePath, TOMLFileExistsErr)
	}

	if JSONFileExists && YAMLFileExists && TOMLFileExists ||
		JSONFileExists && YAMLFileExists ||
		JSONFileExists && TOMLFileExists ||
		YAMLFileExists && TOMLFileExists {
		return nil, ErrMultipleConfigs
	}

	if JSONFileExists {
		cfg.SetFormat(JSON)
	} else if YAMLFileExists {
		cfg.SetFormat(YAML)
	} else if TOMLFileExists {
		cfg.SetFormat(TOML)
	} else {
		cfg.SetFormat(YAML)
	}

	cfgPath := strings.Join([]string{cfg.GetPath(), cfg.GetFormat().String()}, ".")
	bytes, err := afero.ReadFile(cfg.fs, cfgPath)
	if err != nil {
		if os.IsNotExist(err) {
			f, err := cfg.fs.Create(cfgPath)
			defer f.Close()

			if err != nil {
				return nil, fmt.Errorf(ErrCreatingFile, err)
			}
		} else {
			return nil, fmt.Errorf(ErrReadingFile, cfgPath, err)
		}
	}

	unmarshalFn := func(data []byte, target interface{}) error { return nil }
	switch cfg.GetFormat() {
	case JSON:
		unmarshalFn = json.Unmarshal
	case YAML:
		unmarshalFn = yaml.Unmarshal
	case TOML:
		unmarshalFn = toml.Unmarshal
	case Unknown:
		return nil, ErrUnsupportedConfigFormat
	default:
		return nil, ErrUnexpected
	}

	err = unmarshalFn(bytes, cfg)
	if err != nil {
		return nil, fmt.Errorf(ErrUnmarshal, err)
	}

	return cfg, nil
}

// IEnvMap describes a map of environment variables that can be transformed
// to a string slice
type IEnvMap interface {
	StringSlice() []string
}

// EnvMap implements IEnvMap
type EnvMap struct {
	m map[string]string
}

// StringSlice returns the string slice representation of the environment
// variable map
func (e *EnvMap) StringSlice() []string {
	results := []string{}

	for k, v := range e.m {
		results = append(results, strings.Join([]string{k, v}, "="))
	}

	return results
}

// NewEnvMap creates a new EnvMap from a passed map
func NewEnvMap(m map[string]string) *EnvMap { return &EnvMap{m: m} }

var _ IEnvMap = (*EnvMap)(nil)

// GetShell tries to return a shell to use, starting with a configured one and
// falling back to defaults, erroring out if nothing is found
func GetShell() (string, error) {
	shell := os.Getenv("SHELL")

	if shell == "" {
		var err error
		shell, err = exec.LookPath("/bin/bash")
		if err != nil {
			shell, err = exec.LookPath("/bin/sh")
			if err != nil {
				return "", ErrNoShell
			}
		}
	}

	return shell, nil
}
