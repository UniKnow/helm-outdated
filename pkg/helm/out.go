/*******************************************************************************
*
* Copyright 2019 SAP SE
*
* Licensed under the Apache License, Version 2.0 (the "License");
* you may not use this file except in compliance with the License.
* You should have received a copy of the License along with this
* program. If not, you may obtain a copy of the License at
*
*     http://www.apache.org/licenses/LICENSE-2.0
*
* Unless required by applicable law or agreed to in writing, software
* distributed under the License is distributed on an "AS IS" BASIS,
* WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
* See the License for the specific language governing permissions and
* limitations under the License.
*
*******************************************************************************/

package helm

import (
	"bytes"
	"encoding/json"
	"os"
	"path"
	"path/filepath"

	"gopkg.in/yaml.v3"

    log "github.com/sirupsen/logrus"

	"helm.sh/helm/v3/pkg/chart"
)

// Requirements is a list of requirements for a chart.
//
// Requirements are charts upon which this chart depends. This expresses
// developer intent.
type Requirements struct {
	Dependencies []*chart.Dependency `json:"dependencies,flow,inline"`
	test int
}

func toYamlWithIndent(in interface{}, indent int) ([]byte, error) {
	// Unfortunately chartutil.Requirements, charts.Chart structs only have the JSON anchors, but not the YAML ones.
	// So we have to take the JSON detour.
	log.Debug("Converting updated requirements into yaml")
	jsonData, err := json.Marshal(in)
	if err != nil {
		return nil, err
	}

	var jsonObj interface{}
	if err := yaml.Unmarshal(jsonData, &jsonObj); err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	defer enc.Close()

	enc.SetIndent(indent)
	err = enc.Encode(jsonObj)
	return buf.Bytes(), err
}

func writeRequirements(chartPath string, reqs []*chart.Dependency, indent int) error {


	data, err := toYamlWithIndent(&Requirements{Dependencies:   reqs,}, indent)
	if err != nil {
		return err
	}

	absPath, err := filepath.Abs(path.Join(chartPath, requirementsName))
	if err != nil {
		return err
	}

	f, err := os.OpenFile(absPath, os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	if err := f.Truncate(0); err != nil {
		return err
	}

	_, err = f.Write(data)
	return err
}

func writeChartMetadata(chartPath string, c *chart.Metadata) error {
	data, err := toYamlWithIndent(c, 0)
	if err != nil {
		return err
	}

	absPath, err := filepath.Abs(path.Join(chartPath, chartMetadataName))
	if err != nil {
		return err
	}

	f, err := os.OpenFile(absPath, os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteAt(data, 0)
	return err
}
