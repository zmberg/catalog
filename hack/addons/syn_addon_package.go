/*
Copyright 2021 The KubeVela Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"bytes"
	"fmt"
	"net/http"
	"path/filepath"
	"time"

	"sigs.k8s.io/yaml"

	"io/ioutil"
	"os"
	"os/exec"

	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/repo"
)

type Metadata struct {
	Name          string   `json:"name" validate:"required"`
	Version       string   `json:"version"`
	Description   string   `json:"description"`
	Icon          string   `json:"icon"`
	URL           string   `json:"url,omitempty"`
	Tags          []string `json:"tags,omitempty"`
	NeedNamespace []string `json:"needNamespace,omitempty"`
	Invisible     bool     `json:"invisible"`
}

func main() {
	dir := os.Args[1]
	f, err := ioutil.ReadDir(dir)
	if err != nil {
		fmt.Println(err)
		return
	}
	repoURL := os.Args[2]

	if len(repoURL) == 0 {
		fmt.Println("Please set repoURL")
		return
	}

	originIndex := &repo.IndexFile{}

	body, err := http.Get(repoURL + "/index.yaml")
	if err != nil {
		fmt.Println(err)
		return
	}
	if body.StatusCode != 404 {
		indexByte, err := ioutil.ReadAll(body.Body)
		if err != nil {
			fmt.Println(err)
			return
		}
		if len(indexByte) != 0 {
			err := yaml.UnmarshalStrict(indexByte, originIndex)
			if err != nil {
				fmt.Println(err)
			}
		}
	}

	entries := map[string]repo.ChartVersions{}
	for _, info := range f {
		if info.IsDir() {
			f, err := ioutil.ReadFile(filepath.Join(dir, info.Name(), "metadata.yaml"))
			if err != nil {
				fmt.Println(err)
				continue
			}
			m := Metadata{}
			err = yaml.Unmarshal(f, &m)
			if err != nil {
				fmt.Println(err)
				return
			}
			entry := repo.ChartVersions{}
			entry = append(entry, &repo.ChartVersion{Metadata: &chart.Metadata{Name: info.Name(),
				Version: m.Version, Icon: m.Icon, Keywords: m.Tags, Description: m.Description,
				Home: m.URL}, Created: time.Now(), URLs: []string{repoURL + "/" + info.Name() + "-" + m.Version + ".tgz"}})
			entries[info.Name()] = entry

			err = helmSave(dir, info.Name(), info.Name(), m.Version)
			if err != nil {
				fmt.Println(err)
			}
		}
	}
	index := repo.IndexFile{APIVersion: "v1", Entries: entries}

	index.Merge(originIndex)
	out, err := yaml.Marshal(index)
	if err != nil {
		fmt.Println(err)
		return
	}
	err = ioutil.WriteFile(dir+"/index.yaml", out, os.ModePerm)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("handle over")
}

func helmSave(dir, name, addonDir, version string) error {
	filename := fmt.Sprintf("%s%s-%s.tgz", dir, name, version)
	var outInfo bytes.Buffer
	cmd := exec.Command("tar", "zcf", filename, dir+addonDir+"/")
	cmd.Stdout = &outInfo
	fmt.Println(cmd.String())
	if err := cmd.Run(); err != nil {
		fmt.Println(outInfo.String())
		fmt.Println(err)
		return err
	}
	fmt.Printf("addon package %s \n", filename)
	return nil
}
