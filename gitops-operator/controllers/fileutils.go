package controllers

import (
	"fmt"
	"github.com/spf13/afero"
	"gopkg.in/yaml.v3"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
)

/* MIT License
 *
 * Copyright (c) 2017 Roland Singer [roland.singer@desertbit.com]
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

// CopyFile copies the contents of the file named src to the file named
// by dst. The file will be created if it does not already exist. If the
// destination file exists, all it's contents will be replaced by the contents
// of the source file. The file mode will be copied from the source and
// the copied data is synced/flushed to stable storage.
func CopyFile(fs afero.Fs, src, dst string) (err error) {

	in, err := fs.Open(src)
	if err != nil {
		return
	}
	defer in.Close()

	out, err := fs.Create(dst)
	if err != nil {
		return
	}
	defer func() {
		if e := out.Close(); e != nil {
			err = e
		}
	}()

	_, err = io.Copy(out, in)
	if err != nil {
		return
	}

	err = out.Sync()
	if err != nil {
		return
	}

	si, err := fs.Stat(src)
	if err != nil {
		return
	}
	err = fs.Chmod(dst, si.Mode())
	if err != nil {
		return
	}

	return
}

// CopyDir recursively copies a directory tree, attempting to preserve permissions.
// Source directory must exist, destination directory must *not* exist.
// Symlinks are ignored and skipped.
func CopyDir(fs afero.Fs, src string, dst string) (err error) {

	src = filepath.Clean(src)
	dst = filepath.Clean(dst)

	si, err := fs.Stat(src)
	if err != nil {
		return err
	}
	if !si.IsDir() {
		return fmt.Errorf("source is not a directory")
	}

	_, err = fs.Stat(dst)
	if err != nil && !os.IsNotExist(err) {
		return
	}
	if err == nil {
		return fmt.Errorf("destination already exists")
	}

	err = fs.MkdirAll(dst, si.Mode())
	if err != nil {
		return
	}

	entries, err := afero.ReadDir(fs, src)
	if err != nil {
		return
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			err = CopyDir(fs, srcPath, dstPath)
			if err != nil {
				return
			}
		} else {
			// Skip symlinks.
			if entry.Mode()&os.ModeSymlink != 0 {
				continue
			}

			err = CopyFile(fs, srcPath, dstPath)
			if err != nil {
				return
			}
		}
	}

	return
}

func RemoveDir(fs afero.Fs, dir string) (err error) {
	dir = filepath.Clean(dir)
	si, err := fs.Stat(dir)
	if err != nil {
		return err
	}
	if !si.IsDir() {
		return fmt.Errorf("dir is not a directory")
	}

	err = fs.RemoveAll(dir)
	if err != nil {
		return err
	}

	return
}

func cleanupServiceDirs(fs afero.Fs, servicename string, directory string, stages []DirectoryData) error {
	err := fs.RemoveAll(filepath.Join(directory, "base", servicename))
	if err != nil {
		return fmt.Errorf("could not remove directory: %w", err)
	}

	for _, stage := range stages {
		err := fs.RemoveAll(filepath.Join(directory, "stages", stage.DirectoryName, servicename))
		if err != nil {
			return fmt.Errorf("could not remove directory: %w", err)
		}
	}
	return nil
}

func findServiceDirs(fs afero.Fs, basedir string, filePattern string) (map[DirectoryData]KeptnArtifactMetadataSpec, error) {
	metadata := map[DirectoryData]KeptnArtifactMetadataSpec{}
	dirs, err := afero.ReadDir(fs, basedir)
	if err != nil {
		return nil, fmt.Errorf("could not read directories: %w", err)
	}

	for _, dir := range dirs {
		if dir.IsDir() {
			_, err := fs.Stat(filepath.Join(basedir, dir.Name(), filePattern))
			if err == nil {
				fmt.Println(dir.Name())
				yamlFile, err := ioutil.ReadFile(filepath.Join(basedir, dir.Name(), filePattern))
				if err != nil {
					return nil, fmt.Errorf("could not read file: %w", err)
				}
				metadataYaml := KeptnArtifactMetadata{}
				err = yaml.Unmarshal(yamlFile, &metadataYaml)
				if err != nil {
					return nil, fmt.Errorf("could not unmarshal file: %w", err)
				}
				metadata[DirectoryData{DirectoryName: dir.Name(), Path: filepath.Join(basedir, dir.Name())}] = metadataYaml.Spec
			}
		}
	}
	return metadata, nil
}

func findDirs(fs afero.Fs, basedir string) ([]DirectoryData, error) {
	var directories []DirectoryData
	dirs, err := afero.ReadDir(fs, basedir)
	if err != nil {
		return nil, fmt.Errorf("could not read directories: %w", err)
	}

	for _, dir := range dirs {
		if dir.IsDir() {
			directories = append(directories, DirectoryData{DirectoryName: dir.Name(), Path: filepath.Join(basedir, dir.Name())})
		}
	}
	return directories, nil
}
