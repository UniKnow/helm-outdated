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
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/Masterminds/semver"

    log "github.com/sirupsen/logrus"

	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/repo"
	"helm.sh/helm/v3/pkg/chart"
    "helm.sh/helm/v3/pkg/chart/loader"
    "helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/helmpath"
)

const (
	requirementsName  = "requirements.yaml"
	chartMetadataName = "Chart.yaml"
	filePrefix        = "file://"
)

// ListOutdatedDependencies returns a list of outdated dependencies of the given chart.
func ListOutdatedDependencies(chartPath string, settings *cli.EnvSettings, dependencyFilter *Filter) ([]*Result, error) {
	chartDeps, err := loadDependencies(chartPath, dependencyFilter)
	if err != nil {
// 		if err == chartutil.ErrRequirementsNotFound {
// 			fmt.Printf("Chart %v has no requirements.\n", chartPath)
// 			return nil, nil
// 		}
		return nil, err
	}

    // Update local cached repositories
	if err = parallelRepoUpdate(chartDeps, settings); err != nil {
		return nil, err
	}

	var res []*Result
	for _, dep := range chartDeps {
		depVersion, err := semver.NewVersion(dep.Version)
		if err != nil {
			fmt.Printf("Error creating semVersion for dependency %s: %s", dep.Name, err.Error())
			continue
		}

		latestVersion, err := findLatestVersionOfDependency(dep, settings)
		if err != nil {
			fmt.Printf("Error getting latest version of %s: %s\n", dep.Name, err.Error())
			continue
		}

		if depVersion.LessThan(latestVersion) {
			currentVersion, err := semver.NewVersion(dep.Version)
			if err != nil {
				continue
			}

			res = append(res, &Result{
				Dependency:     dep,
				CurrentVersion: currentVersion,
				LatestVersion:  latestVersion,
			})
		}
	}

	return sortResultsAlphabetically(res), nil
}

// UpdateDependencies updates the dependencies of the given chart.
func UpdateDependencies(chartPath string, reqsToUpdate []*Result, indent int) error {
    log.Info("Updating outdated dependencies of chart '" + chartPath +"''")
	c, err := loader.Load(chartPath)
	if err != nil {
		return err
	}

	reqs := c.Metadata.Dependencies

	for _, newDep := range reqsToUpdate {
		for _, oldDep := range reqs {
			if newDep.Name == oldDep.Name && newDep.Repository == newDep.Repository {
			    log.Debug("Updating dependency " + oldDep.Name + " from " + oldDep.Version + " in " + newDep.LatestVersion.String())
				oldDep.Version = newDep.LatestVersion.String()
			}
		}
	}

	reqs = sortRequirementsAlphabetically(reqs)
	if err := writeRequirements(chartPath, reqs, indent); err != nil {
		return err
	}

    return nil;
}

// IncrementChart version increments the patch version of the Chart.
func IncrementChartVersion(chartPath string, incType IncType) error {
	c, err := loader.Load(chartPath)
	if err != nil {
		return err
	}

	chartVersion, err := getChartVersion(c)
	if err != nil {
		return err
	}

	var newVersion semver.Version
	switch incType {
	case IncTypes.Major:
		newVersion = chartVersion.IncMajor()
	case IncTypes.Minor:
		newVersion = chartVersion.IncMinor()
	default:
		newVersion = chartVersion.IncPatch()
	}

	c.Metadata.Version = newVersion.String()
	return writeChartMetadata(chartPath, c.Metadata)
}

// GetChartName returns the name of the chart in the given path or an error.
func GetChartName(chartPath string) (string, error) {
	c, err := loader.Load(chartPath)
	if err != nil {
		return "", err
	}

	return c.Metadata.Name, nil
}

// loadDependencies loads the dependencies of the given chart.
func loadDependencies(chartPath string, f *Filter) ([]*chart.Dependency, error) {
	c, err := loader.Load(chartPath)
	if err != nil {
		return nil, err
	}

	reqs := c.Metadata.Dependencies

	var deps []*chart.Dependency
	for _, d := range reqs {
		if strings.Contains(d.Repository, filePrefix) {
			d.Repository = fmt.Sprintf("%s%s", filePrefix, filepath.Join(chartPath, strings.TrimPrefix(d.Repository, filePrefix)))
		}
		deps = append(deps, d)
	}

	reqs = f.FilterDependencies(deps)
	return reqs, nil
}

// findLatestVersionOfDependency returns the latest version of the given dependency in the repository.
func findLatestVersionOfDependency(dep *chart.Dependency, settings *cli.EnvSettings) (*semver.Version, error) {
	// Handle local dependencies.
	if strings.Contains(dep.Repository, filePrefix) {
		c, err := loader.Load(strings.TrimPrefix(dep.Repository, filePrefix))
		if err != nil {
			return nil, err
		}
		return semver.NewVersion(c.Metadata.Version)
	}

	// Read the index file for the repository to get chart information and return chart URL
	fmt.Printf("Loading cache index file for repository %s from cache dir %s\n", dep.Repository, settings.RepositoryCache)
	repoIndex, err := repo.LoadIndexFile(filepath.Join(settings.RepositoryCache, helmpath.CacheIndexFile(normalizeRepoName(dep.Repository))))
	if err != nil {
		return nil, err
	}

	// With no version given the highest one is returned.
	cv, err := repoIndex.Get(dep.Name, "")
	if err != nil {
		return nil, err
	}

	return semver.NewVersion(cv.Version)
}

func sortRequirementsAlphabetically(reqs []*chart.Dependency) []*chart.Dependency {
	sort.Slice(reqs, func(i, j int) bool {
		return reqs[i].Name < reqs[j].Name
	})
	return reqs
}

func parallelRepoUpdate(chartDeps []*chart.Dependency, settings *cli.EnvSettings) error {
	var repos []string
	for _, dep := range chartDeps {
		if !stringSliceContains(repos, dep.Repository) && !strings.Contains(dep.Repository, filePrefix) {
			repos = append(repos, dep.Repository)
		}
	}

	var wg sync.WaitGroup
	for _, c := range repos {
		tmpRepo := &repo.Entry{
			Name: normalizeRepoName(c),
			URL:  c,
		}

		r, err := repo.NewChartRepository(tmpRepo, getter.All(settings))
		if err != nil {
			return err
		}

		wg.Add(1)
		go func(r *repo.ChartRepository) {
			if idx,err := r.DownloadIndexFile(); err != nil {
				fmt.Printf("unable to get an update from the %q chart repository (%s):\n\t%s\n", r.Config.Name, r.Config.URL, err)
			} else {
				fmt.Printf("successfully got an update from the %q chart repository, updated %s\n", r.Config.URL, idx)
			}
			wg.Done()
		}(r)
	}
	wg.Wait()
	return nil
}
