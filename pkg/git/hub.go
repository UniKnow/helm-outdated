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

	"github.com/pkg/errors"
	"github.com/sapcc/helm-outdated-dependencies/pkg/cmd"
)

// Hub used for interacting with the github.com API.
type Hub struct {
	*cmd.Command
	baseBranch string
}

// NewHub returns a new Hub or an error.
func NewHub(path string) (*Hub, error) {
	c, err := cmd.New("hub", "-C", path)
	if err != nil {
		return nil, err
	}

	return &Hub{
		Command:    c,
		baseBranch: "master",
	}, nil
}

// OpenPullRequestToMaster opens a new pull request on github.com to master branch.
func (h *Hub) OpenPullRequestToMaster(fromBranch, title, description string) (string, error) {
	res, err := h.Run(
		"pull-request",
		"--no-edit",
		"--base", h.baseBranch,
		"--head", fromBranch,
		"--message", fmt.Sprintf(`%s`, title),
		"--message", fmt.Sprintf(`%s`, description),
	)
	if err != nil {
		return "", errors.Wrap(err, "hub pull-request ... failed")
	}
	return "Opened PR: " + res, nil
}
