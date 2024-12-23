package client

import (
	"github.com/pkg/errors"
	"time"
)

type QuayDate struct {
	time.Time
}

// UnmarshalJSON implements the json.Unmarshaler interface
func (qd *QuayDate) UnmarshalJSON(data []byte) error {
	str := string(data)

	if len(str) < 3 {
		return errors.New("empty quay date")
	}

	str = str[1 : len(str)-1]

	t, err := time.Parse("Mon, 02 Jan 2006 15:04:05 -0700", str)
	if err != nil {
		return err
	}

	qd.Time = t
	return nil
}

type Release struct {
	Tag          string   `json:"name,omitempty"`
	LastModified QuayDate `json:"last_modified,omitempty"`
}

type QuayClient interface {
	Releases(container string) ([]Release, error)
}
