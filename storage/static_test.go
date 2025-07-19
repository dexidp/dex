package storage

import (
	"context"
	"errors"
	"log/slog"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type mockStorage struct {
	Storage

	clients    map[string]Client
	connectors map[string]Connector
	passwords  map[string]Password
	updateErr  error
	listErr    error
	getErr     error
	createErr  error
	deleteErr  error
}

func newMockStorage() *mockStorage {
	return &mockStorage{
		clients:    make(map[string]Client),
		connectors: make(map[string]Connector),
		passwords:  make(map[string]Password),
	}
}

func (m *mockStorage) GetClient(ctx context.Context, id string) (Client, error) {
	if m.getErr != nil {
		return Client{}, m.getErr
	}
	c, ok := m.clients[id]
	if !ok {
		return Client{}, ErrNotFound
	}
	return c, nil
}

func (m *mockStorage) ListClients(ctx context.Context) ([]Client, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	cs := make([]Client, 0, len(m.clients))
	for _, c := range m.clients {
		cs = append(cs, c)
	}
	sort.Slice(cs, func(i, j int) bool { return cs[i].ID < cs[j].ID })
	return cs, nil
}

func (m *mockStorage) CreateClient(ctx context.Context, c Client) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.clients[c.ID] = c
	return nil
}

func (m *mockStorage) DeleteClient(ctx context.Context, id string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	delete(m.clients, id)
	return nil
}

func (m *mockStorage) UpdateClient(ctx context.Context, id string, updater func(old Client) (Client, error)) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	old, ok := m.clients[id]
	if !ok {
		return ErrNotFound
	}
	newC, err := updater(old)
	if err != nil {
		return err
	}
	m.clients[id] = newC
	return nil
}

func (m *mockStorage) GetConnector(ctx context.Context, id string) (Connector, error) {
	if m.getErr != nil {
		return Connector{}, m.getErr
	}
	c, ok := m.connectors[id]
	if !ok {
		return Connector{}, ErrNotFound
	}
	return c, nil
}

func (m *mockStorage) ListConnectors(ctx context.Context) ([]Connector, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	cs := make([]Connector, 0, len(m.connectors))
	for _, c := range m.connectors {
		cs = append(cs, c)
	}
	sort.Slice(cs, func(i, j int) bool { return cs[i].ID < cs[j].ID })
	return cs, nil
}

func (m *mockStorage) CreateConnector(ctx context.Context, c Connector) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.connectors[c.ID] = c
	return nil
}

func (m *mockStorage) DeleteConnector(ctx context.Context, id string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	delete(m.connectors, id)
	return nil
}

func (m *mockStorage) UpdateConnector(ctx context.Context, id string, updater func(old Connector) (Connector, error)) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	old, ok := m.connectors[id]
	if !ok {
		return ErrNotFound
	}
	newC, err := updater(old)
	if err != nil {
		return err
	}
	m.connectors[id] = newC
	return nil
}

func (m *mockStorage) GetPassword(ctx context.Context, email string) (Password, error) {
	if m.getErr != nil {
		return Password{}, m.getErr
	}
	p, ok := m.passwords[strings.ToLower(email)]
	if !ok {
		return Password{}, ErrNotFound
	}
	return p, nil
}

func (m *mockStorage) ListPasswords(ctx context.Context) ([]Password, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	ps := make([]Password, 0, len(m.passwords))
	for _, p := range m.passwords {
		ps = append(ps, p)
	}
	sort.Slice(ps, func(i, j int) bool { return strings.ToLower(ps[i].Email) < strings.ToLower(ps[j].Email) })
	return ps, nil
}

func (m *mockStorage) CreatePassword(ctx context.Context, p Password) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.passwords[strings.ToLower(p.Email)] = p
	return nil
}

func (m *mockStorage) DeletePassword(ctx context.Context, email string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	delete(m.passwords, strings.ToLower(email))
	return nil
}

func (m *mockStorage) UpdatePassword(ctx context.Context, email string, updater func(old Password) (Password, error)) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	old, ok := m.passwords[strings.ToLower(email)]
	if !ok {
		return ErrNotFound
	}
	newP, err := updater(old)
	if err != nil {
		return err
	}
	m.passwords[strings.ToLower(email)] = newP
	return nil
}

