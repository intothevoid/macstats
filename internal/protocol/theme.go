package protocol

import (
	"errors"

	"macstats/internal/collector"
)

const themeHeaderLength = 256

type Theme struct {
	Raw          []byte
	Header       []byte
	Payload      []byte
	HeaderLength int
}

func ParseTheme(data []byte) (Theme, error) {
	if len(data) <= themeHeaderLength {
		return Theme{}, errors.New("theme buffer too short")
	}

	return Theme{
		Raw:          data,
		Header:       data[:themeHeaderLength],
		Payload:      data[themeHeaderLength:],
		HeaderLength: themeHeaderLength,
	}, nil
}

func (t Theme) StartupPayload() []byte {
	return t.Raw
}

func (t Theme) RefreshPayload(collector.Metrics) []byte {
	return t.Raw
}
