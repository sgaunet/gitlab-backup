package gitlab

import (
	"encoding/json"
	"fmt"
)

// ErrorMessage is a struct that contains an error message
// It is returned by the Gitlab API when an error occurs
type ErrorMessage struct {
	Message string `json:"message"`
}

// UnmarshalErrorMessage unmarshals the error message from the Gitlab API
func UnmarshalErrorMessage(body []byte) error {
	var errMsg ErrorMessage
	if err := json.Unmarshal(body, &errMsg); err != nil {
		return fmt.Errorf("error unmarshalling json: %s - %s", err.Error(), body)
	}
	return fmt.Errorf("error message from Gitlab API: %s", errMsg.Message)
}
