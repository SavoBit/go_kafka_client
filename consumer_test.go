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
	"testing"
	"github.com/samuel/go-zookeeper/zk"
	"fmt"
	"time"
)

func TestConsumer(t *testing.T) {
	WithZookeeper(t, func(_ *zk.Conn) {
		consumer := NewConsumer(DefaultConsumerConfig())
		AssertNot(t, consumer.zkConn, nil)

		//TODO other
	})
}

func WithZookeeper(t *testing.T, zookeeperWork func (zkConnection *zk.Conn)) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatal(r)
		}
	}()

	testCluster, err := zk.StartTestCluster(1)
	if err != nil {
		t.Fatal(err)
	}

	zkServer := &testCluster.Servers[0]

	conn, _, err := zk.Connect([]string{fmt.Sprintf("127.0.0.1:%d", zkServer.Port)}, time.Second*30000)
	if (err != nil) {
		t.Fatal(err)
	}

	zookeeperWork(conn)

	testCluster.Stop()
}
