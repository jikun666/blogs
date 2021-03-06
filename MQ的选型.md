

主要比对 Kafka 和 RocketMQ。


RocketMQ 由阿里开源捐给 apache 的。

RocketMQ 号称延迟和可靠性方面更优。


RocketMQ 支持：

https://github.com/apache/rocketmq/blob/master/docs/cn/features.md

1. 推动式消费（Push Consumer）。该模式下Broker收到数据后会主动推送给消费端，该消费模式一般实时性较高。


2. 消息过滤。可以根据Tag进行消息过滤，也支持自定义属性过滤。消息过滤目前是在Broker端实现的，优点是减少了对于Consumer无用消息的网络传输，缺点是增加了Broker的负担、而且实现相对复杂。

3. 消息重试/消息重投

4. 死信队列。死信队列用于处理无法被正常消费的消息。当一条消息初次消费失败，消息队列会自动进行消息重试；达到最大重试次数后，若消费依然失败，则表明消费者在正常情况下无法正确地消费该消息，此时，消息队列 不会立刻将消息丢弃，而是将其发送到该消费者对应的特殊队列中。