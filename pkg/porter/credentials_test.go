package porter

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cnabio/cnab-go/credentials"
	"github.com/cnabio/cnab-go/secrets/host"
	"github.com/cnabio/cnab-go/utils/crud"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"get.porter.sh/porter/pkg/printer"
)

func TestGenerateNoName(t *testing.T) {
	p := NewTestPorter(t)
	p.CNAB = &TestCNABProvider{}

	opts := CredentialOptions{
		Silent: true,
	}
	err := p.GenerateCredentials(opts)
	require.NoError(t, err, "no error should have existed")
	creds, err := p.Credentials.Read("testbundle")
	require.NoError(t, err, "expected credential to have been generated")
	var zero time.Time
	assert.True(t, zero.Before(creds.Created), "expected Credentials.Created to be set")
	assert.True(t, creds.Created.Equal(creds.Modified), "expected Credentials.Created to be initialized to Credentials.Modified")
}

func TestGenerateNameProvided(t *testing.T) {
	p := NewTestPorter(t)
	p.CNAB = &TestCNABProvider{}

	opts := CredentialOptions{
		Silent: true,
	}
	opts.Name = "kool-kred"

	err := p.GenerateCredentials(opts)
	require.NoError(t, err, "no error should have existed")
	_, err = p.Credentials.Read("kool-kred")
	require.NoError(t, err, "expected credential to have been generated")
}

func TestGenerateBadNameProvided(t *testing.T) {
	p := NewTestPorter(t)
	p.CNAB = &TestCNABProvider{}

	opts := CredentialOptions{
		Silent: true,
	}
	opts.Name = "this.isabadname"

	err := p.GenerateCredentials(opts)
	require.Error(t, err, "name is invalid, we should have had an error")
	_, err = p.Credentials.Read("this.isabadname")
	require.Error(t, err, "expected credential to not exist")
}

type CredentialsListTest struct {
	name         string
	format       printer.Format
	wantContains []string
	errorMsg     string
}

func TestCredentialsList_None(t *testing.T) {
	testcases := []CredentialsListTest{
		{
			name:         "invalid format",
			format:       "wingdings",
			wantContains: []string{},
			errorMsg:     "invalid format: wingdings",
		},
		{
			name:         "json",
			format:       printer.FormatJson,
			wantContains: []string{"[]\n"},
			errorMsg:     "",
		},
		{
			name:         "yaml",
			format:       printer.FormatYaml,
			wantContains: []string{"[]\n\n"},
			errorMsg:     "",
		},
		{
			name:         "table",
			format:       printer.FormatTable,
			wantContains: []string{"NAME   MODIFIED\n"},
			errorMsg:     "",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			p := NewTestPorter(t)

			p.CNAB = &TestCNABProvider{}

			listOpts := ListOptions{}
			listOpts.Format = tc.format
			err := p.ListCredentials(listOpts)
			if tc.errorMsg != "" {
				require.Equal(t, err.Error(), tc.errorMsg)
			} else {
				require.NoError(t, err, "no error should have existed")
			}

			gotOutput := p.TestConfig.TestContext.GetOutput()
			for _, contains := range tc.wantContains {
				require.Contains(t, gotOutput, contains)
			}
		})
	}
}

func TestCredentialsList(t *testing.T) {
	testcases := []CredentialsListTest{
		{
			name:         "json",
			format:       printer.FormatJson,
			wantContains: []string{`"name": "kool-kreds"`},
			errorMsg:     "",
		},
		{
			name:         "yaml",
			format:       printer.FormatYaml,
			wantContains: []string{`- name: kool-kreds`},
			errorMsg:     "",
		},
		{
			name:   "table",
			format: printer.FormatTable,
			wantContains: []string{`NAME         MODIFIED
kool-kreds   2019-06-24`},
			errorMsg: "",
		},
		{
			name:         "error",
			format:       printer.FormatTable,
			wantContains: []string{},
			errorMsg:     "",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			p := NewTestPorter(t)
			p.CNAB = &TestCNABProvider{}

			p.TestCredentials.AddTestCredentialsDirectory("testdata/test-creds")

			listOpts := ListOptions{}
			listOpts.Format = tc.format
			err := p.ListCredentials(listOpts)
			require.NoError(t, err)

			gotOutput := p.TestConfig.TestContext.GetOutput()
			for _, contains := range tc.wantContains {
				require.Contains(t, gotOutput, contains)
			}
		})
	}
}

