package convert_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	v1 "github.com/jenkins-x-plugins/jx-secret/pkg/apis/external/v1"
	"github.com/jenkins-x-plugins/jx-secret/pkg/apis/mapping/v1alpha1"
	"github.com/jenkins-x-plugins/jx-secret/pkg/cmd/convert"
	"github.com/jenkins-x-plugins/jx-secret/pkg/extsecrets/secretfacade"
	"github.com/jenkins-x-plugins/jx-secret/pkg/secretmapping"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-helpers/v3/pkg/yamls"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// generateTestOutput enable to regenerate the expected output
var generateTestOutput = false

func TestToExtSecrets(t *testing.T) {
	sourceData := filepath.Join("test_data", "simple")
	fileNames, err := os.ReadDir(sourceData)
	assert.NoError(t, err)

	tmpDir := t.TempDir()

	type testCase struct {
		SourceFile   string
		ResultFile   string
		ExpectedFile string
	}

	var testCases []testCase
	for _, f := range fileNames {
		if !f.IsDir() {
			continue
		}
		name := f.Name()
		if name == ".jx" {
			continue
		}
		srcFile := filepath.Join(sourceData, name, "source.yaml")
		expectedFile := filepath.Join(sourceData, name, "expected.yaml")
		require.FileExists(t, srcFile)
		require.FileExists(t, expectedFile)

		outFile := filepath.Join(tmpDir, name+".yaml")
		err = files.CopyFile(srcFile, outFile)
		require.NoError(t, err, "failed to copy %s to %s", srcFile, outFile)

		testCases = append(testCases, testCase{
			SourceFile:   srcFile,
			ResultFile:   outFile,
			ExpectedFile: expectedFile,
		})
	}

	_, eo := convert.NewCmdSecretConvert()
	eo.Dir = tmpDir
	eo.HelmSecretFolder = filepath.Join(tmpDir, "helm-secrets")
	eo.SourceDir = tmpDir

	eo.SecretMapping, _, err = secretmapping.LoadSecretMapping(sourceData, true)
	require.NoError(t, err, "failed to load secret mapping")

	err = eo.Run()
	require.NoError(t, err, "failed to convert to external secrets in dir %s", tmpDir)

	for _, tc := range testCases {
		resultData, err := os.ReadFile(tc.ResultFile)
		require.NoError(t, err, "failed to load results %s", tc.ResultFile)

		expectData, err := os.ReadFile(tc.ExpectedFile)
		require.NoError(t, err, "failed to load results %s", tc.ExpectedFile)

		result := strings.TrimSpace(string(resultData))
		expectedText := strings.TrimSpace(string(expectData))
		if d := cmp.Diff(expectedText, result); d != "" {
			t.Errorf("generated external secret for file %s did not match expected: %s", tc.SourceFile, d)
		}
		t.Logf("generated for file %s file\n%s\n", tc.SourceFile, result)
	}

	helmSecretsFile := filepath.Join(eo.HelmSecretFolder, "jx", "bucketrepo-config.yaml")
	assert.FileExists(t, helmSecretsFile, "should have generated helm secrets file")
	t.Logf("generated canonical helm secrets file %s\n", helmSecretsFile)
}

