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

package git

import (
	"fmt"
	"os"
	"strings"

	"github.com/pkg/errors"
	"github.com/sapcc/helm-outdated-dependencies/pkg/cmd"
)

var (
	errGitNoRemote = errors.New("git remote has no remote configured")
	errGithubNoToken = errors.New("GITHUB_TOKEN environment variable no set")
)

// Git wraps the git command line.
type Git struct {
	*cmd.Command

	branchName,
	remoteName,
	authorName,
	authorEmail string
}

// NewGit returns a new Git or an error.
func NewGit(path, authorName, authorEmail string) (*Git, error) {
	c, err := cmd.New("git", "-C", path)
	if err != nil {
		return nil, err
	}

	g := &Git{
		Command:    c,
		branchName: "master",
		remoteName: "origin",
	}

	if _, err := g.GetRemoteURL(); err != nil {
		return nil, err
	}

	if authorName == "" {
		authorName, err = g.GetGlobalUserName()
		if err != nil {
			return nil, err
		}
	}
	g.authorName = authorName

	if authorEmail == "" {
		authorEmail, err = g.GetGlobalUserEmail()
		if err != nil {
			return nil, err
		}
	}
	g.authorEmail = authorEmail

	return g, nil
}

// Commit adds and commits all changes.
func (g *Git) Commit(message string) (string, error) {
	res, err := g.Run(
		"-c", fmt.Sprintf(`user.name="%s"`, g.authorName),
		"-c", fmt.Sprintf(`user.email="%s"`, g.authorEmail),
		"commit",
		"--all",
		"--author", fmt.Sprintf(`"%s <%s>"`, g.authorName, g.authorEmail),
		"--message", fmt.Sprintf(`"%s"`, message),
	)
	if err != nil {
		return "", errors.Wrap(err, "git commit ... failed")
	}
	return res, nil
}

// Diff shows the changes.
func (g *Git) Diff() (string, error) {
	res, err := g.Run("diff")
	if err != nil {
		return "", errors.Wrap(err, "git diff failed")
	}
	return res, nil
}

// PushToMaster rebases and pushes the commit(s) to upstream.
func (g *Git) RebaseAndPushToMaster() (string, error) {
	if out, err := g.PullRebase(); err != nil {
		return out, err
	}

	return g.Push(g.branchName)
}

// Push pushes to the given upstream branch.
func (g *Git) Push(branchName string) (string, error) {
	tmpPushURL, err := g.getPushURL()
	if err != nil {
		return "", err
	}

	res, err := g.Run("push", tmpPushURL, branchName)
	if err != nil {
		return "", errors.Wrap(err, "git push failed")
	}
	return res, nil
}

// PullRebase pulls and rebases.
func (g *Git) PullRebase() (string, error) {
	res, err := g.Run("pull", "--rebase")
	if err != nil {
		return "", errors.Wrap(err, "git pull failed")
	}
	return res, nil
}

// GetRemoteURL returns the remotes URL or an error.
func (g *Git) GetRemoteURL() (string, error) {
	res, err := g.Run("remote", "get-url", g.remoteName)
	if err != nil {
		return "", errors.Wrapf(err, "git remote get-url %s failed", g.remoteName)
	}
	return res, nil
}

// CreateAndCheckoutBranch does what it says.
func (g *Git) CreateAndCheckoutBranch(branchName string) (string, error) {
	res, err := g.Run("checkout", "-b", branchName)
	if err != nil {
		return "", errors.Wrapf(err, "git checkout -b %s failed", branchName)
	}
	return res, nil
}

// Checkout branch.
func (g *Git) CheckoutBranch(branchName string) (string, error) {
	res, err := g.Run("checkout", branchName)
	if err != nil {
		return "", errors.Wrapf(err, "git checkout %s failed", branchName)
	}
	return res, nil
}

// GetGlobalUserName returns user name from gits global config.
func (g *Git) GetGlobalUserName() (string, error) {
	return g.Run("config", "--global", "user.name")
}

// GetGlobalUserEmail returns user email from gits global config.
func (g *Git) GetGlobalUserEmail() (string, error) {
	return g.Run("config", "--global", "user.email")
}

func (g *Git) getPushURL() (string, error) {
	remote, err := g.GetRemoteURL()
	if err != nil {
		return "", err
	}

	ghToken, ok := os.LookupEnv("GITHUB_TOKEN")
	if !ok {
		return "", errGithubNoToken
	}

	remote = strings.TrimPrefix(remote, "https://")
	remote = strings.TrimPrefix(remote, "git@")
	remote = strings.ReplaceAll(remote, ":", "/")

	return fmt.Sprintf("https://%s:%s@%s", g.authorName, ghToken, remote), nil
}
