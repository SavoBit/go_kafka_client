/**
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 * 
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package go_kafka_client

import (
	"time"
	"sync"
	"fmt"
)

type MessageBuffer struct {
	OutputChannel chan []*Message
	Messages      []*Message
	Config *ConsumerConfig
	Timer *time.Timer
	MessageLock   sync.Mutex
	Close         chan bool
	stopSending   bool
	TopicPartition TopicAndPartition
	askNextBatch chan TopicAndPartition
	disconnectChannelsForPartition chan TopicAndPartition
	flush chan bool
}

func NewMessageBuffer(topicPartition TopicAndPartition, outputChannel chan []*Message, config *ConsumerConfig, askNextBatch chan TopicAndPartition, disconnectChannelsForPartition chan TopicAndPartition) *MessageBuffer {
	buffer := &MessageBuffer{
		OutputChannel : outputChannel,
		Messages : make([]*Message, 0),
		Config : config,
		Timer : time.NewTimer(config.FetchBatchTimeout),
		Close : make(chan bool),
		TopicPartition : topicPartition,
		askNextBatch : askNextBatch,
		disconnectChannelsForPartition : disconnectChannelsForPartition,
		flush : make(chan bool),
	}

	go buffer.autoFlush()
	go buffer.flushLoop()

	return buffer
}

func (mb *MessageBuffer) String() string {
	return fmt.Sprintf("%s-MessageBuffer", &mb.TopicPartition)
}

func (mb *MessageBuffer) autoFlush() {
	for {
		select {
		case <-mb.Close: return
		case <-mb.Timer.C: {
			Debug(mb, "Batch accumulation timed out. Flushing...")
			mb.Timer.Reset(mb.Config.FetchBatchTimeout)

			select {
			case mb.flush <- true:
			default:
			}
		}
		}
	}
}

func (mb *MessageBuffer) flushLoop() {
	for _ = range mb.flush {
		if len(mb.Messages) > 0 {
			Debug(mb, "Flushing")
			mb.Timer.Reset(mb.Config.FetchBatchTimeout)
		flushLoop:
			for {
				select {
				case mb.OutputChannel <- mb.Messages: break flushLoop
				case <-time.After(200 * time.Millisecond): if mb.stopSending {
					return
				}
				}
			}
			Debug(mb, "Flushed")
			mb.Messages = make([]*Message, 0)
		}
	}
}

func (mb *MessageBuffer) Stop() {
	Debug(mb, "Stopping message buffer")
	mb.stopSending = true
	mb.Close <- true
	close(mb.flush)
	mb.disconnectChannelsForPartition <- mb.TopicPartition
	Debug(mb, "Stopped message buffer")
}

func (mb *MessageBuffer) AddBatch(data *TopicPartitionData) {
	InLock(&mb.MessageLock, func() {
		fetchResponseBlock := data.Data
		topicPartition := data.TopicPartition
		if topicPartition != mb.TopicPartition {
			panic(fmt.Sprintf("%s got batch for wrong topic and partition: %s", mb, topicPartition))
		}
		if fetchResponseBlock != nil {
			for _, message := range fetchResponseBlock.MsgSet.Messages {
				mb.add(&Message {
					Key : message.Msg.Key,
					Value : message.Msg.Value,
					Topic : topicPartition.Topic,
					Partition : topicPartition.Partition,
					Offset : message.Offset,
				})
			}
		}
		mb.askNextBatch <- mb.TopicPartition
	})
}

func (mb *MessageBuffer) add(msg *Message) {
	Debugf(mb, "Added message: %s", msg)
	mb.Messages = append(mb.Messages, msg)
	if len(mb.Messages) == mb.Config.FetchBatchSize {
		Debug(mb, "Batch is ready. Flushing")
		mb.flush <- true
	}
}