func TestToNamespaceSpecificExtSecrets(t *testing.T) {
	sourceData := filepath.Join("test_data", "namespace-vault")
	fileNames, err := os.ReadDir(sourceData)
	assert.NoError(t, err)

	tmpDir := t.TempDir()

	type testCase struct {
		SourceFile   string
		ResultFile   string
		ExpectedFile string
	}

	var testCases []testCase
	for _, f := range fileNames {
		if !f.IsDir() {
			continue
		}
		name := f.Name()
		if name == ".jx" {
			continue
		}
		srcFile := filepath.Join(sourceData, name, "source.yaml")
		expectedFile := filepath.Join(sourceData, name, "expected.yaml")
		require.FileExists(t, srcFile)
		require.FileExists(t, expectedFile)

		outFile := filepath.Join(tmpDir, name+".yaml")
		err = files.CopyFile(srcFile, outFile)
		require.NoError(t, err, "failed to copy %s to %s", srcFile, outFile)

		testCases = append(testCases, testCase{
			SourceFile:   srcFile,
			ResultFile:   outFile,
			ExpectedFile: expectedFile,
		})
	}

	_, eo := convert.NewCmdSecretConvert()
	eo.Dir = tmpDir
	eo.HelmSecretFolder = filepath.Join(tmpDir, "helm-secrets")
	eo.SourceDir = tmpDir

	eo.SecretMapping, _, err = secretmapping.LoadSecretMapping(sourceData, true)
	require.NoError(t, err, "failed to load secret mapping")

	err = eo.Run()
	require.NoError(t, err, "failed to convert to external secrets in dir %s", tmpDir)

	for _, tc := range testCases {
		resultData, err := os.ReadFile(tc.ResultFile)
		require.NoError(t, err, "failed to load results %s", tc.ResultFile)

		expectData, err := os.ReadFile(tc.ExpectedFile)
		require.NoError(t, err, "failed to load results %s", tc.ExpectedFile)

		result := strings.TrimSpace(string(resultData))
		expectedText := strings.TrimSpace(string(expectData))
		if d := cmp.Diff(expectedText, result); d != "" {
			t.Errorf("generated external secret for file %s did not match expected: %s", tc.SourceFile, d)
		}
		t.Logf("generated for file %s file\n%s\n", tc.SourceFile, result)
	}

	helmSecretsFile := filepath.Join(eo.HelmSecretFolder, "jx", "bucketrepo-config.yaml")
	assert.FileExists(t, helmSecretsFile, "should have generated helm secrets file")
	t.Logf("generated canonical helm secrets file %s\n", helmSecretsFile)
}

func TestToUnsecuredSecrets(t *testing.T) {
	sourceData := filepath.Join("test_data", "unsecureSecrets")
	fileNames, err := os.ReadDir(sourceData)
	assert.NoError(t, err)

	tmpDir := t.TempDir()

	type testCase struct {
		SourceFile   string
		ResultFile   string
		ExpectedFile string
	}

	var testCases []testCase
	for _, f := range fileNames {
		if !f.IsDir() {
			continue
		}
		name := f.Name()
		if name == ".jx" {
			continue
		}
		srcFile := filepath.Join(sourceData, name, "source.yaml")
		expectedFile := filepath.Join(sourceData, name, "expected.yaml")
		require.FileExists(t, srcFile)
		require.FileExists(t, expectedFile)
		assert.NoError(t, err)

		outFile := filepath.Join(tmpDir, name+".yaml")
		err = files.CopyFile(srcFile, outFile)
		require.NoError(t, err, "failed to copy %s to %s", srcFile, outFile)

		testCases = append(testCases, testCase{
			SourceFile:   srcFile,
			ResultFile:   outFile,
			ExpectedFile: expectedFile,
		})
	}

	_, eo := convert.NewCmdSecretConvert()
	eo.Dir = tmpDir
	eo.HelmSecretFolder = filepath.Join(tmpDir, "helm-secrets")
	eo.SourceDir = tmpDir
	eo.Filter.Selector = map[string]string{"secret.jenkins-x.io/convert-exclude": "true"}
	eo.Filter.SelectTarget = "metadata.annotations"
	eo.Filter.InvertSelector = true

	eo.SecretMapping, _, err = secretmapping.LoadSecretMapping(sourceData, true)
	require.NoError(t, err, "failed to load secret mapping")

	err = eo.Run()
	require.NoError(t, err, "failed to convert to external secrets in dir %s", tmpDir)

	for _, tc := range testCases {

		require.FileExists(t, tc.ExpectedFile)

		resultData, err := os.ReadFile(tc.ResultFile)
		require.NoError(t, err, "failed to load results %s", tc.ResultFile)

		expectData, err := os.ReadFile(tc.ExpectedFile)
		require.NoError(t, err, "failed to load results %s", tc.ExpectedFile)

		result := strings.TrimSpace(string(resultData))
		expectedText := strings.TrimSpace(string(expectData))
		if d := cmp.Diff(expectedText, result); d != "" {
			t.Errorf("generated secret for file %s did not match expected: %s", tc.SourceFile, d)
		}
		t.Logf("generated secret for file %s file\n%s\n", tc.SourceFile, result)
	}
}

