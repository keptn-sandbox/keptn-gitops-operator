package utils

import (
	"github.com/spf13/afero"
	testify_assert "github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"gotest.tools/assert"
	"io/ioutil"
	"log"
	"path/filepath"
	"testing"
)

func TestPerformFileMove(t *testing.T) {

	fs := afero.NewMemMapFs()
	setupTestFsBase(t, fs)
	setupTestFsStage(t, fs)

	err := performFileMove(fs, "service", "dev", "source", "dest")
	assert.NilError(t, err)

	assertFilesExist(t, fs)
}

func TestPerformFileMove_NoBaseDir(t *testing.T) {

	fs := afero.NewMemMapFs()
	setupTestFsStage(t, fs)

	err := performFileMove(fs, "service", "dev", "source", "dest")
	assert.NilError(t, err)

	resultDir := filepath.Join("dest", "service", "helm")
	assertDirExists(t, fs, resultDir)
	assertFileExists(t, fs, filepath.Join(resultDir, "values.yaml"))
}

func TestPerformFileMove_NoStageDir(t *testing.T) {

	fs := afero.NewMemMapFs()
	setupTestFsBase(t, fs)

	err := performFileMove(fs, "service", "dev", "source", "dest")
	assert.NilError(t, err)

	resultDir := filepath.Join("dest", "service", "helm")
	assertDirExists(t, fs, resultDir)
	assertFileExists(t, fs, filepath.Join(resultDir, "Chart.yaml"))
}

func setupTestFsBase(t *testing.T, fs afero.Fs) {

	serviceSourceDir := filepath.Join("source", "base", "service", "helm")

	err := fs.MkdirAll(serviceSourceDir, 0700)
	assert.NilError(t, err)

	_, err = fs.Create(filepath.Join(serviceSourceDir, "Chart.yaml"))
	assert.NilError(t, err)
}

func setupTestFsStage(t *testing.T, fs afero.Fs) {

	serviceSourceStageDir := filepath.Join("source", "stages", "dev", "service", "helm")

	err := fs.MkdirAll(serviceSourceStageDir, 0700)
	assert.NilError(t, err)

	_, err = fs.Create(filepath.Join(serviceSourceStageDir, "values.yaml"))
	assert.NilError(t, err)
}

func assertFilesExist(t *testing.T, fs afero.Fs) {

	resultDir := filepath.Join("dest", "service", "helm")

	assertDirExists(t, fs, resultDir)
	assertFileExists(t, fs, filepath.Join(resultDir, "Chart.yaml"))
	assertFileExists(t, fs, filepath.Join(resultDir, "values.yaml"))
}

func assertDirExists(t *testing.T, fs afero.Fs, directory string) {
	dir, err := afero.ReadDir(fs, directory)
	assert.NilError(t, err)
	assert.Check(t, dir != nil)
}

func assertFileExists(t *testing.T, fs afero.Fs, file string) {
	dir, err := afero.ReadFile(fs, file)
	assert.NilError(t, err)
	assert.Check(t, dir != nil)
}

func Test_mergeMaps(t *testing.T) {
	t.Run("merge test", func(t *testing.T) {

		err, inputChart := MergeValues("testdata/values_input.yaml", "testdata/values_stage.yaml")
		if err != nil {
			t.Errorf("Could not merge values")
		}

		expectedValuesFile, err := ioutil.ReadFile("testdata/values_expected.yaml")
		if err != nil {
			log.Fatalf("Could not read file")
		}

		expectedValues := map[string]interface{}{}
		if err := yaml.Unmarshal(expectedValuesFile, &expectedValues); err != nil {
			log.Fatalf("Unmarshalling error")
		}

		testify_assert.Equal(t, expectedValues, inputChart)
	})

	t.Run("wrong stage values", func(t *testing.T) {

		err, inputChart := MergeValues("testdata/values_input.yaml", "testdata/values_stagex.yaml")
		if err != nil {
			t.Errorf("Could not merge values")
		}

		expectedValuesFile, err := ioutil.ReadFile("testdata/values_input.yaml")
		if err != nil {
			log.Fatalf("Could not read file")
		}

		expectedValues := map[string]interface{}{}
		if err := yaml.Unmarshal(expectedValuesFile, &expectedValues); err != nil {
			log.Fatalf("Unmarshalling error")
		}

		testify_assert.Equal(t, expectedValues, inputChart)
	})

	t.Run("wrong base values", func(t *testing.T) {

		err, inputChart := MergeValues("testdata/values_inputx.yaml", "testdata/values_stage.yaml")
		if err != nil {
			t.Errorf("Could not merge values")
		}

		expectedValuesFile, err := ioutil.ReadFile("testdata/values_stage.yaml")
		if err != nil {
			log.Fatalf("Could not read file")
		}

		expectedValues := map[string]interface{}{}
		if err := yaml.Unmarshal(expectedValuesFile, &expectedValues); err != nil {
			log.Fatalf("Unmarshalling error")
		}

		testify_assert.Equal(t, expectedValues, inputChart)
	})

	t.Run("no values", func(t *testing.T) {
		err, _ := MergeValues("testdata/values_inputx.yaml", "testdata/values_stagex.yaml")
		if err == nil {
			t.Errorf("Did not threw an error and no values were present")
		}

	})

}
