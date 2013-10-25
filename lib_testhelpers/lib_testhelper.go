package lib_testhelpers

import (
	"code.google.com/p/gogoprotobuf/proto"
	"github.com/cloudfoundry/loggregatorlib/logmessage"
	"github.com/cloudfoundry/loggregatorlib/signature"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func MarshalledLogMessage(t *testing.T, messageString string, appId string) []byte {
	message := NewLogMessage(t, messageString, appId)

	marshalledMessage, err := proto.Marshal(message)
	assert.NoError(t, err)

	return marshalledMessage
}

func MarshalledLogEnvelope(t *testing.T, messageString string, appId string, secret string) []byte {
	message := NewLogMessage(t, messageString, appId)

	signatureOfMessage, err := signature.Encrypt(secret, signature.Digest(message.String()))
	assert.NoError(t, err)

	envelope := &logmessage.LogEnvelope{
		LogMessage: message,
		RoutingKey: proto.String(appId),
		Signature:  signatureOfMessage,
	}

	marshalledEnvelope, err := proto.Marshal(envelope)
	assert.NoError(t, err)

	return marshalledEnvelope
}

func NewLogMessage(t *testing.T, messageString string, appId string) *logmessage.LogMessage {
	currentTime := time.Now()

	messageType := logmessage.LogMessage_OUT
	sourceType := logmessage.LogMessage_DEA
	return &logmessage.LogMessage{
		Message:     []byte(messageString),
		AppId:       proto.String(appId),
		MessageType: &messageType,
		SourceType:  &sourceType,
		Timestamp:   proto.Int64(currentTime.UnixNano()),
	}
}