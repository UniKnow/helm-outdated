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

package cmd

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"


    log "github.com/sirupsen/logrus"

	"github.com/gosuri/uitable"
	"github.com/uniknow/helm-outdated/pkg/git"
	"github.com/uniknow/helm-outdated/pkg/helm"
	"github.com/spf13/cobra"

	"helm.sh/helm/v3/pkg/cli"
)

type updateCmd struct {
	chartPath               string
	maxColumnWidth          uint
	indent                  int
	isIncrementChartVersion bool
	dependencyFilter        *helm.Filter
	git                     *git.Git
	hub                     *git.Hub

	// **Experimental**
	// isAutoUpdate updates the dependencies, increments version of the chart with the dependency and (git) commits the changes.
	isAutoUpdate,
	isOnlyPullRequest bool
	authorName,
	authorEmail string
}

var updateLongUsage = `
Update outdated dependencies of a given chart to their latest version.

Examples:
  # Update dependencies of the given chart.
  $ helm outdated update <chartPath>

	# Only update specific dependencies of the given chart.
	$ helm outdated update <chartPath> --dependencies kube-state-metrics,prometheus-operator
`

func newUpdateOutdatedDependenciesCmd() *cobra.Command {
	u := &updateCmd{
		dependencyFilter: &helm.Filter{},
		maxColumnWidth:   60,
	}

	cmd := &cobra.Command{
		Use:          "update",
		Long:         updateLongUsage,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
            if debug, err := cmd.Flags().GetBool("debug"); err == nil {
                if debug == true {
                    log.SetLevel(log.DebugLevel)
                } else {
                    log.SetLevel(log.InfoLevel)
                }
            }

			if maxColumnWidth, err := cmd.Flags().GetUint("max-column-width"); err == nil {
				u.maxColumnWidth = maxColumnWidth
			}

			if repositories, err := cmd.Flags().GetStringSlice("repositories"); err == nil {
				u.dependencyFilter.Repositories = repositories
			}

			if deps, err := cmd.Flags().GetStringSlice("dependencies"); err == nil {
				u.dependencyFilter.DependencyNames = deps
			}

			path := "."
			if len(args) > 0 {
				path = args[0]
			}

			path, err := filepath.Abs(path)
			if err != nil {
				return err
			}
			u.chartPath = path

			return u.update()
		},
	}

	addCommonFlags(cmd)
	cmd.Flags().BoolVarP(&u.isIncrementChartVersion, "increment-chart-version", "", false, "Increment the version of the Helm chart if requirements are updated.")
	cmd.Flags().IntVarP(&u.indent, "indent", "", 4, "Indent to use when writing the requirements.yaml .")

	// **Experimental** Update dependencies of the given chart, commit and push to upstream using git.
	cmd.Flags().BoolVar(&u.isAutoUpdate, "auto-update", false, "**Experimental** Update dependencies of the given chart, commit and push to upstream using git.")
	cmd.Flags().StringVar(&u.authorName, "author-name", "", "The name of the author and committer to be used when auto update is enabled.")
	cmd.Flags().StringVar(&u.authorEmail, "author-email", "", "The email of the author and committer to be used when auto update is enabled.")
	cmd.Flags().BoolVar(&u.isOnlyPullRequest, "only-pull-requests", false, "Only use pull requests. Do not commit minor changes to master branch.")

	return cmd
}

