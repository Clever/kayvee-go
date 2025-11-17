package logger

import (
	"bytes"
	"context"
	"testing"

	"github.com/Clever/kayvee-go/v7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFromContext(t *testing.T) {
	loggerFromEmptyContext := FromContext(context.Background())
	assert.NotNil(t, loggerFromEmptyContext, "FromContext should return a non-nil logger, even if the context does not contain a logger")

	logger := New("logger-from-context")
	ctx := context.Background()
	ctx = NewContext(ctx, logger)
	loggerFromContext := FromContext(ctx)
	assert.Equal(t, logger, loggerFromContext, "Logger retrieved from context should be the same one we placed in the context")
}

func TestFromContextWithName(t *testing.T) {
	r := require.New(t)

	buf := &bytes.Buffer{}
	logger := New("logger-from-context")
	logger.SetOutput(buf)
	ctx := context.WithValue(context.Background(), ContextName, "some-unit-of-work")
	ctx = NewContext(ctx, logger)

	loggerFromContext := FromContext(ctx)
	r.Equal(
		logger,
		loggerFromContext,
		"Logger retrieved from context should be the same one we placed in the context",
	)

	logger.Info("something happened")
	assertLogFormatAndCompareContent(
		t,
		buf.String(),
		kayvee.FormatLog(
			"logger-from-context",
			kayvee.Info,
			"something happened",
			M{"context_name": "some-unit-of-work"},
		),
	)
}
