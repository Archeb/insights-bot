package smr

import (
	"github.com/nekomeowww/insights-bot/internal/configs"
	"github.com/nekomeowww/insights-bot/internal/datastore"
	"github.com/nekomeowww/insights-bot/internal/lib"
	"github.com/nekomeowww/insights-bot/pkg/tutils"
	"testing"

	"github.com/nekomeowww/insights-bot/internal/models/smr"
	"github.com/stretchr/testify/assert"
)

var testService *Service

func TestMain(m *testing.M) {
	config := configs.NewTestConfig()()
	redis, _ := datastore.NewRedis()(datastore.NewRedisParams{
		Configs: config,
	})
	testService, _ = NewService()(NewServiceParam{
		Config: config,
		Model: smr.NewModel()(smr.NewModelParams{
			Logger: lib.NewLogger()(lib.NewLoggerParams{
				Configs: config,
			}),
		}),
		Logger: lib.NewLogger()(lib.NewLoggerParams{
			Configs: config,
		}),
		RedisClient: redis,
		LifeCycle:   tutils.NewEmtpyLifecycle(),
	})

	m.Run()
}

func TestService_botExists(t *testing.T) {
	t.Run("BotNotExists", func(t *testing.T) {
		a := assert.New(t)
		a.False(testService.isBotExists(smr.FromPlatformDiscord))
		a.False(testService.isBotExists(smr.FromPlatformSlack))
		a.False(testService.isBotExists(smr.FromPlatformTelegram))
	})
}