func TestMultipleBackendTypes(t *testing.T) {
	sourceData := filepath.Join("test_data", "backend_types")
	fileNames, err := os.ReadDir(sourceData)
	assert.NoError(t, err)

	tmpDir := t.TempDir()

	type testCase struct {
		SourceFile   string
		ResultFile   string
		ExpectedFile string
	}

	var testCases []testCase
	for _, f := range fileNames {
		if !f.IsDir() {
			continue
		}
		name := f.Name()
		if name == ".jx" {
			continue
		}
		srcFile := filepath.Join(sourceData, name, "source.yaml")
		expectedFile := filepath.Join(sourceData, name, "expected.yaml")
		require.FileExists(t, srcFile)
		require.FileExists(t, expectedFile)

		outFile := filepath.Join(tmpDir, name+".yaml")
		err = files.CopyFile(srcFile, outFile)
		require.NoError(t, err, "failed to copy %s to %s", srcFile, outFile)

		testCases = append(testCases, testCase{
			SourceFile:   srcFile,
			ResultFile:   outFile,
			ExpectedFile: expectedFile,
		})
	}

	_, eo := convert.NewCmdSecretConvert()
	eo.Dir = tmpDir
	eo.HelmSecretFolder = filepath.Join(tmpDir, "helm-secrets")
	eo.SourceDir = tmpDir

	eo.SecretMapping, _, err = secretmapping.LoadSecretMapping(sourceData, true)
	require.NoError(t, err, "failed to load secret mapping")

	assert.Equal(t, v1alpha1.BackendTypeVault, eo.SecretMapping.Spec.Secrets[0].BackendType)
	assert.Equal(t, v1alpha1.BackendTypeGSM, eo.SecretMapping.Spec.Secrets[1].BackendType)
	assert.Equal(t, v1alpha1.BackendTypeAzure, eo.SecretMapping.Spec.Secrets[2].BackendType)

	err = eo.Run()
	require.NoError(t, err, "failed to convert to external secrets in dir %s", tmpDir)

	for _, tc := range testCases {
		resultData, err := os.ReadFile(tc.ResultFile)
		require.NoError(t, err, "failed to load results %s", tc.ResultFile)
		expectData, err := os.ReadFile(tc.ExpectedFile)
		require.NoError(t, err, "failed to load results %s", tc.ExpectedFile)

		result := strings.TrimSpace(string(resultData))
		expectedText := strings.TrimSpace(string(expectData))
		if d := cmp.Diff(expectedText, result); d != "" {
			t.Errorf("Generated external secret for file %s did not match expected: %s", tc.SourceFile, d)
		}
		t.Logf("generated for file %s file\n%s\n", tc.SourceFile, result)
	}
}

