// Copyright © 2017 Aaron Donovan <amdonov@gmail.com>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package saml

import (
	"fmt"

	"github.com/google/uuid"
)

func NewID() string {
	return fmt.Sprintf("_%s", uuid.New())
}

func NewIssuer(issuer string) *Issuer {
	return &Issuer{Format: "urn:oasis:names:tc:SAML:2.0:nameid-format:entity", Value: issuer}
}
