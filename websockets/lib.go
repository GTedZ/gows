package websockets

import (
	"crypto/rand"
	"fmt"

	jsoniter "github.com/json-iterator/go"
)

func generateRandomString() string {
	b := make([]byte, 32)
	_, err := rand.Read(b[:])
	if err != nil {
		panic(fmt.Errorf("failed to generate secure random int64: %w", err))
	}
	return fmt.Sprintf("%x", b)
}

func CheckMessageIsPrivate(msg []byte, privateMessagePropertyName string) (requestId string, isPrivate bool) {
	if len(msg) == 0 {
		return "", false
	}
	if msg[0] == '[' {
		return "", false
	}

	var parsedMessage map[string]interface{}
	err := jsoniter.Unmarshal(msg, &parsedMessage)
	if err != nil {
		Logger.ERROR(fmt.Sprintf("failed to unmarshall the following message => %s", msg), err)
		return "", false
	}

	propertyInterface, exists := parsedMessage[privateMessagePropertyName]
	if !exists {
		return "", false
	}

	propertyValue, ok := propertyInterface.(string)
	if !ok {
		return "", false
	}

	if propertyValue == "" {
		return "", false
	}

	return propertyValue, true
}
