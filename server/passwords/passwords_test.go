package passwords

import "testing"

func TestCheckCost(t *testing.T) {
	tests := []struct {
		name      string
		inputHash []byte
		wantErr   bool
	}{
		{
			name: "valid cost",
			// bcrypt hash of the value "test1" with cost 12 (default)
			inputHash: []byte("$2a$12$M2Ot95Qty1MuQdubh1acWOiYadJDzeVg3ve4n5b.dgcgPdjCseKx2"),
		},
		{
			name:      "invalid hash",
			inputHash: []byte(""),
			wantErr:   true,
		},
		{
			name: "cost below default",
			// bcrypt hash of the value "test1" with cost 4
			inputHash: []byte("$2a$04$8bSTbuVCLpKzaqB3BmgI7edDigG5tIQKkjYUu/mEO9gQgIkw9m7eG"),
			wantErr:   true,
		},
		{
			name: "cost above recommendation",
			// bcrypt hash of the value "test1" with cost 17
			inputHash: []byte("$2a$17$tWuZkTxtSmRyWZAGWVHQE.7npdl.TgP8adjzLJD.SyjpFznKBftPe"),
			wantErr:   true,
		},
	}

	for _, tc := range tests {
		err := CheckCost(tc.inputHash)
		if tc.wantErr {
			if err == nil {
				t.Errorf("%s: expected err", tc.name)
			}
			continue
		}
		if err != nil {
			t.Errorf("%s: %s", tc.name, err)
		}
	}
}