func TestGenerateNoCredentialDirectory(t *testing.T) {
	p := NewTestPorter(t)
	home := p.UseFilesystem()

	// Write credentials to the real file system for this test, not sure if this test is worth keeping
	fsStore := crud.NewFileSystemStore(home, "json")
	credStore := credentials.NewCredentialStore(fsStore)
	p.TestCredentials.CredentialStorage.CredentialsStore = &credStore

	p.TestConfig.SetupPorterHome()
	p.CNAB = &TestCNABProvider{}

	opts := CredentialOptions{
		Silent: true,
	}
	opts.Name = "name"

	// Check if the credentials directory exists in the FS. It shouldn't.
	credDir := filepath.Join(home, "credentials")
	credDirExists, err := p.Porter.Context.FileSystem.DirExists(credDir)
	require.NoError(t, err, "shouldn't have failed on dir exists")
	require.False(t, credDirExists, "there should not have been a credential directory for this test")

	// Now generate the credentials. After completion, the directory should now exist. It should be
	// created if it does not exit
	err = p.GenerateCredentials(opts)
	require.NoError(t, err, "credential generation should have been successful")
	credDirExists, err = p.Porter.Context.FileSystem.DirExists(credDir)
	require.NoError(t, err, "shouldn't have gotten an error checking credential directory after generate")
	assert.True(t, credDirExists, "should have been a credential directory after the generation")

	// Verify that the credential was actually created.
	c, err := p.Credentials.Read("name")
	require.NoError(t, err, "the credential 'name' was not generated")
	assert.NotNil(t, c, "the credential should have a value after being read")
}

func TestGenerateCredentialDirectoryExists(t *testing.T) {
	p := NewTestPorter(t)
	p.TestConfig.SetupPorterHome()
	p.CNAB = &TestCNABProvider{}

	opts := CredentialOptions{
		Silent: true,
	}
	opts.Name = "name"

	// Create the credentials directory
	home, err := p.Config.GetHomeDir()
	require.NoError(t, err, "should have been able to get home directory path")
	credDir := filepath.Join(home, "credentials")
	err = p.Config.FileSystem.MkdirAll(credDir, 0600)
	require.NoError(t, err, "should have been able to make directory path")

	// Verify the directory does in fact, exist.
	credDirExists, err := p.Porter.Context.FileSystem.DirExists(credDir)
	require.NoError(t, err, "shouldn't have failed on dir exists")
	require.True(t, credDirExists, "there should have been a credential directory for this test")

	// Generate the credential now. The directory does exist, so there should be no error.
	err = p.GenerateCredentials(opts)
	assert.NoError(t, err, "credential generation should have been successful")
	credDirExists, err = p.Porter.Context.FileSystem.DirExists(credDir)
	assert.NoError(t, err, "shouldn't have gotten an error checking credential directory after generate")
	assert.True(t, credDirExists, "should have been a credential directory after the generation")

	// Verify that the credential was actually created.
	_, err = p.Credentials.Read("name")
	assert.NoError(t, err, "the credential 'name' was not generated")
}

type CredentialShowTest struct {
	name       string
	format     printer.Format
	wantOutput string
}

func TestShowCredential_NotFound(t *testing.T) {
	p := NewTestPorter(t)
	p.TestConfig.SetupPorterHome()
	p.CNAB = &TestCNABProvider{}

	opts := CredentialShowOptions{
		PrintOptions: printer.PrintOptions{
			Format: printer.FormatTable,
		},
		Name: "non-existent-cred",
	}

	err := p.ShowCredential(opts)
	assert.Error(t, err, "an error should have occurred")
	assert.EqualError(t, err, "Credential set does not exist")
}

