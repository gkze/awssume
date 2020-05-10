package awssume

import (
	"encoding/json"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestConfigFormat(t *testing.T) {
	testCases := []struct {
		formatStr string
		format    ConfigFormat
	}{
		{
			formatStr: "json",
			format:    JSON,
		},
		{
			formatStr: "yaml",
			format:    YAML,
		},
		{
			formatStr: "toml",
			format:    TOML,
		},
		{
			formatStr: "UNKNOWN",
			format:    Unknown,
		},
	}

	for _, tc := range testCases {
		var cfgFmt ConfigFormat
		cfgFmt.FromExt(tc.formatStr)
		assert.Equal(t, cfgFmt, tc.format)
		assert.Equal(t, tc.format.String(), tc.formatStr)
	}
}

func TestParseARN(t *testing.T) {
	testCases := []struct {
		arnString   string
		arnStruct   *ARN
		errExpected bool
	}{
		{
			arnString: "arn:aws:iam::000000000000:role/skunk",
			arnStruct: &ARN{&arn.ARN{
				Partition: "aws",
				Service:   "iam",
				Region:    "",
				AccountID: "000000000000",
				Resource:  "role/skunk",
			}},
			errExpected: false,
		},
		{
			arnString: "iNv@LiD",
			arnStruct: &ARN{&arn.ARN{
				Partition: "",
				Service:   "",
				Region:    "",
				AccountID: "",
				Resource:  "",
			}},
			errExpected: true,
		},
	}

	for _, tc := range testCases {
		parsed, err := ParseARN(tc.arnString)
		if tc.errExpected {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
		}

		assert.Equal(t, tc.arnStruct, &parsed)
	}
}

func TestARNString(t *testing.T) {
	testCases := []struct {
		arnString   string
		arnStruct   *ARN
		errExpected bool
	}{
		{
			arnStruct: &ARN{&arn.ARN{
				Partition: "aws",
				Service:   "iam",
				Region:    "",
				AccountID: "000000000000",
				Resource:  "role/skunk",
			}},
			arnString: "arn:aws:iam::000000000000:role/skunk",
		},
		{
			arnStruct: &ARN{&arn.ARN{
				Partition: "",
				Service:   "",
				Region:    "",
				AccountID: "",
				Resource:  "",
			}},
			arnString: "arn:::::",
		},
	}

	for _, tc := range testCases {
		assert.Equal(t, tc.arnStruct.String(), tc.arnString)
	}
}

func TestARNMarshalYAML(t *testing.T) {
	testCases := []struct {
		arnStruct   *ARN
		yamlString  string
		errExpected bool
	}{
		{
			arnStruct: &ARN{&arn.ARN{
				Partition: "aws",
				Service:   "iam",
				Region:    "",
				AccountID: "000000000000",
				Resource:  "role/skunk",
			}},
			yamlString:  "arn:aws:iam::000000000000:role/skunk",
			errExpected: false,
		},
		{
			arnStruct: &ARN{&arn.ARN{
				Partition: "",
				Service:   "",
				Region:    "",
				AccountID: "",
				Resource:  "",
			}},
			yamlString:  "arn:::::",
			errExpected: false,
		},
	}

	for _, tc := range testCases {
		str, err := tc.arnStruct.MarshalYAML()
		if tc.errExpected {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
		}

		assert.Equal(t, tc.yamlString, str)
	}
}

func TestARNUnmarshalYAML(t *testing.T) {
	testCases := []struct {
		arnYAML     []byte
		arnStruct   *ARN
		errExpected bool
		errStr      string
	}{
		{
			arnYAML: []byte("arn:aws:iam::587928718845:role/skunk"),
			arnStruct: &ARN{&arn.ARN{
				Partition: "aws",
				Service:   "iam",
				Region:    "",
				AccountID: "587928718845",
				Resource:  "role/skunk",
			}},
			errExpected: false,
			errStr:      "",
		},
		{
			arnYAML: []byte("arn:::::"),
			arnStruct: &ARN{&arn.ARN{
				Partition: "",
				Service:   "",
				Region:    "",
				AccountID: "",
				Resource:  "",
			}},
			errExpected: true,
			errStr:      "arn: invalid prefix",
		},
	}

	for _, tc := range testCases {
		a := &ARN{}
		err := yaml.Unmarshal(tc.arnYAML, a)
		if tc.errExpected {
			assert.Error(t, err)
			assert.EqualError(t, err, tc.errStr)
		} else {
			assert.NoError(t, err)
		}

		assert.Equal(t, tc.arnStruct, a)
	}
}

func TestARNMarshalJSON(t *testing.T) {
	testCases := []struct {
		arnStruct   *ARN
		jsonBytes   []byte
		errExpected bool
	}{
		{
			arnStruct: &ARN{&arn.ARN{
				Partition: "aws",
				Service:   "iam",
				Region:    "",
				AccountID: "000000000000",
				Resource:  "role/skunk",
			}},
			jsonBytes:   []byte("arn:aws:iam::000000000000:role/skunk"),
			errExpected: false,
		},
		{
			arnStruct: &ARN{&arn.ARN{
				Partition: "",
				Service:   "",
				Region:    "",
				AccountID: "",
				Resource:  "",
			}},
			jsonBytes:   []byte("arn:::::"),
			errExpected: false,
		},
	}

	for _, tc := range testCases {
		jsonBytes, err := tc.arnStruct.MarshalJSON()
		if tc.errExpected {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
		}

		assert.Equal(t, tc.jsonBytes, jsonBytes)
	}
}

func TestARNUnmarshalJSON(t *testing.T) {
	testCases := []struct {
		arnJSON     []byte
		arnStruct   *ARN
		errExpected bool
		errStr      string
	}{
		{
			arnJSON: []byte(`{"arn":"arn:aws:iam::587928718845:role/skunk"}`),
			arnStruct: &ARN{&arn.ARN{
				Partition: "aws",
				Service:   "iam",
				Region:    "",
				AccountID: "587928718845",
				Resource:  "role/skunk",
			}},
			errExpected: false,
			errStr:      "",
		},
		{
			arnJSON: []byte(`"arn:::::"`),
			arnStruct: &ARN{&arn.ARN{
				Partition: "",
				Service:   "",
				Region:    "",
				AccountID: "",
				Resource:  "",
			}},
			errExpected: true,
			errStr:      "arn: invalid prefix",
		},
	}

	for _, tc := range testCases {
		a := &ARN{}
		err := json.Unmarshal(tc.arnJSON, a)
		if tc.errExpected {
			assert.Error(t, err)
			assert.EqualError(t, err, tc.errStr)
		} else {
			assert.NoError(t, err)
		}

		assert.Equal(t, tc.arnStruct, a)
	}
}

func TestRoleGetAlias(t *testing.T) {
	testCases := []struct {
		role  *Role
		alias string
	}{
		{
			role:  &Role{Alias: "skunk"},
			alias: "skunk",
		},
	}

	for _, tc := range testCases {
		assert.Equal(t, tc.role.GetAlias(), tc.alias)
	}
}

func TestRoleSetAlias(t *testing.T) {
	testCases := []struct {
		role  *Role
		alias string
	}{
		{
			role:  &Role{Alias: ""},
			alias: "skunk",
		},
	}

	for _, tc := range testCases {
		tc.role.SetAlias(tc.alias)
		assert.Equal(t, tc.alias, tc.role.Alias)
	}
}

func TestRoleGetARN(t *testing.T) {
	testCases := []struct {
		role *Role
		arn  *ARN
	}{
		{
			role: &Role{ARN: &ARN{&arn.ARN{
				Partition: "aws",
				Service:   "iam",
				Region:    "",
				AccountID: "000000000000",
				Resource:  "user/skunk",
			}}},
			arn: &ARN{&arn.ARN{
				Partition: "aws",
				Service:   "iam",
				Region:    "",
				AccountID: "000000000000",
				Resource:  "user/skunk",
			}},
		},
	}

	for _, tc := range testCases {
		assert.Equal(t, tc.role.GetARN(), tc.arn)
	}
}

func TestRoleSetARN(t *testing.T) {
	testCases := []struct {
		role *Role
		arn  *ARN
	}{
		{
			role: &Role{ARN: &ARN{&arn.ARN{
				Partition: "aws",
				Service:   "iam",
				Region:    "",
				AccountID: "000000000000",
				Resource:  "user/skunk",
			}}},
			arn: &ARN{&arn.ARN{
				Partition: "aws",
				Service:   "iam",
				Region:    "",
				AccountID: "000000000000",
				Resource:  "user/skunk",
			}},
		},
	}

	for _, tc := range testCases {
		tc.role.SetARN(tc.arn)
		assert.Equal(t, tc.role.ARN, tc.arn)
	}
}

func TestRoleGetSessionName(t *testing.T) {
	testCases := []struct {
		role        *Role
		sessionName string
	}{
		{
			role:        &Role{SessionName: "skunk"},
			sessionName: "skunk",
		},
	}

	for _, tc := range testCases {
		assert.Equal(t, tc.role.GetSessionName(), tc.sessionName)
	}
}

func TestRoleSetSessionName(t *testing.T) {
	testCases := []struct {
		role        *Role
		sessionName string
	}{
		{
			role:        &Role{SessionName: "skunk"},
			sessionName: "skunk",
		},
	}

	for _, tc := range testCases {
		tc.role.SetSessionName(tc.sessionName)
		assert.Equal(t, tc.role.SessionName, tc.sessionName)
	}
}
