package gendoc

import (
	"encoding/json"
	"io/ioutil"
	"testing"

	"github.com/kylelemons/godebug/pretty"
)

func TestToSchema(t *testing.T) {
	tests := []struct {
		s    schema
		want Schema
	}{
		{
			s: schema{
				ID:   "UsersResponse",
				Type: "object",
				Properties: map[string]schema{
					"users": {
						Type: "array",
						Items: &schema{
							Ref: "User",
						},
					},
					"nextPageToken": {Type: "string"},
				},
			},
			want: Schema{
				Name: "UsersResponse",
				Type: "object",
				Children: []Schema{
					{
						Name: "nextPageToken",
						Type: "string",
					},
					{
						Name: "users",
						Type: "array",
						Children: []Schema{
							{
								Ref: "User",
							},
						},
					},
				},
			},
		},
		{
			s: schema{
				ID:   "UserCreateResponse",
				Type: "object",
				Properties: map[string]schema{
					"user": {
						Type: "object",
						Ref:  "User",
					},
					"resetPasswordLink": {Type: "string"},
					"emailSent":         {Type: "boolean"},
				},
			},
			want: Schema{
				Name: "UserCreateResponse",
				Type: "object",
				Children: []Schema{
					{
						Name: "emailSent",
						Type: "boolean",
					},
					{
						Name: "resetPasswordLink",
						Type: "string",
					},
					{
						Name: "user",
						Type: "object",
						Ref:  "User",
					},
				},
			},
		},
	}

	for i, tt := range tests {
		got := tt.s.toSchema()
		if diff := pretty.Compare(got, tt.want); diff != "" {
			t.Errorf("case %d: got != want: %s", i, diff)
		}
	}
}

func TestUnmarsal(t *testing.T) {
	tests := []struct {
		file string
		want doc
	}{
		{
			file: "testdata/admin.json",
			want: doc{
				Name:              "adminschema",
				Version:           "v1",
				Title:             "Dex Admin API",
				Description:       "The Dex Admin API.",
				DocumentationLink: "http://github.com/coreos/dex",
				Protocol:          "rest",
				BasePath:          "/api/v1/",
				Schemas: map[string]schema{
					"Admin": schema{
						ID:          "Admin",
						Type:        "object",
						Description: "Admin represents an admin user within the database",
						Properties: map[string]schema{
							"id":       {Type: "string"},
							"email":    {Type: "string"},
							"password": {Type: "string"},
						},
					},
					"State": schema{
						ID:          "State",
						Type:        "object",
						Description: "Admin represents dex data within.",
						Properties: map[string]schema{
							"AdminUserCreated": {Type: "boolean"},
						},
					},
				},
				Resources: map[string]methods{
					"Admin": methods{
						Methods: map[string]resource{
							"Get": resource{
								Description: "Retrieve information about an admin user.",
								Method:      "GET",
								Path:        "admin/{id}",
								Parameters: map[string]param{
									"id": param{
										Type:     "string",
										Required: true,
										Location: "path",
									},
								},
								Response: &ref{"Admin"},
							},
							"Create": resource{
								Description: "Create a new admin user.",
								Method:      "POST",
								Path:        "admin",
								Request:     &ref{"Admin"},
								Response:    &ref{"Admin"},
							},
						},
					},
					"State": methods{
						Methods: map[string]resource{
							"Get": resource{
								Description: "Get the state of the Dex DB",
								Method:      "GET",
								Path:        "state",
								Response:    &ref{"State"},
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		data, err := ioutil.ReadFile(tt.file)
		if err != nil {
			t.Errorf("case %q: read file failed %v", tt.file, err)
			continue
		}
		var d doc
		if err := json.Unmarshal(data, &d); err != nil {
			t.Errorf("case %q: failed to unmarshal %v", tt.file, err)
			continue
		}
		if diff := pretty.Compare(d, tt.want); diff != "" {
			t.Errorf("case %q: got did not match want: %s", tt.file, diff)
		}
	}
}

func TestUnmarshalSchema(t *testing.T) {
	tests := []struct {
		file string
		want map[string]schema
	}{
		{
			file: "testdata/worker.json",
			want: map[string]schema{
				"Error": schema{
					ID:   "Error",
					Type: "object",
					Properties: map[string]schema{
						"error":             {Type: "string"},
						"error_description": {Type: "string"},
					},
				},
				"Client": schema{
					ID:   "Client",
					Type: "object",
					Properties: map[string]schema{
						"id": {Type: "string"},
						"redirectURIs": {
							Type:  "array",
							Items: &schema{Type: "string"},
						},
					},
				},
				"ClientWithSecret": schema{
					ID:   "Client",
					Type: "object",
					Properties: map[string]schema{
						"id":     {Type: "string"},
						"secret": {Type: "string"},
						"redirectURIs": {
							Type:  "array",
							Items: &schema{Type: "string"},
						},
					},
				},
				"ClientPage": schema{
					ID:   "ClientPage",
					Type: "object",
					Properties: map[string]schema{
						"clients": {
							Type: "array",
							Items: &schema{
								Ref: "Client",
							},
						},
						"nextPageToken": {
							Type: "string",
						},
					},
				},
				"User": schema{
					ID:   "User",
					Type: "object",
					Properties: map[string]schema{
						"id":            {Type: "string"},
						"email":         {Type: "string"},
						"displayName":   {Type: "string"},
						"emailVerified": {Type: "boolean"},
						"admin":         {Type: "boolean"},
						"disabled":      {Type: "boolean"},
						"createdAt": {
							Type:   "string",
							Format: "date-time",
						},
					},
				},
				"UserResponse": schema{
					ID:   "UserResponse",
					Type: "object",
					Properties: map[string]schema{
						"user": {Ref: "User"},
					},
				},
				"UsersResponse": schema{
					ID:   "UsersResponse",
					Type: "object",
					Properties: map[string]schema{
						"users": {
							Type: "array",
							Items: &schema{
								Ref: "User",
							},
						},
						"nextPageToken": {Type: "string"},
					},
				},
				"UserCreateRequest": schema{
					ID:   "UserCreateRequest",
					Type: "object",
					Properties: map[string]schema{
						"user": {
							Ref: "User",
						},
						"redirectURL": {Type: "string", Format: "url"},
					},
				},
				"UserCreateResponse": schema{
					ID:   "UserCreateResponse",
					Type: "object",
					Properties: map[string]schema{
						"user": {
							Type: "object",
							Ref:  "User",
						},
						"resetPasswordLink": {Type: "string"},
						"emailSent":         {Type: "boolean"},
					},
				},
				"UserDisableRequest": schema{
					ID:   "UserDisableRequest",
					Type: "object",
					Properties: map[string]schema{
						"disable": {
							Type:        "boolean",
							Description: "If true, disable this user, if false, enable them. No error is signaled if the user state doesn't change.",
						},
					},
				},
				"UserDisableResponse": schema{
					ID:   "UserDisableResponse",
					Type: "object",
					Properties: map[string]schema{
						"ok": {Type: "boolean"},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		data, err := ioutil.ReadFile(tt.file)
		if err != nil {
			t.Errorf("case %q: read file failed %v", tt.file, err)
			continue
		}
		var d doc
		if err := json.Unmarshal(data, &d); err != nil {
			t.Errorf("case %q: failed to unmarshal %v", tt.file, err)
			continue
		}
		if diff := pretty.Compare(d.Schemas, tt.want); diff != "" {
			t.Errorf("case %q: got did not match want: %s", tt.file, diff)
		}
	}
}
