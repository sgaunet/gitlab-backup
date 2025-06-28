// Package gitlab provides GitLab API client functionality.
package gitlab

import (
	"encoding/json"
	"errors"
	"fmt"
)

var (
	// ErrUnmarshalJSON is returned when JSON unmarshalling fails.
	ErrUnmarshalJSON = errors.New("error unmarshalling json")
	// ErrGitlabAPI is returned when GitLab API returns an error.
	ErrGitlabAPI     = errors.New("error message from Gitlab API")
)

// ErrorMessage is a struct that contains an error message.
// It is returned by the Gitlab API when an error occurs.
type ErrorMessage struct {
	Message string `json:"message"`
}

// UnmarshalErrorMessage unmarshals the error message from the Gitlab API.
func UnmarshalErrorMessage(body []byte) error {
	var errMsg ErrorMessage
	if err := json.Unmarshal(body, &errMsg); err != nil {
		return fmt.Errorf("%w: %w - %s", ErrUnmarshalJSON, err, body)
	}
	return fmt.Errorf("%w: %s", ErrGitlabAPI, errMsg.Message)
}
