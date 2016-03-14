package simplehttp

import (
	"encoding/json"
	"net/http"
)

func GetJSONInput(r *http.Request, w http.ResponseWriter, v interface{}) (err error) {
	decoder := json.NewDecoder(r.Body)
	err = decoder.Decode(v)
	if err != nil {
		http.Error(w, "Bad request.", http.StatusBadRequest)
		return err
	}

	return nil
}

func OutputJSON(w http.ResponseWriter, v interface{}) (err error) {
	var data []byte
	data, err = json.Marshal(v)
	if err != nil {
		http.Error(w, "Internal error.", http.StatusInternalServerError)
		return err
	}

	w.Header().Add("Content-Type", "application/json")
	_, err = w.Write(data)

	return nil
}