func TestAlicloud(t *testing.T) {
	sourceData := filepath.Join("test_data", "alicloud")
	fileNames, err := os.ReadDir(sourceData)
	assert.NoError(t, err)

	tmpDir := t.TempDir()

	type testCase struct {
		SourceFile   string
		ResultFile   string
		ExpectedFile string
	}

	var testCases []testCase
	for _, f := range fileNames {
		if !f.IsDir() {
			continue
		}
		name := f.Name()
		if name == ".jx" {
			continue
		}
		srcFile := filepath.Join(sourceData, name, "source.yaml")
		expectedFile := filepath.Join(sourceData, name, "expected.yaml")
		require.FileExists(t, srcFile)
		require.FileExists(t, expectedFile)

		outFile := filepath.Join(tmpDir, name+".yaml")
		err = files.CopyFile(srcFile, outFile)
		require.NoError(t, err, "failed to copy %s to %s", srcFile, outFile)

		testCases = append(testCases, testCase{
			SourceFile:   srcFile,
			ResultFile:   outFile,
			ExpectedFile: expectedFile,
		})
	}

	_, eo := convert.NewCmdSecretConvert()
	eo.Dir = tmpDir
	eo.HelmSecretFolder = filepath.Join(tmpDir, "helm-secrets")
	eo.SourceDir = tmpDir

	eo.SecretMapping, _, err = secretmapping.LoadSecretMapping(sourceData, true)
	require.NoError(t, err, "failed to load secret mapping")

	assert.Equal(t, v1alpha1.BackendTypeAlicloud, eo.SecretMapping.Spec.Secrets[0].BackendType)
	assert.Equal(t, v1alpha1.BackendTypeAlicloud, eo.SecretMapping.Spec.Secrets[1].BackendType)
	assert.Equal(t, v1alpha1.BackendTypeAlicloud, eo.SecretMapping.Spec.Secrets[2].BackendType)

	err = eo.Run()
	require.NoError(t, err, "failed to convert to external secrets in dir %s", tmpDir)

	for _, tc := range testCases {
		if generateTestOutput {
			generatedFile := tc.ResultFile
			expectedPath := tc.ExpectedFile
			data, err := os.ReadFile(generatedFile)
			require.NoError(t, err, "failed to load %s", generatedFile)

			err = os.WriteFile(expectedPath, data, 0o600)
			require.NoError(t, err, "failed to save file %s", expectedPath)

			t.Logf("saved file %s\n", expectedPath)
			continue
		}

		resultData, err := os.ReadFile(tc.ResultFile)
		require.NoError(t, err, "failed to load results %s", tc.ResultFile)
		expectData, err := os.ReadFile(tc.ExpectedFile)
		require.NoError(t, err, "failed to load results %s", tc.ExpectedFile)

		result := strings.TrimSpace(string(resultData))
		expectedText := strings.TrimSpace(string(expectData))

		if d := cmp.Diff(expectedText, result); d != "" {
			t.Errorf("Generated external secret for file %s did not match expected: %s", tc.SourceFile, d)
		}
		t.Logf("generated for file %s file\n%s\n", tc.SourceFile, result)
	}
}

func TestAWSSecretsManager(t *testing.T) {
	sourceData := filepath.Join("test_data", "awsSecretsManager")
	fileNames, err := os.ReadDir(sourceData)
	assert.NoError(t, err)

	tmpDir := t.TempDir()

	type testCase struct {
		SourceFile   string
		ResultFile   string
		ExpectedFile string
	}

	var testCases []testCase
	for _, f := range fileNames {
		if !f.IsDir() {
			continue
		}
		name := f.Name()
		if name == ".jx" {
			continue
		}
		srcFile := filepath.Join(sourceData, name, "source.yaml")
		expectedFile := filepath.Join(sourceData, name, "expected.yaml")
		require.FileExists(t, srcFile)
		require.FileExists(t, expectedFile)

		outFile := filepath.Join(tmpDir, name+".yaml")
		err = files.CopyFile(srcFile, outFile)
		require.NoError(t, err, "failed to copy %s to %s", srcFile, outFile)

		testCases = append(testCases, testCase{
			SourceFile:   srcFile,
			ResultFile:   outFile,
			ExpectedFile: expectedFile,
		})
	}

	_, eo := convert.NewCmdSecretConvert()
	eo.Dir = tmpDir
	eo.HelmSecretFolder = filepath.Join(tmpDir, "helm-secrets")
	eo.SourceDir = tmpDir

	eo.SecretMapping, _, err = secretmapping.LoadSecretMapping(sourceData, true)
	require.NoError(t, err, "failed to load secret mapping")
	assert.Equal(t, v1alpha1.BackendTypeAWSSecretsManager, eo.SecretMapping.Spec.Secrets[0].BackendType)
	assert.Equal(t, v1alpha1.BackendTypeAWSSecretsManager, eo.SecretMapping.Spec.Secrets[1].BackendType)
	assert.Equal(t, v1alpha1.BackendTypeAWSSecretsManager, eo.SecretMapping.Spec.Secrets[2].BackendType)

	err = eo.Run()
	require.NoError(t, err, "failed to convert to external secrets in dir %s", tmpDir)

	for _, tc := range testCases {
		if generateTestOutput {
			generatedFile := tc.ResultFile
			expectedPath := tc.ExpectedFile
			data, err := os.ReadFile(generatedFile)
			require.NoError(t, err, "failed to load %s", generatedFile)

			err = os.WriteFile(expectedPath, data, 0o600)
			require.NoError(t, err, "failed to save file %s", expectedPath)

			t.Logf("saved file %s\n", expectedPath)
			continue
		}

		resultData, err := os.ReadFile(tc.ResultFile)
		require.NoError(t, err, "failed to load results %s", tc.ResultFile)
		expectData, err := os.ReadFile(tc.ExpectedFile)
		require.NoError(t, err, "failed to load results %s", tc.ExpectedFile)

		result := strings.TrimSpace(string(resultData))
		expectedText := strings.TrimSpace(string(expectData))

		if d := cmp.Diff(expectedText, result); d != "" {
			t.Errorf("Generated external secret for file %s did not match expected: %s", tc.SourceFile, d)
		}
		t.Logf("generated for file %s file\n%s\n", tc.SourceFile, result)
	}
}

