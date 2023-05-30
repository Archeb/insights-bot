package tgbot

import (
	"encoding/json"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/nekomeowww/insights-bot/pkg/logger"
	"github.com/redis/rueidis"
	"github.com/samber/lo"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBindFromCallbackQueryData(t *testing.T) {
	logger := logger.NewLogger(logrus.DebugLevel, "insights-bot", "", nil)

	c, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{"localhost:6379"},
		DisableCache: true,
	})
	require.NoError(t, err)

	data := struct {
		Hello string `json:"hello"`
	}{
		Hello: "world",
	}

	ctx := NewContext(nil, tgbotapi.Update{}, logger, c)
	ctx.isCallbackQuery = true

	ctx.callbackQueryHandlerActionData = string(lo.Must(json.Marshal(data)))

	var dst struct {
		Hello string `json:"hello"`
	}

	err = ctx.BindFromCallbackQueryData(&dst)
	require.NoError(t, err)
	assert.Equal(t, data, dst)
}