func (u *updateCmd) update() error {
	outdatedDeps, err := helm.ListOutdatedDependencies(u.chartPath, cli.New(), u.dependencyFilter)
	if err != nil {
		return err
	}

	if len(outdatedDeps) == 0 {
		fmt.Println("All charts up-to-date.")
		return nil
	}
	fmt.Println(u.formatResults(outdatedDeps))

	if u.isIncrementChartVersion || u.isAutoUpdate {
	    fmt.Println("UPDATING CHART VERSION")
		if err = helm.IncrementChartVersion(u.chartPath, helm.IncTypes.Patch); err != nil {
		    fmt.Println("ERROR OCCURRED WHILE UPDATING CHART VERSION")
			return err
		}
	}

	fmt.Println("UPDATING DEPENDENCIES")
	if err := helm.UpdateDependencies(u.chartPath, outdatedDeps, u.indent); err != nil {
		fmt.Println("ERROR OCCURRED WHILE UPDATING DEPENDENCIES")
		return err
	}

	// Return here if the auto update is not enabled.
	if !u.isAutoUpdate {
		return nil
	}

	// maxIncType is used to keep track of the version changes when updating dependencies.
	maxIncType := helm.IncTypes.Patch
	depNames := make([]string, len(outdatedDeps))
	for idx, dep := range outdatedDeps {
		if i := helm.GetIncType(dep.CurrentVersion, dep.LatestVersion); maxIncType.IsGreater(i) {
			maxIncType = i
		}
		depName := dep.Alias
		if depName == "" {
			depName = dep.Name
		}
		depNames[idx] = fmt.Sprintf("%s@%s",  depName, dep.LatestVersion)
	}

	chartName, err := helm.GetChartName(u.chartPath)
	if err != nil {
		return err
	}

	commitMessage := fmt.Sprintf("[%s] updated dependency to %s", chartName, strings.Join(depNames, ", "))

	// If potential breaking changes are expected, use a pull request.
	if u.isOnlyPullRequest || maxIncType == helm.IncTypes.Major || maxIncType == helm.IncTypes.Minor {
		return u.upstreamMajorChanges(commitMessage, chartName)
	}

	return u.upstreamMinorChanges(commitMessage)
}

// upstreamMinorChanges commits the changes to the master branch of the upstream github repository.
func (u *updateCmd) upstreamMinorChanges(commitMessage string) error {
	g, err := git.NewGit(u.chartPath, u.authorName, u.authorEmail)
	if err != nil {
		return err
	}

	res, err := g.Diff()
	if err != nil {
		return err
	}
	fmt.Println(res)

	res, err = g.Commit(commitMessage)
	if err != nil {
		return err
	}
	fmt.Println(res)

	res, err = g.RebaseAndPushToMaster()
	fmt.Println(res)
	return err
}

// upstreamMajorChanges same as upstreamMinorChanges but via github.com pull request.
func (u *updateCmd) upstreamMajorChanges(commitMessage, chartName string) error {
	g, err := git.NewGit(u.chartPath, u.authorName, u.authorEmail)
	if err != nil {
		return err
	}

	branchName := fmt.Sprintf("%s-%d", chartName, time.Now().UTC().Unix())
	res, err := g.CreateAndCheckoutBranch(branchName)
	if err != nil {
		return err
	}
	defer g.CheckoutBranch("master")

	res, err = g.Diff()
	if err != nil {
		return err
	}
	fmt.Println(res)

	res, err = g.Commit(commitMessage)
	if err != nil {
		return err
	}
	fmt.Println(res)

	res, err = g.Push(branchName)
	if err != nil {
		return err
	}
	fmt.Println(res)

	hub, err := git.NewHub(u.chartPath)
	if err != nil {
		return err
	}

	res, err = hub.OpenPullRequestToMaster(branchName, fmt.Sprintf("[%s] updating dependencies", chartName), commitMessage)
	fmt.Println(res)
	return err
}

func (u *updateCmd) formatResults(results []*helm.Result) string {
	if len(results) == 0 {
		return "All charts up to date."
	}
	table := uitable.New()
	table.MaxColWidth = u.maxColumnWidth
	table.AddRow("Updating the following dependencies to their latest version:")
	table.AddRow("ALIAS", "VERSION", "LATEST_VERSION", "REPOSITORY")
	for _, r := range results {
		name := r.Alias
		if name == "" {
			name = r.Name
		}
		table.AddRow(name, r.Version, r.LatestVersion, r.Repository)
	}
	return table.String()
}