func TestAWSParameterStore(t *testing.T) {
	sourceData := filepath.Join("test_data", "awsParameterStore")
	fileNames, err := os.ReadDir(sourceData)
	assert.NoError(t, err)

	tmpDir := t.TempDir()

	type testCase struct {
		SourceFile   string
		ResultFile   string
		ExpectedFile string
	}

	var testCases []testCase
	for _, f := range fileNames {
		if !f.IsDir() {
			continue
		}
		name := f.Name()
		if name == ".jx" {
			continue
		}
		srcFile := filepath.Join(sourceData, name, "source.yaml")
		expectedFile := filepath.Join(sourceData, name, "expected.yaml")
		require.FileExists(t, srcFile)
		require.FileExists(t, expectedFile)

		outFile := filepath.Join(tmpDir, name+".yaml")
		err = files.CopyFile(srcFile, outFile)
		require.NoError(t, err, "failed to copy %s to %s", srcFile, outFile)

		testCases = append(testCases, testCase{
			SourceFile:   srcFile,
			ResultFile:   outFile,
			ExpectedFile: expectedFile,
		})
	}

	_, eo := convert.NewCmdSecretConvert()
	eo.Dir = tmpDir
	eo.HelmSecretFolder = filepath.Join(tmpDir, "helm-secrets")
	eo.SourceDir = tmpDir

	eo.SecretMapping, _, err = secretmapping.LoadSecretMapping(sourceData, true)
	require.NoError(t, err, "failed to load secret mapping")

	assert.Equal(t, v1alpha1.BackendTypeAWSParameterStore, eo.SecretMapping.Spec.Secrets[0].BackendType)
	assert.Equal(t, v1alpha1.BackendTypeAWSParameterStore, eo.SecretMapping.Spec.Secrets[1].BackendType)
	assert.Equal(t, v1alpha1.BackendTypeAWSParameterStore, eo.SecretMapping.Spec.Secrets[2].BackendType)

	err = eo.Run()
	require.NoError(t, err, "failed to convert to external secrets in dir %s", tmpDir)

	for _, tc := range testCases {
		if generateTestOutput {
			generatedFile := tc.ResultFile
			expectedPath := tc.ExpectedFile
			data, err := os.ReadFile(generatedFile)
			require.NoError(t, err, "failed to load %s", generatedFile)

			err = os.WriteFile(expectedPath, data, 0o600)
			require.NoError(t, err, "failed to save file %s", expectedPath)

			t.Logf("saved file %s\n", expectedPath)
			continue
		}

		resultData, err := os.ReadFile(tc.ResultFile)
		require.NoError(t, err, "failed to load results %s", tc.ResultFile)
		expectData, err := os.ReadFile(tc.ExpectedFile)
		require.NoError(t, err, "failed to load results %s", tc.ExpectedFile)

		result := strings.TrimSpace(string(resultData))
		expectedText := strings.TrimSpace(string(expectData))

		if d := cmp.Diff(expectedText, result); d != "" {
			t.Errorf("Generated external secret for file %s did not match expected: %s", tc.SourceFile, d)
		}
		t.Logf("generated for file %s file\n%s\n", tc.SourceFile, result)
	}
}

