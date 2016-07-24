package go_kafka_client
import (
        "fmt"
)

func (f *FailedMessage) String() string {
        return fmt.Sprintf("topic: %v, error %v", f.message.Topic, f.err)
}
