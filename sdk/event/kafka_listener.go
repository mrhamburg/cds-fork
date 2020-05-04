package event

import (
  "crypto/tls"
  "encoding/json"
	"fmt"
  "time"

  "github.com/Shopify/sarama"
	"gopkg.in/bsm/sarama-cluster.v2"

	"github.com/ovh/cds/sdk"
)

// ConsumeKafka consume CDS Event from a kafka topic
func ConsumeKafka(addr, topic, group, user, password string, ProcessEventFunc func(sdk.Event) error, ErrorLogFunc func(string, ...interface{})) error {
	var config = sarama.NewConfig()
	config.Net.TLS.Enable = true
	config.Net.SASL.Enable = true
	config.Net.SASL.User = user
	config.Net.SASL.Password = password

  //Check for Azure EventHubs
  if user == "$ConnectionString" {
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
  }

  config.Producer.Return.Successes = true
  config.ClientID = user
  config.Version = sarama.V1_0_0_0

	clusterConfig := cluster.NewConfig()
	clusterConfig.Config = *config
	clusterConfig.Consumer.Return.Errors = true

	var errConsumer error
	consumer, errConsumer := cluster.NewConsumer(
		[]string{addr},
		group,
		[]string{topic},
		clusterConfig)

	if errConsumer != nil {
		return fmt.Errorf("Error creating consumer: %s", errConsumer)
	}

	// Consume errors
	go func() {
		for err := range consumer.Errors() {
			ErrorLogFunc("Error during consumption: %s", err)
		}
	}()

	// Consume message
	for msg := range consumer.Messages() {
		var event sdk.Event
		json.Unmarshal(msg.Value, &event)
		if err := ProcessEventFunc(event); err != nil {
			ErrorLogFunc("Error on ProcessEventFunc:%s", err)
		} else {
			consumer.MarkOffset(msg, "delivered")
		}
	}
	return nil
}
