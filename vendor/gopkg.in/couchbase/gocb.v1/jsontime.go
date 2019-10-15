package gocb

import (
	"encoding/json"
	"time"
)

type jsonMillisecondDuration time.Duration

func (d *jsonMillisecondDuration) UnmarshalJSON(data []byte) error {
	var milliseconds int64
	err := json.Unmarshal(data, &milliseconds)
	if err != nil {
		return err
	}
	*d = jsonMillisecondDuration(time.Duration(milliseconds) * time.Millisecond)
	return nil
}

func (d jsonMillisecondDuration) MarshalJSON() ([]byte, error) {
	var milliseconds = int64(time.Duration(d) / time.Millisecond)
	return json.Marshal(milliseconds)
}