func TestShowCredential_Found(t *testing.T) {
	testcases := []CredentialShowTest{
		{
			name:   "json",
			format: printer.FormatJson,
			wantOutput: `{
  "name": "kool-kreds",
  "created": "2019-06-24T16:07:57.415378-05:00",
  "modified": "2019-06-24T16:07:57.415378-05:00",
  "credentials": [
    {
      "name": "kool-config",
      "source": {
        "path": "/path/to/kool-config"
      }
    },
    {
      "name": "kool-envvar",
      "source": {
        "env": "KOOL_ENV_VAR"
      }
    },
    {
      "name": "kool-cmd",
      "source": {
        "command": "echo 'kool'"
      }
    },
    {
      "name": "kool-val",
      "source": {
        "value": "kool"
      }
    }
  ]
}
`,
		},
		{
			name:   "yaml",
			format: printer.FormatYaml,
			wantOutput: `name: kool-kreds
created: 2019-06-24T16:07:57.415378-05:00
modified: 2019-06-24T16:07:57.415378-05:00
credentials:
- name: kool-config
  source:
    path: /path/to/kool-config
- name: kool-envvar
  source:
    env: KOOL_ENV_VAR
- name: kool-cmd
  source:
    command: echo 'kool'
- name: kool-val
  source:
    value: kool

`,
		},
		{
			name:   "table",
			format: printer.FormatTable,
			wantOutput: `Name: kool-kreds
Created: 2019-06-24
Modified: 2019-06-24

--------------------------------------------------
  Name         Local Source          Source Type  
--------------------------------------------------
  kool-config  /path/to/kool-config  path         
  kool-envvar  KOOL_ENV_VAR          env          
  kool-cmd     echo 'kool'           command      
  kool-val     kool                  value        
`,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			p := NewTestPorter(t)
			p.TestConfig.SetupPorterHome()
			p.CNAB = &TestCNABProvider{}

			opts := CredentialShowOptions{
				PrintOptions: printer.PrintOptions{
					Format: tc.format,
				},
				Name: "kool-kreds",
			}

			p.TestCredentials.AddTestCredentialsDirectory("testdata/test-creds")

			err := p.ShowCredential(opts)
			assert.NoError(t, err, "an error should not have occurred")
			gotOutput := p.TestConfig.TestContext.GetOutput()
			assert.Equal(t, tc.wantOutput, gotOutput)
		})
	}
}

func TestShowCredential_PreserveCase(t *testing.T) {
	opts := CredentialShowOptions{}
	opts.RawFormat = string(printer.FormatTable)

	err := opts.Validate([]string{"HELLO"})
	require.NoError(t, err, "Validate failed")
	assert.Equal(t, "HELLO", opts.Name, "Validate should preserve the credential set name case")
}

type SourceTest struct {
	name      string
	source    credentials.Source
	wantValue string
	wantType  string
}

func TestGetCredentialSourceValueAndType(t *testing.T) {
	testcases := []SourceTest{
		{
			name: "Source: EnvVar",
			source: credentials.Source{
				Key:   host.SourceEnv,
				Value: "ENVY",
			},
			wantValue: "ENVY",
			wantType:  "env",
		},
		{
			name: "Source: Path",
			source: credentials.Source{
				Key:   host.SourcePath,
				Value: "/pathy/patheson",
			},
			wantValue: "/pathy/patheson",
			wantType:  "path",
		},
		{
			name: "Source: Command",
			source: credentials.Source{
				Key:   host.SourceCommand,
				Value: "sed s/true/false/g",
			},
			wantValue: "sed s/true/false/g",
			wantType:  "command",
		},
		{
			name: "Source: Value",
			source: credentials.Source{
				Key:   host.SourceValue,
				Value: "abc123",
			},
			wantValue: "abc123",
			wantType:  "value",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			sv, st := GetCredentialSourceValueAndType(tc.source)
			assert.Equal(t, tc.wantValue, sv)
			assert.Equal(t, tc.wantType, st)
		})
	}
}

func TestCredentialsDelete(t *testing.T) {
	testcases := []struct {
		name       string
		credName   string
		wantOutput string
	}{{
		name:     "delete",
		credName: "kool-kreds",
	}, {
		name:       "error",
		credName:   "noop-kreds",
		wantOutput: "credential set does not exist",
	}}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			p := NewTestPorter(t)
			p.CNAB = &TestCNABProvider{}

			p.TestCredentials.AddTestCredentialsDirectory("testdata/test-creds")

			opts := CredentialDeleteOptions{Name: tc.credName}
			err := p.DeleteCredential(opts)
			require.NoError(t, err, "no error should have existed")

			_, err = p.TestCredentials.Read(tc.credName)
			assert.Error(t, err, "credential set still exists")

			gotOutput := p.TestConfig.TestContext.GetOutput()
			assert.Equal(t, tc.wantOutput, strings.TrimSpace(gotOutput))
		})
	}
}