func TestIBMSecretsManager(t *testing.T) {
	sourceData := filepath.Join("test_data", "ibmSecretsManager")
	fileNames, err := os.ReadDir(sourceData)
	assert.NoError(t, err)

	tmpDir := t.TempDir()

	type testCase struct {
		SourceFile   string
		ResultFile   string
		ExpectedFile string
	}

	var testCases []testCase
	for _, f := range fileNames {
		if !f.IsDir() {
			continue
		}
		name := f.Name()
		if name == ".jx" {
			continue
		}
		srcFile := filepath.Join(sourceData, name, "source.yaml")
		expectedFile := filepath.Join(sourceData, name, "expected.yaml")
		require.FileExists(t, srcFile)
		require.FileExists(t, expectedFile)

		outFile := filepath.Join(tmpDir, name+".yaml")
		err = files.CopyFile(srcFile, outFile)
		require.NoError(t, err, "failed to copy %s to %s", srcFile, outFile)

		testCases = append(testCases, testCase{
			SourceFile:   srcFile,
			ResultFile:   outFile,
			ExpectedFile: expectedFile,
		})
	}

	_, eo := convert.NewCmdSecretConvert()
	eo.Dir = tmpDir
	eo.HelmSecretFolder = filepath.Join(tmpDir, "helm-secrets")
	eo.SourceDir = tmpDir

	eo.SecretMapping, _, err = secretmapping.LoadSecretMapping(sourceData, true)
	require.NoError(t, err, "failed to load secret mapping")

	assert.Equal(t, v1alpha1.BackendTypeIBMSecretsManager, eo.SecretMapping.Spec.Secrets[0].BackendType)
	assert.Equal(t, v1alpha1.BackendTypeIBMSecretsManager, eo.SecretMapping.Spec.Secrets[1].BackendType)
	assert.Equal(t, v1alpha1.BackendTypeIBMSecretsManager, eo.SecretMapping.Spec.Secrets[2].BackendType)

	err = eo.Run()
	require.NoError(t, err, "failed to convert to external secrets in dir %s", tmpDir)

	for _, tc := range testCases {
		if generateTestOutput {
			generatedFile := tc.ResultFile
			expectedPath := tc.ExpectedFile
			data, err := os.ReadFile(generatedFile)
			require.NoError(t, err, "failed to load %s", generatedFile)

			err = os.WriteFile(expectedPath, data, 0o600)
			require.NoError(t, err, "failed to save file %s", expectedPath)

			t.Logf("saved file %s\n", expectedPath)
			continue
		}

		resultData, err := os.ReadFile(tc.ResultFile)
		require.NoError(t, err, "failed to load results %s", tc.ResultFile)
		expectData, err := os.ReadFile(tc.ExpectedFile)
		require.NoError(t, err, "failed to load results %s", tc.ExpectedFile)

		result := strings.TrimSpace(string(resultData))
		expectedText := strings.TrimSpace(string(expectData))

		if d := cmp.Diff(expectedText, result); d != "" {
			t.Errorf("Generated external secret for file %s did not match expected: %s", tc.SourceFile, d)
		}
		t.Logf("generated for file %s file\n%s\n", tc.SourceFile, result)
	}
}