func TestWithStaticClients(t *testing.T) {
	type args struct {
		s             Storage
		staticClients []Client
	}
	tests := []struct {
		name string
		args args
		want Storage
	}{
		{
			name: "basic",
			args: args{
				s:             newMockStorage(),
				staticClients: []Client{{ID: "static1"}, {ID: "static2"}},
			},
			want: staticClientsStorage{
				Storage:     newMockStorage(),
				clients:     []Client{{ID: "static1"}, {ID: "static2"}},
				clientsByID: map[string]Client{"static1": {ID: "static1"}, "static2": {ID: "static2"}},
			},
		},
		{
			name: "empty static",
			args: args{
				s:             newMockStorage(),
				staticClients: []Client{},
			},
			want: staticClientsStorage{
				Storage:     newMockStorage(),
				clients:     []Client{},
				clientsByID: map[string]Client{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := WithStaticClients(tt.args.s, tt.args.staticClients)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestWithStaticConnectors(t *testing.T) {
	type args struct {
		s                Storage
		staticConnectors []Connector
	}
	tests := []struct {
		name string
		args args
		want Storage
	}{
		{
			name: "basic",
			args: args{
				s:                newMockStorage(),
				staticConnectors: []Connector{{ID: "static1"}, {ID: "static2"}},
			},
			want: staticConnectorsStorage{
				Storage:        newMockStorage(),
				connectors:     []Connector{{ID: "static1"}, {ID: "static2"}},
				connectorsByID: map[string]Connector{"static1": {ID: "static1"}, "static2": {ID: "static2"}},
			},
		},
		{
			name: "empty static",
			args: args{
				s:                newMockStorage(),
				staticConnectors: []Connector{},
			},
			want: staticConnectorsStorage{
				Storage:        newMockStorage(),
				connectors:     []Connector{},
				connectorsByID: map[string]Connector{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := WithStaticConnectors(tt.args.s, tt.args.staticConnectors)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestWithStaticPasswords(t *testing.T) {
	logger := slog.Default()
	type args struct {
		s               Storage
		staticPasswords []Password
		logger          *slog.Logger
	}
	tests := []struct {
		name string
		args args
		want Storage
	}{
		{
			name: "basic",
			args: args{
				s:               newMockStorage(),
				staticPasswords: []Password{{Email: "static1@example.com"}, {Email: "Static2@Example.com"}},
				logger:          logger,
			},
			want: staticPasswordsStorage{
				Storage:          newMockStorage(),
				passwords:        []Password{{Email: "static1@example.com"}, {Email: "Static2@Example.com"}},
				passwordsByEmail: map[string]Password{"static1@example.com": {Email: "static1@example.com"}, "static2@example.com": {Email: "Static2@Example.com"}},
				logger:           logger,
			},
		},
		{
			name: "empty static",
			args: args{
				s:               newMockStorage(),
				staticPasswords: []Password{},
				logger:          logger,
			},
			want: staticPasswordsStorage{
				Storage:          newMockStorage(),
				passwords:        []Password{},
				passwordsByEmail: map[string]Password{},
				logger:           logger,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := WithStaticPasswords(tt.args.s, tt.args.staticPasswords, tt.args.logger)
			require.Equal(t, tt.want, got)
		})
	}
}

func Test_staticClientsStorage_CreateClient(t *testing.T) {
	ctx := context.Background()
	type fields struct {
		Storage     Storage
		clients     []Client
		clientsByID map[string]Client
	}
	type args struct {
		ctx context.Context
		c   Client
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "create dynamic",
			fields: fields{
				Storage: newMockStorage(),
			},
			args: args{
				ctx: ctx,
				c:   Client{ID: "dynamic"},
			},
			wantErr: false,
		},
		{
			name: "create static",
			fields: fields{
				Storage:     newMockStorage(),
				clientsByID: map[string]Client{"static": {ID: "static"}},
			},
			args: args{
				ctx: ctx,
				c:   Client{ID: "static"},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := staticClientsStorage{
				Storage:     tt.fields.Storage,
				clients:     tt.fields.clients,
				clientsByID: tt.fields.clientsByID,
			}
			err := s.CreateClient(tt.args.ctx, tt.args.c)
			if tt.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), "read-only cannot create client")
				return
			}
			require.NoError(t, err)
			got, err := tt.fields.Storage.GetClient(ctx, tt.args.c.ID)
			require.NoError(t, err)
			require.Equal(t, tt.args.c, got)
		})
	}
}

func Test_staticClientsStorage_DeleteClient(t *testing.T) {
	ctx := context.Background()
	type fields struct {
		Storage     Storage
		clients     []Client
		clientsByID map[string]Client
	}
	type args struct {
		ctx context.Context
		id  string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		setup   func(s Storage)
		wantErr bool
	}{
		{
			name: "delete dynamic",
			fields: fields{
				Storage: newMockStorage(),
			},
			args: args{
				ctx: ctx,
				id:  "dynamic",
			},
			setup: func(s Storage) {
				s.CreateClient(ctx, Client{ID: "dynamic"})
			},
			wantErr: false,
		},
		{
			name: "delete static",
			fields: fields{
				Storage:     newMockStorage(),
				clientsByID: map[string]Client{"static": {ID: "static"}},
			},
			args: args{
				ctx: ctx,
				id:  "static",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := staticClientsStorage{
				Storage:     tt.fields.Storage,
				clients:     tt.fields.clients,
				clientsByID: tt.fields.clientsByID,
			}
			if tt.setup != nil {
				tt.setup(s.Storage)
			}
			err := s.DeleteClient(tt.args.ctx, tt.args.id)
			if tt.wantErr {
				require.Error(t, err)
				if tt.args.id != "missing" {
					require.Contains(t, err.Error(), "read-only cannot delete client")
				}
				return
			}
			require.NoError(t, err)
			_, err = s.Storage.GetClient(ctx, tt.args.id)
			require.Error(t, err)
		})
	}
}

func Test_staticClientsStorage_GetClient(t *testing.T) {
	ctx := context.Background()
	type fields struct {
		Storage     Storage
		clients     []Client
		clientsByID map[string]Client
	}
	type args struct {
		ctx context.Context
		id  string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		setup   func(s Storage)
		want    Client
		wantErr bool
	}{
		{
			name: "get static",
			fields: fields{
				Storage:     newMockStorage(),
				clientsByID: map[string]Client{"static": {ID: "static", Name: "static client"}},
			},
			args: args{
				ctx: ctx,
				id:  "static",
			},
			want: Client{ID: "static", Name: "static client"},
		},
		{
			name: "get dynamic",
			fields: fields{
				Storage: newMockStorage(),
			},
			args: args{
				ctx: ctx,
				id:  "dynamic",
			},
			setup: func(s Storage) {
				s.CreateClient(ctx, Client{ID: "dynamic", Name: "dynamic client"})
			},
			want: Client{ID: "dynamic", Name: "dynamic client"},
		},
		{
			name: "missing",
			fields: fields{
				Storage: newMockStorage(),
			},
			args: args{
				ctx: ctx,
				id:  "missing",
			},
			wantErr: true,
		},
		{
			name: "prefer static over dynamic",
			fields: fields{
				Storage:     newMockStorage(),
				clientsByID: map[string]Client{"overlap": {ID: "overlap", Name: "static overlap"}},
			},
			args: args{
				ctx: ctx,
				id:  "overlap",
			},
			setup: func(s Storage) {
				s.CreateClient(ctx, Client{ID: "overlap", Name: "dynamic overlap"})
			},
			want: Client{ID: "overlap", Name: "static overlap"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := staticClientsStorage{
				Storage:     tt.fields.Storage,
				clients:     tt.fields.clients,
				clientsByID: tt.fields.clientsByID,
			}
			if tt.setup != nil {
				tt.setup(s.Storage)
			}
			got, err := s.GetClient(tt.args.ctx, tt.args.id)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func Test_staticClientsStorage_ListClients(t *testing.T) {
	ctx := context.Background()
	type fields struct {
		Storage     Storage
		clients     []Client
		clientsByID map[string]Client
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		setup   func(s Storage)
		want    []Client
		wantErr bool
	}{
		{
			name: "static only",
			fields: fields{
				Storage:     newMockStorage(),
				clients:     []Client{{ID: "static1"}, {ID: "static2"}},
				clientsByID: map[string]Client{"static1": {ID: "static1"}, "static2": {ID: "static2"}},
			},
			args: args{
				ctx: ctx,
			},
			want: []Client{{ID: "static1"}, {ID: "static2"}},
		},
		{
			name: "dynamic only",
			fields: fields{
				Storage: newMockStorage(),
			},
			args: args{
				ctx: ctx,
			},
			setup: func(s Storage) {
				s.CreateClient(ctx, Client{ID: "dynamic1"})
				s.CreateClient(ctx, Client{ID: "dynamic2"})
			},
			want: []Client{{ID: "dynamic1"}, {ID: "dynamic2"}},
		},
		{
			name: "mixed",
			fields: fields{
				Storage:     newMockStorage(),
				clients:     []Client{{ID: "static"}},
				clientsByID: map[string]Client{"static": {ID: "static"}},
			},
			args: args{
				ctx: ctx,
			},
			setup: func(s Storage) {
				s.CreateClient(ctx, Client{ID: "dynamic"})
			},
			want: []Client{{ID: "dynamic"}, {ID: "static"}},
		},
		{
			name: "overlap prefers static",
			fields: fields{
				Storage:     newMockStorage(),
				clients:     []Client{{ID: "overlap", Name: "static"}},
				clientsByID: map[string]Client{"overlap": {ID: "overlap", Name: "static"}},
			},
			args: args{
				ctx: ctx,
			},
			setup: func(s Storage) {
				s.CreateClient(ctx, Client{ID: "overlap", Name: "dynamic"})
			},
			want: []Client{{ID: "overlap", Name: "static"}},
		},
		{
			name: "base list error",
			fields: fields{
				Storage: newMockStorage(),
			},
			args: args{
				ctx: ctx,
			},
			setup: func(s Storage) {
				s.(*mockStorage).listErr = errors.New("list error")
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := staticClientsStorage{
				Storage:     tt.fields.Storage,
				clients:     tt.fields.clients,
				clientsByID: tt.fields.clientsByID,
			}
			if tt.setup != nil {
				tt.setup(s.Storage)
			}
			got, err := s.ListClients(tt.args.ctx)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			sort.Slice(got, func(i, j int) bool { return got[i].ID < got[j].ID })
			sort.Slice(tt.want, func(i, j int) bool { return tt.want[i].ID < tt.want[j].ID })
			require.Equal(t, tt.want, got)
		})
	}
}

func Test_staticClientsStorage_UpdateClient(t *testing.T) {
	ctx := context.Background()
	type fields struct {
		Storage     Storage
		clients     []Client
		clientsByID map[string]Client
	}
	type args struct {
		ctx     context.Context
		id      string
		updater func(old Client) (Client, error)
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		setup   func(s Storage)
		wantErr bool
	}{
		{
			name: "update dynamic",
			fields: fields{
				Storage: newMockStorage(),
			},
			args: args{
				ctx: ctx,
				id:  "dynamic",
				updater: func(old Client) (Client, error) {
					old.Name = "new"
					return old, nil
				},
			},
			setup: func(s Storage) {
				s.CreateClient(ctx, Client{ID: "dynamic", Name: "old"})
			},
			wantErr: false,
		},
		{
			name: "update static",
			fields: fields{
				Storage:     newMockStorage(),
				clientsByID: map[string]Client{"static": {ID: "static"}},
			},
			args: args{
				ctx: ctx,
				id:  "static",
				updater: func(old Client) (Client, error) {
					old.Name = "new"
					return old, nil
				},
			},
			wantErr: true,
		},
		{
			name: "update missing",
			fields: fields{
				Storage: newMockStorage(),
			},
			args: args{
				ctx: ctx,
				id:  "missing",
				updater: func(old Client) (Client, error) {
					return old, nil
				},
			},
			wantErr: true,
		},
		{
			name: "base update error",
			fields: fields{
				Storage: newMockStorage(),
			},
			args: args{
				ctx: ctx,
				id:  "dynamic",
				updater: func(old Client) (Client, error) {
					return old, nil
				},
			},
			setup: func(s Storage) {
				s.(*mockStorage).updateErr = errors.New("update error")
				s.CreateClient(ctx, Client{ID: "dynamic"})
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := staticClientsStorage{
				Storage:     tt.fields.Storage,
				clients:     tt.fields.clients,
				clientsByID: tt.fields.clientsByID,
			}
			if tt.setup != nil {
				tt.setup(s.Storage)
			}
			err := s.UpdateClient(tt.args.ctx, tt.args.id, tt.args.updater)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			got, err := s.Storage.GetClient(ctx, tt.args.id)
			require.NoError(t, err)
			require.Equal(t, "new", got.Name)
		})
	}
}

func Test_staticClientsStorage_isStatic(t *testing.T) {
	type fields struct {
		Storage     Storage
		clients     []Client
		clientsByID map[string]Client
	}
	type args struct {
		id string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "static",
			fields: fields{
				clientsByID: map[string]Client{"static": {}},
			},
			args: args{
				id: "static",
			},
			want: true,
		},
		{
			name: "dynamic",
			fields: fields{
				clientsByID: map[string]Client{},
			},
			args: args{
				id: "dynamic",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := staticClientsStorage{
				Storage:     tt.fields.Storage,
				clients:     tt.fields.clients,
				clientsByID: tt.fields.clientsByID,
			}
			got := s.isStatic(tt.args.id)
			require.Equal(t, tt.want, got)
		})
	}
}

func Test_staticConnectorsStorage_CreateConnector(t *testing.T) {
	ctx := context.Background()
	type fields struct {
		Storage        Storage
		connectors     []Connector
		connectorsByID map[string]Connector
	}
	type args struct {
		ctx context.Context
		c   Connector
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "create dynamic",
			fields: fields{
				Storage: newMockStorage(),
			},
			args: args{
				ctx: ctx,
				c:   Connector{ID: "dynamic"},
			},
			wantErr: false,
		},
		{
			name: "create static",
			fields: fields{
				Storage:        newMockStorage(),
				connectorsByID: map[string]Connector{"static": {ID: "static"}},
			},
			args: args{
				ctx: ctx,
				c:   Connector{ID: "static"},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := staticConnectorsStorage{
				Storage:        tt.fields.Storage,
				connectors:     tt.fields.connectors,
				connectorsByID: tt.fields.connectorsByID,
			}
			err := s.CreateConnector(tt.args.ctx, tt.args.c)
			if tt.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), "read-only cannot create connector")
				return
			}
			require.NoError(t, err)
			got, err := tt.fields.Storage.GetConnector(ctx, tt.args.c.ID)
			require.NoError(t, err)
			require.Equal(t, tt.args.c, got)
		})
	}
}

func Test_staticConnectorsStorage_DeleteConnector(t *testing.T) {
	ctx := context.Background()
	type fields struct {
		Storage        Storage
		connectors     []Connector
		connectorsByID map[string]Connector
	}
	type args struct {
		ctx context.Context
		id  string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		setup   func(s Storage)
		wantErr bool
	}{
		{
			name: "delete dynamic",
			fields: fields{
				Storage: newMockStorage(),
			},
			args: args{
				ctx: ctx,
				id:  "dynamic",
			},
			setup: func(s Storage) {
				s.CreateConnector(ctx, Connector{ID: "dynamic"})
			},
			wantErr: false,
		},
		{
			name: "delete static",
			fields: fields{
				Storage:        newMockStorage(),
				connectorsByID: map[string]Connector{"static": {ID: "static"}},
			},
			args: args{
				ctx: ctx,
				id:  "static",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := staticConnectorsStorage{
				Storage:        tt.fields.Storage,
				connectors:     tt.fields.connectors,
				connectorsByID: tt.fields.connectorsByID,
			}
			if tt.setup != nil {
				tt.setup(s.Storage)
			}
			err := s.DeleteConnector(tt.args.ctx, tt.args.id)
			if tt.wantErr {
				require.Error(t, err)
				if tt.args.id != "missing" {
					require.Contains(t, err.Error(), "read-only cannot delete connector")
				}
				return
			}
			require.NoError(t, err)
			_, err = s.Storage.GetConnector(ctx, tt.args.id)
			require.Error(t, err)
		})
	}
}

func Test_staticConnectorsStorage_GetConnector(t *testing.T) {
	ctx := context.Background()
	type fields struct {
		Storage        Storage
		connectors     []Connector
		connectorsByID map[string]Connector
	}
	type args struct {
		ctx context.Context
		id  string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		setup   func(s Storage)
		want    Connector
		wantErr bool
	}{
		{
			name: "get static",
			fields: fields{
				Storage:        newMockStorage(),
				connectorsByID: map[string]Connector{"static": {ID: "static", Name: "static connector"}},
			},
			args: args{
				ctx: ctx,
				id:  "static",
			},
			want: Connector{ID: "static", Name: "static connector"},
		},
		{
			name: "get dynamic",
			fields: fields{
				Storage: newMockStorage(),
			},
			args: args{
				ctx: ctx,
				id:  "dynamic",
			},
			setup: func(s Storage) {
				s.CreateConnector(ctx, Connector{ID: "dynamic", Name: "dynamic connector"})
			},
			want: Connector{ID: "dynamic", Name: "dynamic connector"},
		},
		{
			name: "missing",
			fields: fields{
				Storage: newMockStorage(),
			},
			args: args{
				ctx: ctx,
				id:  "missing",
			},
			wantErr: true,
		},
		{
			name: "prefer static over dynamic",
			fields: fields{
				Storage:        newMockStorage(),
				connectorsByID: map[string]Connector{"overlap": {ID: "overlap", Name: "static overlap"}},
			},
			args: args{
				ctx: ctx,
				id:  "overlap",
			},
			setup: func(s Storage) {
				s.CreateConnector(ctx, Connector{ID: "overlap", Name: "dynamic overlap"})
			},
			want: Connector{ID: "overlap", Name: "static overlap"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := staticConnectorsStorage{
				Storage:        tt.fields.Storage,
				connectors:     tt.fields.connectors,
				connectorsByID: tt.fields.connectorsByID,
			}
			if tt.setup != nil {
				tt.setup(s.Storage)
			}
			got, err := s.GetConnector(tt.args.ctx, tt.args.id)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func Test_staticConnectorsStorage_ListConnectors(t *testing.T) {
	ctx := context.Background()
	type fields struct {
		Storage        Storage
		connectors     []Connector
		connectorsByID map[string]Connector
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		setup   func(s Storage)
		want    []Connector
		wantErr bool
	}{
		{
			name: "static only",
			fields: fields{
				Storage:        newMockStorage(),
				connectors:     []Connector{{ID: "static1"}, {ID: "static2"}},
				connectorsByID: map[string]Connector{"static1": {ID: "static1"}, "static2": {ID: "static2"}},
			},
			args: args{
				ctx: ctx,
			},
			want: []Connector{{ID: "static1"}, {ID: "static2"}},
		},
		{
			name: "dynamic only",
			fields: fields{
				Storage: newMockStorage(),
			},
			args: args{
				ctx: ctx,
			},
			setup: func(s Storage) {
				s.CreateConnector(ctx, Connector{ID: "dynamic1"})
				s.CreateConnector(ctx, Connector{ID: "dynamic2"})
			},
			want: []Connector{{ID: "dynamic1"}, {ID: "dynamic2"}},
		},
		{
			name: "mixed",
			fields: fields{
				Storage:        newMockStorage(),
				connectors:     []Connector{{ID: "static"}},
				connectorsByID: map[string]Connector{"static": {ID: "static"}},
			},
			args: args{
				ctx: ctx,
			},
			setup: func(s Storage) {
				s.CreateConnector(ctx, Connector{ID: "dynamic"})
			},
			want: []Connector{{ID: "dynamic"}, {ID: "static"}},
		},
		{
			name: "overlap prefers static",
			fields: fields{
				Storage:        newMockStorage(),
				connectors:     []Connector{{ID: "overlap", Name: "static"}},
				connectorsByID: map[string]Connector{"overlap": {ID: "overlap", Name: "static"}},
			},
			args: args{
				ctx: ctx,
			},
			setup: func(s Storage) {
				s.CreateConnector(ctx, Connector{ID: "overlap", Name: "dynamic"})
			},
			want: []Connector{{ID: "overlap", Name: "static"}},
		},
		{
			name: "base list error",
			fields: fields{
				Storage: newMockStorage(),
			},
			args: args{
				ctx: ctx,
			},
			setup: func(s Storage) {
				s.(*mockStorage).listErr = errors.New("list error")
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := staticConnectorsStorage{
				Storage:        tt.fields.Storage,
				connectors:     tt.fields.connectors,
				connectorsByID: tt.fields.connectorsByID,
			}
			if tt.setup != nil {
				tt.setup(s.Storage)
			}
			got, err := s.ListConnectors(tt.args.ctx)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			sort.Slice(got, func(i, j int) bool { return got[i].ID < got[j].ID })
			sort.Slice(tt.want, func(i, j int) bool { return tt.want[i].ID < tt.want[j].ID })
			require.Equal(t, tt.want, got)
		})
	}
}

func Test_staticConnectorsStorage_UpdateConnector(t *testing.T) {
	ctx := context.Background()
	type fields struct {
		Storage        Storage
		connectors     []Connector
		connectorsByID map[string]Connector
	}
	type args struct {
		ctx     context.Context
		id      string
		updater func(old Connector) (Connector, error)
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		setup   func(s Storage)
		wantErr bool
	}{
		{
			name: "update dynamic",
			fields: fields{
				Storage: newMockStorage(),
			},
			args: args{
				ctx: ctx,
				id:  "dynamic",
				updater: func(old Connector) (Connector, error) {
					old.Name = "new"
					return old, nil
				},
			},
			setup: func(s Storage) {
				s.CreateConnector(ctx, Connector{ID: "dynamic", Name: "old"})
			},
			wantErr: false,
		},
		{
			name: "update static",
			fields: fields{
				Storage:        newMockStorage(),
				connectorsByID: map[string]Connector{"static": {ID: "static"}},
			},
			args: args{
				ctx: ctx,
				id:  "static",
				updater: func(old Connector) (Connector, error) {
					old.Name = "new"
					return old, nil
				},
			},
			wantErr: true,
		},
		{
			name: "update missing",
			fields: fields{
				Storage: newMockStorage(),
			},
			args: args{
				ctx: ctx,
				id:  "missing",
				updater: func(old Connector) (Connector, error) {
					return old, nil
				},
			},
			wantErr: true,
		},
		{
			name: "base update error",
			fields: fields{
				Storage: newMockStorage(),
			},
			args: args{
				ctx: ctx,
				id:  "dynamic",
				updater: func(old Connector) (Connector, error) {
					return old, nil
				},
			},
			setup: func(s Storage) {
				s.(*mockStorage).updateErr = errors.New("update error")
				s.CreateConnector(ctx, Connector{ID: "dynamic"})
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := staticConnectorsStorage{
				Storage:        tt.fields.Storage,
				connectors:     tt.fields.connectors,
				connectorsByID: tt.fields.connectorsByID,
			}
			if tt.setup != nil {
				tt.setup(s.Storage)
			}
			err := s.UpdateConnector(tt.args.ctx, tt.args.id, tt.args.updater)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			got, err := s.Storage.GetConnector(ctx, tt.args.id)
			require.NoError(t, err)
			require.Equal(t, "new", got.Name)
		})
	}
}

func Test_staticConnectorsStorage_isStatic(t *testing.T) {
	type fields struct {
		Storage        Storage
		connectors     []Connector
		connectorsByID map[string]Connector
	}
	type args struct {
		id string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "static",
			fields: fields{
				connectorsByID: map[string]Connector{"static": {}},
			},
			args: args{
				id: "static",
			},
			want: true,
		},
		{
			name: "dynamic",
			fields: fields{
				connectorsByID: map[string]Connector{},
			},
			args: args{
				id: "dynamic",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := staticConnectorsStorage{
				Storage:        tt.fields.Storage,
				connectors:     tt.fields.connectors,
				connectorsByID: tt.fields.connectorsByID,
			}
			got := s.isStatic(tt.args.id)
			require.Equal(t, tt.want, got)
		})
	}
}

func Test_staticPasswordsStorage_CreatePassword(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()
	type fields struct {
		Storage          Storage
		passwords        []Password
		passwordsByEmail map[string]Password
		logger           *slog.Logger
	}
	type args struct {
		ctx context.Context
		p   Password
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "create dynamic",
			fields: fields{
				Storage: newMockStorage(),
				logger:  logger,
			},
			args: args{
				ctx: ctx,
				p:   Password{Email: "dynamic@example.com"},
			},
			wantErr: false,
		},
		{
			name: "create static",
			fields: fields{
				Storage:          newMockStorage(),
				passwordsByEmail: map[string]Password{"static@example.com": {Email: "static@example.com"}},
				logger:           logger,
			},
			args: args{
				ctx: ctx,
				p:   Password{Email: "static@example.com"},
			},
			wantErr: true,
		},
		{
			name: "create static case insensitive",
			fields: fields{
				Storage:          newMockStorage(),
				passwordsByEmail: map[string]Password{"static@example.com": {Email: "Static@Example.com"}},
				logger:           logger,
			},
			args: args{
				ctx: ctx,
				p:   Password{Email: "static@example.com"},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := staticPasswordsStorage{
				Storage:          tt.fields.Storage,
				passwords:        tt.fields.passwords,
				passwordsByEmail: tt.fields.passwordsByEmail,
				logger:           tt.fields.logger,
			}
			err := s.CreatePassword(tt.args.ctx, tt.args.p)
			if tt.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), "read-only cannot create password")
				return
			}
			require.NoError(t, err)
			got, err := tt.fields.Storage.GetPassword(ctx, tt.args.p.Email)
			require.NoError(t, err)
			require.Equal(t, tt.args.p, got)
		})
	}
}

func Test_staticPasswordsStorage_DeletePassword(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()
	type fields struct {
		Storage          Storage
		passwords        []Password
		passwordsByEmail map[string]Password
		logger           *slog.Logger
	}
	type args struct {
		ctx   context.Context
		email string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		setup   func(s Storage)
		wantErr bool
	}{
		{
			name: "delete dynamic",
			fields: fields{
				Storage: newMockStorage(),
				logger:  logger,
			},
			args: args{
				ctx:   ctx,
				email: "dynamic@example.com",
			},
			setup: func(s Storage) {
				s.CreatePassword(ctx, Password{Email: "dynamic@example.com"})
			},
			wantErr: false,
		},
		{
			name: "delete static",
			fields: fields{
				Storage:          newMockStorage(),
				passwordsByEmail: map[string]Password{"static@example.com": {Email: "static@example.com"}},
				logger:           logger,
			},
			args: args{
				ctx:   ctx,
				email: "static@example.com",
			},
			wantErr: true,
		},
		{
			name: "delete static case insensitive",
			fields: fields{
				Storage:          newMockStorage(),
				passwordsByEmail: map[string]Password{"static@example.com": {Email: "Static@Example.com"}},
				logger:           logger,
			},
			args: args{
				ctx:   ctx,
				email: "static@example.com",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := staticPasswordsStorage{
				Storage:          tt.fields.Storage,
				passwords:        tt.fields.passwords,
				passwordsByEmail: tt.fields.passwordsByEmail,
				logger:           tt.fields.logger,
			}
			if tt.setup != nil {
				tt.setup(s.Storage)
			}
			err := s.DeletePassword(tt.args.ctx, tt.args.email)
			if tt.wantErr {
				require.Error(t, err)
				if tt.args.email != "missing@example.com" {
					require.Contains(t, err.Error(), "read-only cannot delete password")
				}
				return
			}
			require.NoError(t, err)
			_, err = s.Storage.GetPassword(ctx, tt.args.email)
			require.Error(t, err)
		})
	}
}

func Test_staticPasswordsStorage_GetPassword(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()
	type fields struct {
		Storage          Storage
		passwords        []Password
		passwordsByEmail map[string]Password
		logger           *slog.Logger
	}
	type args struct {
		ctx   context.Context
		email string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		setup   func(s Storage)
		want    Password
		wantErr bool
	}{
		{
			name: "get static case insensitive",
			fields: fields{
				Storage:          newMockStorage(),
				passwordsByEmail: map[string]Password{"static@example.com": {Email: "Static@Example.com", Username: "static"}},
				logger:           logger,
			},
			args: args{
				ctx:   ctx,
				email: "static@example.com",
			},
			want: Password{Email: "Static@Example.com", Username: "static"},
		},
		{
			name: "get dynamic",
			fields: fields{
				Storage: newMockStorage(),
				logger:  logger,
			},
			args: args{
				ctx:   ctx,
				email: "dynamic@example.com",
			},
			setup: func(s Storage) {
				s.CreatePassword(ctx, Password{Email: "dynamic@example.com", Username: "dynamic"})
			},
			want: Password{Email: "dynamic@example.com", Username: "dynamic"},
		},
		{
			name: "missing",
			fields: fields{
				Storage: newMockStorage(),
				logger:  logger,
			},
			args: args{
				ctx:   ctx,
				email: "missing@example.com",
			},
			wantErr: true,
		},
		{
			name: "prefer static over dynamic",
			fields: fields{
				Storage:          newMockStorage(),
				passwordsByEmail: map[string]Password{"overlap@example.com": {Email: "overlap@example.com", Username: "static"}},
				logger:           logger,
			},
			args: args{
				ctx:   ctx,
				email: "overlap@example.com",
			},
			setup: func(s Storage) {
				s.CreatePassword(ctx, Password{Email: "overlap@example.com", Username: "dynamic"})
			},
			want: Password{Email: "overlap@example.com", Username: "static"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := staticPasswordsStorage{
				Storage:          tt.fields.Storage,
				passwords:        tt.fields.passwords,
				passwordsByEmail: tt.fields.passwordsByEmail,
				logger:           tt.fields.logger,
			}
			if tt.setup != nil {
				tt.setup(s.Storage)
			}
			got, err := s.GetPassword(tt.args.ctx, tt.args.email)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func Test_staticPasswordsStorage_ListPasswords(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()
	type fields struct {
		Storage          Storage
		passwords        []Password
		passwordsByEmail map[string]Password
		logger           *slog.Logger
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		setup   func(s Storage)
		want    []Password
		wantErr bool
	}{
		{
			name: "static only",
			fields: fields{
				Storage:          newMockStorage(),
				passwords:        []Password{{Email: "static1@example.com"}, {Email: "static2@example.com"}},
				passwordsByEmail: map[string]Password{"static1@example.com": {Email: "static1@example.com"}, "static2@example.com": {Email: "static2@example.com"}},
				logger:           logger,
			},
			args: args{
				ctx: ctx,
			},
			want: []Password{{Email: "static1@example.com"}, {Email: "static2@example.com"}},
		},
		{
			name: "dynamic only",
			fields: fields{
				Storage: newMockStorage(),
				logger:  logger,
			},
			args: args{
				ctx: ctx,
			},
			setup: func(s Storage) {
				s.CreatePassword(ctx, Password{Email: "dynamic1@example.com"})
				s.CreatePassword(ctx, Password{Email: "dynamic2@example.com"})
			},
			want: []Password{{Email: "dynamic1@example.com"}, {Email: "dynamic2@example.com"}},
		},
		{
			name: "mixed",
			fields: fields{
				Storage:          newMockStorage(),
				passwords:        []Password{{Email: "static@example.com"}},
				passwordsByEmail: map[string]Password{"static@example.com": {Email: "static@example.com"}},
				logger:           logger,
			},
			args: args{
				ctx: ctx,
			},
			setup: func(s Storage) {
				s.CreatePassword(ctx, Password{Email: "dynamic@example.com"})
			},
			want: []Password{{Email: "dynamic@example.com"}, {Email: "static@example.com"}},
		},
		{
			name: "overlap prefers static case insensitive",
			fields: fields{
				Storage:          newMockStorage(),
				passwords:        []Password{{Email: "Overlap@Example.com", Username: "static"}},
				passwordsByEmail: map[string]Password{"overlap@example.com": {Email: "Overlap@Example.com", Username: "static"}},
				logger:           logger,
			},
			args: args{
				ctx: ctx,
			},
			setup: func(s Storage) {
				s.CreatePassword(ctx, Password{Email: "overlap@example.com", Username: "dynamic"})
			},
			want: []Password{{Email: "Overlap@Example.com", Username: "static"}},
		},
		{
			name: "base list error",
			fields: fields{
				Storage: newMockStorage(),
				logger:  logger,
			},
			args: args{
				ctx: ctx,
			},
			setup: func(s Storage) {
				s.(*mockStorage).listErr = errors.New("list error")
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := staticPasswordsStorage{
				Storage:          tt.fields.Storage,
				passwords:        tt.fields.passwords,
				passwordsByEmail: tt.fields.passwordsByEmail,
				logger:           tt.fields.logger,
			}
			if tt.setup != nil {
				tt.setup(s.Storage)
			}
			got, err := s.ListPasswords(tt.args.ctx)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			sort.Slice(got, func(i, j int) bool { return strings.ToLower(got[i].Email) < strings.ToLower(got[j].Email) })
			sort.Slice(tt.want, func(i, j int) bool { return strings.ToLower(tt.want[i].Email) < strings.ToLower(tt.want[j].Email) })
			require.Equal(t, tt.want, got)
		})
	}
}

func Test_staticPasswordsStorage_UpdatePassword(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()
	type fields struct {
		Storage          Storage
		passwords        []Password
		passwordsByEmail map[string]Password
		logger           *slog.Logger
	}
	type args struct {
		ctx     context.Context
		email   string
		updater func(old Password) (Password, error)
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		setup   func(s Storage)
		wantErr bool
	}{
		{
			name: "update dynamic",
			fields: fields{
				Storage: newMockStorage(),
				logger:  logger,
			},
			args: args{
				ctx:   ctx,
				email: "dynamic@example.com",
				updater: func(old Password) (Password, error) {
					old.Username = "new"
					return old, nil
				},
			},
			setup: func(s Storage) {
				s.CreatePassword(ctx, Password{Email: "dynamic@example.com", Username: "old"})
			},
			wantErr: false,
		},
		{
			name: "update static",
			fields: fields{
				Storage:          newMockStorage(),
				passwordsByEmail: map[string]Password{"static@example.com": {Email: "static@example.com"}},
				logger:           logger,
			},
			args: args{
				ctx:   ctx,
				email: "static@example.com",
				updater: func(old Password) (Password, error) {
					old.Username = "new"
					return old, nil
				},
			},
			wantErr: true,
		},
		{
			name: "update static case insensitive",
			fields: fields{
				Storage:          newMockStorage(),
				passwordsByEmail: map[string]Password{"static@example.com": {Email: "Static@Example.com"}},
				logger:           logger,
			},
			args: args{
				ctx:   ctx,
				email: "static@example.com",
				updater: func(old Password) (Password, error) {
					old.Username = "new"
					return old, nil
				},
			},
			wantErr: true,
		},
		{
			name: "update missing",
			fields: fields{
				Storage: newMockStorage(),
				logger:  logger,
			},
			args: args{
				ctx:   ctx,
				email: "missing@example.com",
				updater: func(old Password) (Password, error) {
					return old, nil
				},
			},
			wantErr: true,
		},
		{
			name: "base update error",
			fields: fields{
				Storage: newMockStorage(),
				logger:  logger,
			},
			args: args{
				ctx:   ctx,
				email: "dynamic@example.com",
				updater: func(old Password) (Password, error) {
					return old, nil
				},
			},
			setup: func(s Storage) {
				s.(*mockStorage).updateErr = errors.New("update error")
				s.CreatePassword(ctx, Password{Email: "dynamic@example.com"})
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := staticPasswordsStorage{
				Storage:          tt.fields.Storage,
				passwords:        tt.fields.passwords,
				passwordsByEmail: tt.fields.passwordsByEmail,
				logger:           tt.fields.logger,
			}
			if tt.setup != nil {
				tt.setup(s.Storage)
			}
			err := s.UpdatePassword(tt.args.ctx, tt.args.email, tt.args.updater)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			got, err := s.Storage.GetPassword(ctx, tt.args.email)
			require.NoError(t, err)
			require.Equal(t, "new", got.Username)
		})
	}
}

func Test_staticPasswordsStorage_isStatic(t *testing.T) {
	type fields struct {
		Storage          Storage
		passwords        []Password
		passwordsByEmail map[string]Password
		logger           *slog.Logger
	}
	type args struct {
		email string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "static",
			fields: fields{
				passwordsByEmail: map[string]Password{"static@example.com": {}},
			},
			args: args{
				email: "Static@Example.com",
			},
			want: true,
		},
		{
			name: "dynamic",
			fields: fields{
				passwordsByEmail: map[string]Password{},
			},
			args: args{
				email: "dynamic@example.com",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := staticPasswordsStorage{
				Storage:          tt.fields.Storage,
				passwords:        tt.fields.passwords,
				passwordsByEmail: tt.fields.passwordsByEmail,
				logger:           tt.fields.logger,
			}
			got := s.isStatic(tt.args.email)
			require.Equal(t, tt.want, got)
		})
	}
}
