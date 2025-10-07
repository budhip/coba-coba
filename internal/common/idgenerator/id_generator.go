// Package idgenerator provides functionality for generating unique IDs
// that are composed of a prefix, a timestamp, and a base64-encoded UUID.
// This package uses the github.com/google/uuid library to generate UUIDs and
// the encoding/base64 library to encode them in base64 format.
package idgenerator

import (
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Generator interface {
	Generate(prefixes ...string) string
}

type IDGenerator struct{}

func New() Generator {
	return &IDGenerator{}
}

// New function generates a unique ID by combining a prefix string, a timestamp, and
// a base64-encoded UUID.
// If the prefix parameter is empty, the function returns the generated ID without prefix. Otherwise,
// it returns string that represents the generated ID.
func (g *IDGenerator) Generate(prefixes ...string) string {
	prefix := strings.Join(prefixes, "-")
	epocTime := getEPOCTime()
	generatedUUID := getUUID()
	encodedUUID := rawURLEncodedUUID(generatedUUID)
	generatedID := fmt.Sprintf("%s-%d%s", prefix, epocTime, encodedUUID)

	if len(prefixes) == 0 || prefix == "" {
		generatedID = fmt.Sprintf("%d%s", epocTime, encodedUUID)
	}

	return generatedID
}

func getEPOCTime() int64 {
	return time.Now().UnixMilli()
}

func getUUID() uuid.UUID {
	return uuid.New()
}

func rawURLEncodedUUID(id uuid.UUID) string {
	uuidInBytes := id[:]
	return base64.RawURLEncoding.EncodeToString(uuidInBytes)
}
