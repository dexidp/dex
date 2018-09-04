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

package idp

import (
	"github.com/amdonov/lite-idp/model"
	"github.com/theherk/viper"
)

type AttributeSource interface {
	AddAttributes(*model.User) error
}

type simpleSource struct {
	users map[string][]*model.Attribute
}

type UserAttributes struct {
	Name       string
	Attributes map[string][]string
}

func (ss *simpleSource) AddAttributes(user *model.User) error {
	if atts, ok := ss.users[user.Name]; ok {
		user.AppendAttributes(atts)
	}
	return nil
}

func NewAttributeSource() (AttributeSource, error) {
	userAttributes := []UserAttributes{}
	err := viper.UnmarshalKey("users", &userAttributes)
	if err != nil {
		return nil, err
	}
	users := make(map[string][]*model.Attribute)
	for i := range userAttributes {
		user := userAttributes[i]
		atts := []*model.Attribute{}
		for key, value := range user.Attributes {
			att := &model.Attribute{Name: key, Value: value}
			atts = append(atts, att)
		}
		users[user.Name] = atts
	}
	return &simpleSource{users}, nil
}
