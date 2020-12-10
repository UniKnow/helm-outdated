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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/helm/pkg/chartutil"
	"os"
	"path"
	"testing"
)

func newRequirements() *chartutil.Requirements {
	return &chartutil.Requirements{
		Dependencies: []*chartutil.Dependency{
			{
				Name:       "testdependency",
				Version:    "v0.0.1",
				Repository: "https://repo.evil.corp",
			},
			{
				Name:       "testdepdendency1",
				Version:    "v0.0.2",
				Repository: "https://repo.evil.corp",
			},
		},
	}
}

func ensureEmptyFileExists(pathParts ...string) error {
	filepath := path.Join(pathParts...)
	if f, err := os.Create(filepath); err != nil {
		if os.IsExist(err) {
			return f.Truncate(0)
		}
		return err
	}
	return nil
}

func TestWriteRequirements(t *testing.T) {
	dir, err := os.Getwd()
	require.NoError(t, err, "there must be no error getting the current path")
	chartPath := path.Join(dir, "fixtures")
	require.NoError(t, ensureEmptyFileExists(chartPath, requirementsName), "there must be no error creating the requirements.yaml")

	err = writeRequirements(chartPath, newRequirements(), 4)
	assert.NoError(t, err, "there should be no error writing the chart requirements")
}

func TestIncrementChartVersion(t *testing.T) {
	dir, err := os.Getwd()
	require.NoError(t, err, "there must be no error getting the current path")
	chartPath := path.Join(dir, "fixtures")

	err = IncrementChartVersion(chartPath, IncTypes.Patch)
	assert.NoError(t, err, "there should be no error incrementing the chart version and writing the new Chart.yaml")
}