func TestAzureKeyVault(t *testing.T) {
	sourceData := filepath.Join("test_data", "azure_key_vault")
	fileNames, err := os.ReadDir(sourceData)
	assert.NoError(t, err)

	tmpDir := t.TempDir()

	type testCase struct {
		SourceFile   string
		ResultFile   string
		ExpectedFile string
	}

	var testCases []testCase
	for _, f := range fileNames {
		if !f.IsDir() {
			continue
		}
		name := f.Name()
		if name == ".jx" {
			continue
		}
		srcFile := filepath.Join(sourceData, name, "source.yaml")
		expectedFile := filepath.Join(sourceData, name, "expected.yaml")
		require.FileExists(t, srcFile)
		require.FileExists(t, expectedFile)

		outFile := filepath.Join(tmpDir, name+".yaml")
		err = files.CopyFile(srcFile, outFile)
		require.NoError(t, err, "failed to copy %s to %s", srcFile, outFile)

		testCases = append(testCases, testCase{
			SourceFile:   srcFile,
			ResultFile:   outFile,
			ExpectedFile: expectedFile,
		})
	}

	_, eo := convert.NewCmdSecretConvert()
	eo.Dir = tmpDir
	eo.HelmSecretFolder = filepath.Join(tmpDir, "helm-secrets")
	eo.SourceDir = tmpDir

	eo.SecretMapping, _, err = secretmapping.LoadSecretMapping(sourceData, true)
	require.NoError(t, err, "failed to load secret mapping")

	assert.Equal(t, v1alpha1.BackendTypeAzure, eo.SecretMapping.Spec.Secrets[0].BackendType)
	assert.Equal(t, v1alpha1.BackendTypeAzure, eo.SecretMapping.Spec.Secrets[1].BackendType)
	assert.Equal(t, v1alpha1.BackendTypeAzure, eo.SecretMapping.Spec.Secrets[2].BackendType)

	err = eo.Run()
	require.NoError(t, err, "failed to convert to external secrets in dir %s", tmpDir)

	for _, tc := range testCases {
		resultData, err := os.ReadFile(tc.ResultFile)
		require.NoError(t, err, "failed to load results %s", tc.ResultFile)
		expectData, err := os.ReadFile(tc.ExpectedFile)
		require.NoError(t, err, "failed to load results %s", tc.ExpectedFile)

		result := strings.TrimSpace(string(resultData))
		expectedText := strings.TrimSpace(string(expectData))
		if d := cmp.Diff(expectedText, result); d != "" {
			t.Errorf("Generated external secret for file %s did not match expected: %s", tc.SourceFile, d)
		}
		t.Logf("generated for file %s file\n%s\n", tc.SourceFile, result)
	}
}

func TestGCPProjectIDValidation(t *testing.T) {
	_, _, err := secretmapping.LoadSecretMapping(filepath.Join("test_data", "validation"), true)
	require.Error(t, err, "failed to get validation error")
	log.Logger().Infof("%s", err.Error())
	assert.True(t, strings.Contains(err.Error(), "Spec.Defaults.BackendType: zero value"), "failed to get correct validation error")
}

