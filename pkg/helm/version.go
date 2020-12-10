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

import "github.com/Masterminds/semver"

// IncType is one of IncTypes.
type IncType string

// IncTypes enumerates available IncType.
var IncTypes = struct {
	Major IncType
	Minor IncType
	Patch IncType
	None  IncType
}{
	"major",
	"minor",
	"patch",
	"none",
}

// IsGreater check whether the given IncType is greater.
func (i IncType) IsGreater(inc IncType) bool {
	switch i {
	case IncTypes.Minor:
		return inc == IncTypes.Major
	case IncTypes.Patch:
		return inc == IncTypes.Major || inc == IncTypes.Minor
	}
	return false
}

// GetIncType returns IncType based on which segment of the Version was changed.
func GetIncType(oldVersion, newVersion *semver.Version) IncType {
	if newVersion.Major() > oldVersion.Major() {
		return IncTypes.Major
	}

	if newVersion.Minor() > oldVersion.Minor() {
		return IncTypes.Minor
	}

	if newVersion.Patch() > oldVersion.Patch() {
		return IncTypes.Patch
	}

	return IncTypes.None
}
