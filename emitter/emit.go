package emitter

import (
	"code.google.com/p/gogoprotobuf/proto"
	"fmt"
	"github.com/cloudfoundry/gosteno"
	"github.com/cloudfoundry/loggregatorlib/loggregatorclient"
	"github.com/cloudfoundry/loggregatorlib/logmessage"
	//	"regexp"
	"strings"
	"time"
)

var (
	MAX_MESSAGE_BYTE_SIZE = (9 * 1024) - 512
	TRUNCATED_BYTES       = []byte("TRUNCATED")
	TRUNCATED_OFFSET      = MAX_MESSAGE_BYTE_SIZE - len(TRUNCATED_BYTES)
)

type Emitter interface {
	Emit(string, string)
	EmitLogMessage(*logmessage.LogMessage)
}

type loggregatoremitter struct {
	LoggregatorClient loggregatorclient.LoggregatorClient
	st                logmessage.LogMessage_SourceType
	sId               string
	sharedSecret      string
	logger            *gosteno.Logger
}

func isEmpty(s string) bool {
	return len(strings.TrimSpace(s)) == 0
}

func splitMessage(message string) []string {
	return strings.FieldsFunc(message, func(r rune) bool {
		return r == '\n' || r == '\r'
	})
}

func (e *loggregatoremitter) Emit(appid, message string) {
	if isEmpty(appid) || isEmpty(message) {
		return
	}
	logMessage := e.newLogMessage(appid, message)
	e.logger.Debugf("Logging message from %s of type %s with appid %s and with data %s", logMessage.SourceType, logMessage.MessageType, logMessage.AppId, string(logMessage.Message))

	e.EmitLogMessage(logMessage)
}

func (e *loggregatoremitter) EmitLogMessage(logMessage *logmessage.LogMessage) {
	messages := splitMessage(string(logMessage.GetMessage()))

	for _, message := range messages {
		if isEmpty(message) {
			continue
		}

		message := logMessage.Message
		if len(message) > MAX_MESSAGE_BYTE_SIZE {
			logMessage.Message = append([]byte(message)[0:TRUNCATED_OFFSET], TRUNCATED_BYTES...)
		} else {
			logMessage.Message = []byte(message)
		}

		if e.sharedSecret == "" {
			marshalledLogMessage, err := proto.Marshal(logMessage)
			if err != nil {
				e.logger.Errorf("Error marshalling message: %s", err)
				return
			}
			e.logger.Debugf("Sent LogMessage: %s", logMessage.String())
			e.LoggregatorClient.Send(marshalledLogMessage)
		} else {
			logEnvelope := e.newLogEnvelope(*logMessage.AppId, logMessage)
			marshalledLogEnvelope, err := proto.Marshal(logEnvelope)
			if err != nil {
				e.logger.Errorf("Error marshalling envelope: %s", err)
				return
			}
			e.logger.Debugf("Sent LogEnvelope: %s", logEnvelope.String())
			e.LoggregatorClient.Send(marshalledLogEnvelope)
		}
	}
}

func NewEmitter(loggregatorServer, sourceType, sourceId string, logger *gosteno.Logger) (e *loggregatoremitter, err error) {
	return NewLogEnvelopeEmitter(loggregatorServer, sourceType, sourceId, "", logger)
}

func NewLogMessageEmitter(loggregatorServer, sourceType, sourceId string, logger *gosteno.Logger) (e *loggregatoremitter, err error) {
	return NewLogEnvelopeEmitter(loggregatorServer, sourceType, sourceId, "", logger)
}

func NewLogEnvelopeEmitter(loggregatorServer, sourceType, sourceId, sharedSecret string, logger *gosteno.Logger) (e *loggregatoremitter, err error) {
	if logger == nil {
		logger = gosteno.NewLogger("loggregatorlib.emitter")
	}

	e = &loggregatoremitter{sharedSecret: sharedSecret}

	if name, ok := logmessage.LogMessage_SourceType_value[sourceType]; ok {
		e.st = logmessage.LogMessage_SourceType(name)
	} else {

		err = fmt.Errorf("Unable to map SourceType [%s] to a logmessage.LogMessage_SourceType", sourceType)
		return
	}

	e.logger = logger
	e.LoggregatorClient = loggregatorclient.NewLoggregatorClient(loggregatorServer, logger, loggregatorclient.DefaultBufferSize)
	e.sId = sourceId

	e.logger.Debugf("Created new loggregator emitter: %#v", e)
	return
}

func (e *loggregatoremitter) newLogMessage(appId, message string) *logmessage.LogMessage {
	currentTime := time.Now()
	mt := logmessage.LogMessage_OUT

	return &logmessage.LogMessage{
		Message:     []byte(message),
		AppId:       proto.String(appId),
		MessageType: &mt,
		SourceType:  &e.st,
		SourceId:    &e.sId,
		Timestamp:   proto.Int64(currentTime.UnixNano()),
	}
}

func (e *loggregatoremitter) newLogEnvelope(appId string, message *logmessage.LogMessage) *logmessage.LogEnvelope {
	envelope := &logmessage.LogEnvelope{
		LogMessage: message,
		RoutingKey: proto.String(appId),
		Signature:  []byte{},
	}
	envelope.SignEnvelope(e.sharedSecret)

	return envelope
}