func TestConvertAndSchemaEnrich(t *testing.T) {
	sourceData := filepath.Join("test_data", "schema")
	require.DirExists(t, sourceData)

	tmpDir := t.TempDir()

	err := files.CopyDir(sourceData, tmpDir, true)
	require.NoError(t, err, "failed to copy %s to %s", sourceData, tmpDir)

	_, eo := convert.NewCmdSecretConvert()
	eo.Dir = tmpDir
	eo.HelmSecretFolder = filepath.Join(tmpDir, "helm-secrets")

	err = eo.Run()
	require.NoError(t, err, "failed to convert to external secrets in dir %s", tmpDir)

	t.Logf("converted the Secrets to ExternalSecrets in dir %s", eo.Dir)

	// lets verify that we did not convert the empty tekton webhook certs secret
	webhookSecretFile := filepath.Join(tmpDir, "config-root", "namespaces", "jx", "tekton", "webhook-certs-secret.yaml")
	require.FileExists(t, webhookSecretFile, "should have empty tekton secret file")
	webhookSecret := &unstructured.Unstructured{}
	err = yamls.LoadFile(webhookSecretFile, webhookSecret)
	require.NoError(t, err, "failed to parse tekton webhook secret %s", webhookSecretFile)
	assert.Equal(t, "Secret", webhookSecret.GetKind(), "kind of resource in file %s", webhookSecretFile)

	// now lets verify a number of schema properties
	testCases := []struct {
		path       string
		properties []string
	}{
		{
			path:       filepath.Join("chartmuseum", "secret.yaml"),
			properties: []string{"username", "password"},
		},
		{
			path:       filepath.Join("tekton", "250-docker-secret.yaml"),
			properties: []string{"username", "password"},
		},
		{
			path:       filepath.Join("tekton", "250-git-secret.yaml"),
			properties: []string{"username", "token"},
		},
		{
			path:       filepath.Join("jxboot-helmfile-resources", "maven-settings-secret.yaml"),
			properties: []string{"settingsXml", "securityXml"},
		},
	}

	for _, tc := range testCases {
		file := filepath.Join(eo.SourceDir, "namespaces", "jx", tc.path)
		require.FileExists(t, file)

		// lets load it and assert its got the schema
		es := v1.ExternalSecret{}
		err := yamls.LoadFile(file, &es)
		require.NoError(t, err, "failed to load ExternalSecret %s", file)

		sp := &secretfacade.SecretPair{
			ExternalSecret: es,
		}
		object, err := sp.SchemaObject()
		require.NoError(t, err, "failed to not find Object schema for file %s", file)
		require.NotNil(t, object, "no Object schema for file %s", file)

		for _, propertyName := range tc.properties {
			property := object.FindProperty(propertyName)
			assert.NotNil(t, property, "could not find property %s for object schema on file %s", propertyName, file)
		}
		t.Logf("ExternalSecret %s has object schema annotation with properties %#v\n", es.Name, tc.properties)
	}
}

func TestConvertAndSchemaEnrichWithLocalSchemas(t *testing.T) {
	sourceData := filepath.Join("test_data", "local-schema")
	require.DirExists(t, sourceData)

	tmpDir := t.TempDir()

	err := files.CopyDir(sourceData, tmpDir, true)
	require.NoError(t, err, "failed to copy %s to %s", sourceData, tmpDir)

	_, eo := convert.NewCmdSecretConvert()
	eo.VersionStreamDir = filepath.Join(tmpDir, "versionStream")
	eo.Dir = tmpDir
	eo.HelmSecretFolder = filepath.Join(tmpDir, "helm-secrets")

	err = eo.Run()
	require.NoError(t, err, "failed to convert to external secrets in dir %s", tmpDir)

	t.Logf("converted the Secrets to ExternalSecrets in dir %s", eo.Dir)

	// now lets verify a number of schema properties
	testCases := []struct {
		path       string
		properties []string
	}{
		{
			path:       filepath.Join("chartmuseum", "secret.yaml"),
			properties: []string{"username", "password"},
		},
	}

	for _, tc := range testCases {
		file := filepath.Join(eo.SourceDir, "namespaces", "jx", tc.path)
		require.FileExists(t, file)

		// lets load it and assert its got the schema
		es := v1.ExternalSecret{}
		err := yamls.LoadFile(file, &es)
		require.NoError(t, err, "failed to load ExternalSecret %s", file)

		sp := &secretfacade.SecretPair{
			ExternalSecret: es,
		}
		object, err := sp.SchemaObject()
		require.NoError(t, err, "failed to not find Object schema for file %s", file)
		require.NotNil(t, object, "no Object schema for file %s", file)

		for _, propertyName := range tc.properties {
			property := object.FindProperty(propertyName)
			assert.NotNil(t, property, "could not find property %s for object schema on file %s", propertyName, file)
		}
		t.Logf("ExternalSecret %s has object schema annotation with properties %#v\n", es.Name, tc.properties)
	}
}
