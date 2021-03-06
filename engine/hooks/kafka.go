package hooks

import (
	"context"
  "crypto/tls"
  "encoding/json"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/Shopify/sarama"
	"github.com/fsamin/go-dump"
	cluster "gopkg.in/bsm/sarama-cluster.v2"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

var nbKafkaConsumers int64

func (s *Service) saveKafkaExecution(t *sdk.Task, error string, nbError int64) {
	exec := &sdk.TaskExecution{
		Timestamp: time.Now().UnixNano(),
		Type:      t.Type,
		UUID:      t.UUID,
		Config:    t.Config,
		Status:    TaskExecutionDone,
		LastError: error,
		NbErrors:  nbError,
	}
	s.Dao.SaveTaskExecution(exec)
}

func (s *Service) startKafkaHook(ctx context.Context, t *sdk.Task) error {
	var kafkaIntegration, kafkaUser, projectKey, topic string
	for k, v := range t.Config {
		switch k {
		case sdk.HookModelIntegration:
			kafkaIntegration = v.Value
		case sdk.KafkaHookModelTopic:
			topic = v.Value
		case sdk.HookConfigProject:
			projectKey = v.Value
		}
	}
	pf, err := s.Client.ProjectIntegrationGet(projectKey, kafkaIntegration, true)
	if err != nil {
		_ = s.stopTask(ctx, t)
		return sdk.WrapError(err, "Cannot get kafka configuration for %s/%s", projectKey, kafkaIntegration)
	}

	var password, broker string
	for k, v := range pf.Config {
		switch k {
		case "password":
			password = v.Value
		case "broker url":
			broker = v.Value
		case "username":
			kafkaUser = v.Value
		}

	}

	var config = sarama.NewConfig()
	config.Net.TLS.Enable = true
	config.Net.SASL.Enable = true
	config.Net.SASL.User = kafkaUser
	config.Net.SASL.Password = password
	config.ClientID = kafkaUser

	//Check for Azure EventHubs
	if kafkaUser == "$ConnectionString" {
		config := sarama.NewConfig()
		config.Net.DialTimeout = 10 * time.Second

		config.Net.SASL.Enable = true
		config.Net.SASL.User = "$ConnectionString"
		config.Net.SASL.Password = password
		config.Net.SASL.Mechanism = "PLAIN"

		config.Net.TLS.Enable = true
		config.Net.TLS.Config = &tls.Config{
		InsecureSkipVerify: true,
		ClientAuth:         0,
		}

		config.ClientID = "unknown"
	}

	config.Version = sarama.V1_0_0_0
	config.Producer.Return.Successes = true

	clusterConfig := cluster.NewConfig()
	clusterConfig.Config = *config
	clusterConfig.Consumer.Return.Errors = true

	var consumerGroup = fmt.Sprintf("%s.%s", kafkaUser, t.UUID)
	var errConsumer error
	consumer, errConsumer := cluster.NewConsumer(
		strings.Split(broker, ","),
		consumerGroup,
		[]string{topic},
		clusterConfig)

	if errConsumer != nil {
		_ = s.stopTask(ctx, t)
		return fmt.Errorf("startKafkaHook>Error creating consumer: (%s %s %s %s): %v", broker, consumerGroup, topic, kafkaUser, errConsumer)
	}

	// Consume errors
	go func() {
		for err := range consumer.Errors() {
			s.saveKafkaExecution(t, err.Error(), 1)
		}
	}()

	// consume message
	go func() {
		atomic.AddInt64(&nbKafkaConsumers, 1)
		defer atomic.AddInt64(&nbKafkaConsumers, -1)
		for msg := range consumer.Messages() {
			exec := sdk.TaskExecution{
				Status:    TaskExecutionScheduled,
				Config:    t.Config,
				Type:      TypeKafka,
				UUID:      t.UUID,
				Timestamp: time.Now().UnixNano(),
				Kafka:     &sdk.KafkaTaskExecution{Message: msg.Value},
			}
			s.Dao.SaveTaskExecution(&exec)
			consumer.MarkOffset(msg, "delivered")
		}
	}()

	return nil
}

func (s *Service) doKafkaTaskExecution(t *sdk.TaskExecution) (*sdk.WorkflowNodeRunHookEvent, error) {
	log.Debug("Hooks> Processing kafka %s %s", t.UUID, t.Type)

	// Prepare a struct to send to CDS API
	h := sdk.WorkflowNodeRunHookEvent{
		WorkflowNodeHookUUID: t.UUID,
		Payload:              map[string]string{},
	}

	var bodyJSON interface{}

	//Try to parse the body as an array
	bodyJSONArray := []interface{}{}
	if err := json.Unmarshal(t.Kafka.Message, &bodyJSONArray); err != nil {
		//Try to parse the body as a map
		bodyJSONMap := map[string]interface{}{}
		if err2 := json.Unmarshal(t.Kafka.Message, &bodyJSONMap); err2 == nil {
			bodyJSON = bodyJSONMap
		}
	} else {
		bodyJSON = bodyJSONArray
	}

	//Go Dump
	e := dump.NewDefaultEncoder()
	e.Formatters = []dump.KeyFormatterFunc{dump.WithDefaultLowerCaseFormatter()}
	e.ExtraFields.DetailedMap = false
	e.ExtraFields.DetailedStruct = false
	e.ExtraFields.DeepJSON = true
	e.ExtraFields.Len = false
	e.ExtraFields.Type = false
	m, err := e.ToStringMap(bodyJSON)
	if err != nil {
		return nil, sdk.WrapError(err, "Unable to dump body %s", t.WebHook.RequestBody)
	}
	h.Payload = m
	h.Payload["payload"] = string(t.Kafka.Message)

	return &h, nil
}
