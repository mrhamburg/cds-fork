package main

import (
  "crypto/tls"
  "strings"
  "time"

  "github.com/Shopify/sarama"
)

//Client to send to kafka
func initKafkaProducer(kafka, user, password string) (sarama.SyncProducer, error) {
	c := sarama.NewConfig()
	c.Net.TLS.Enable = true
	c.Net.SASL.Enable = true
	c.Net.SASL.User = user
	c.Net.SASL.Password = password
	c.ClientID = user
	c.Producer.Return.Successes = true

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
    config.Version = sarama.V1_0_0_0
    config.Producer.Return.Successes = true
  }

	producer, err := sarama.NewSyncProducer(strings.Split(kafka, ","), c)
	if err != nil {
		return nil, err
	}
	return producer, nil
}

//Send data as a byte arrays array to kafka
func sendDataOnKafka(producer sarama.SyncProducer, topic string, data [][]byte) (int, int, error) {
	var partition int32
	var offset int64
	var err error

	for _, m := range data {
		msg := &sarama.ProducerMessage{Topic: topic, Value: sarama.ByteEncoder(m)}
		partition, offset, err = producer.SendMessage(msg)
		if err != nil {
			return 0, 0, err
		}
	}
	return int(partition), int(offset), nil
}
