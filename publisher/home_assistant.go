package publisher

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/krzysztof-gzocha/prometheus2mqtt/config"
)

type haConfigMessage struct {
	Name       string `json:"name"`
	StateTopic string `json:"state_topic"`
}

type HomeAssistant struct {
	cfg               config.Mqtt
	mqtt              mqtt.Client
	logger            *log.Logger
	alreadyConfigured map[string]interface{}
}

func NewHomeAssistant(
	cfg config.Mqtt,
	mqtt mqtt.Client,
	logger *log.Logger,
) *HomeAssistant {
	return &HomeAssistant{
		cfg:               cfg,
		mqtt:              mqtt,
		logger:            logger,
		alreadyConfigured: make(map[string]interface{}),
	}
}

func (h *HomeAssistant) Publish(ctx context.Context, name, value string) error {
	if !h.isConfigured(name) {
		err := h.configure(ctx, name)
		if err != nil {
			return fmt.Errorf("could not send configuration message for metric %s: %s", name, err.Error())
		}
	}

	h.logger.Printf("Sending \t%s\t to \t%s\n", value, h.stateTopic(name))

	return h.sendMsg(ctx, h.stateTopic(name), value)
}

func (h *HomeAssistant) isConfigured(name string) bool {
	_, exists := h.alreadyConfigured[name]

	return exists
}

func (h *HomeAssistant) configure(ctx context.Context, name string) error {
	h.logger.Printf("Configuring sensor: %s\n", h.cfg.ClientID+": "+name)
	haCfg := haConfigMessage{
		Name:       h.cfg.ClientID + ": " + name,
		StateTopic: h.stateTopic(name),
	}

	j, err := json.Marshal(&haCfg)
	if err != nil {
		return err
	}

	h.logger.Printf(
		"Configuring device on topic %s with payload %s\n",
		h.configTopic(name),
		string(j),
	)

	err = h.sendMsg(ctx, h.configTopic(name), string(j))
	if err != nil {
		return err
	}

	h.alreadyConfigured[name] = struct{}{}

	return nil
}

func (h *HomeAssistant) stateTopic(name string) string {
	return fmt.Sprintf(
		"%s/sensor/%s/state",
		h.cfg.DiscoveryPrefix,
		h.stripSlashes(h.cfg.ClientID+"_"+name),
	)
}

func (h *HomeAssistant) configTopic(name string) string {
	return fmt.Sprintf(
		"%s/sensor/%s/config",
		h.cfg.DiscoveryPrefix,
		h.stripSlashes(h.cfg.ClientID+"_"+name),
	)
}

func (h *HomeAssistant) stripSlashes(name string) string {
	return strings.Replace(name, "/", "_", -1)
}

func (h *HomeAssistant) sendMsg(ctx context.Context, topic, msg string) error {
	token := h.mqtt.Publish(
		topic,
		h.cfg.Qos,
		h.cfg.RetainMessages,
		msg,
	)

	ctx, cancel := context.WithTimeout(ctx, h.cfg.PublishTimeout)
	defer cancel()

	select {
	case <-token.Done():
		return token.Error()
	case <-ctx.Done():
		return fmt.Errorf("publishing exceeded timeout: %w", token.Error())
	}
}
