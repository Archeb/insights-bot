package autorecap

import (
	"context"
	"fmt"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	cron "github.com/robfig/cron/v3"
	"github.com/samber/lo"
	"go.uber.org/fx"

	"github.com/nekomeowww/insights-bot/internal/models/chathistories"
	"github.com/nekomeowww/insights-bot/internal/models/tgchats"
	"github.com/nekomeowww/insights-bot/pkg/bots/tgbot"
	"github.com/nekomeowww/insights-bot/pkg/healthchecker"
	"github.com/nekomeowww/insights-bot/pkg/logger"
	"github.com/nekomeowww/insights-bot/pkg/openai"
)

type NewAutoRecapServiceParam struct {
	fx.In

	Lifecycle fx.Lifecycle

	Bot           *tgbot.BotService
	Logger        *logger.Logger
	ChatHistories *chathistories.Model
	TgChats       *tgchats.Model
	OpenAI        openai.Client
}

var _ healthchecker.HealthChecker = (*AutoRecapService)(nil)

type AutoRecapService struct {
	Cron    *cron.Cron
	started bool

	bot           *tgbot.BotService
	logger        *logger.Logger
	chatHistories *chathistories.Model
	tgchats       *tgchats.Model
	openai        openai.Client
}

func NewAutoRecapService() func(NewAutoRecapServiceParam) (*AutoRecapService, error) {
	return func(param NewAutoRecapServiceParam) (*AutoRecapService, error) {
		service := &AutoRecapService{
			Cron:          cron.New(),
			bot:           param.Bot,
			logger:        param.Logger,
			chatHistories: param.ChatHistories,
			tgchats:       param.TgChats,
			openai:        param.OpenAI,
		}

		_, err := service.Cron.AddFunc("@every 6h", service.SendChatHistoriesRecap)
		if err != nil {
			return nil, err
		}

		param.Lifecycle.Append(fx.Hook{
			OnStop: func(context.Context) error {
				service.Cron.Stop()
				return nil
			},
		})

		service.logger.Infof("chat history recap service started")

		return service, nil
	}
}

func (s *AutoRecapService) Check(ctx context.Context) error {
	return lo.Ternary(s.started, nil, fmt.Errorf("auto recap not started yet"))
}

func Run() func(service *AutoRecapService) {
	return func(service *AutoRecapService) {
		service.Cron.Start()
		service.started = true
	}
}

func (s *AutoRecapService) SendChatHistoriesRecap() {
	chatIDs, err := s.tgchats.ListChatHistoriesRecapEnabledChats()
	if err != nil {
		s.logger.Errorf("failed to list chat histories recap enabled chats: %v", err)
		return
	}

	for _, chatID := range chatIDs {
		s.logger.Infof("generating chat histories recap for chat %d", chatID)

		histories, err := s.chatHistories.FindLastSixHourChatHistories(chatID)
		if err != nil {
			s.logger.Errorf("failed to find last six hour chat histories: %v", err)
			continue
		}
		if len(histories) <= 5 {
			s.logger.Warn("no enough chat histories")
			continue
		}

		summarizations, err := s.chatHistories.SummarizeChatHistories(chatID, histories)
		if err != nil {
			s.logger.Errorf("failed to summarize last six hour chat histories: %v", err)
			continue
		}

		summarizations = lo.Filter(summarizations, func(item string, _ int) bool { return item != "" })
		if len(summarizations) == 0 {
			s.logger.Warn("summarization is empty")
			continue
		}

		for i, s := range summarizations {
			summarizations[i] = tgbot.ReplaceMarkdownTitlesToTelegramBoldElement(s)
		}

		summarizationBatches := tgbot.SplitMessagesAgainstLengthLimitIntoMessageGroups(summarizations)
		for i, b := range summarizationBatches {
			var content string
			if len(summarizationBatches) > 1 {
				content = fmt.Sprintf("%s\n\n(%d/%d)\n#recap #recap_auto\n<em>🤖️ Generated by chatGPT</em>", strings.Join(b, "\n\n"), i+1, len(summarizationBatches))
			} else {
				content = fmt.Sprintf("%s\n\n#recap #recap_auto\n<em>🤖️ Generated by chatGPT</em>", strings.Join(b, "\n\n"))
			}

			msg := tgbotapi.NewMessage(chatID, content)
			msg.ParseMode = tgbotapi.ModeHTML

			s.logger.Infof("sending chat histories recap for chat %d: %s", chatID, msg.Text)

			_, err = s.bot.Send(msg)
			if err != nil {
				s.logger.Errorf("failed to send chat histories recap: %v", err)
			}
		}
	}
}
