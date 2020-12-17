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
    "helm.sh/helm/v3/pkg/chart"
)

// Filter for dependencies.
type Filter struct {
	Repositories,
	DependencyNames []string
}

// FilterDependencies ...
func (f *Filter) FilterDependencies(dependencies []*chart.Dependency) []*chart.Dependency {
	var filteredDeps []*chart.Dependency
	for _, dep := range dependencies {
		keep := true

		// Filter by repositories.
		if f.Repositories != nil && len(f.Repositories) > 0 && !stringSliceContains(f.Repositories, dep.Repository) {
			keep = false
		}

		// Filter by dependency name.
		if f.DependencyNames != nil && len(f.DependencyNames) > 0 && !stringSliceContains(f.DependencyNames, dep.Name) {
			keep = false
		}

		if keep {
			filteredDeps = append(filteredDeps, dep)
		}
	}

	return filteredDeps
}
