package nats

import (
	"encoding/json"
	"strings"

	"github.com/apcera/nats"
	"github.com/micro/go-micro/broker"
	"github.com/micro/go-micro/cmd"
)

type nbroker struct {
	addrs []string
	conn  *nats.Conn
	opts  broker.Options
}

type subscriber struct {
	s    *nats.Subscription
	opts broker.SubscribeOptions
}

type publication struct {
	t string
	m *broker.Message
}

func init() {
	cmd.DefaultBrokers["nats"] = NewBroker
}

func (n *publication) Topic() string {
	return n.t
}

func (n *publication) Message() *broker.Message {
	return n.m
}

func (n *publication) Ack() error {
	return nil
}

func (n *subscriber) Options() broker.SubscribeOptions {
	return n.opts
}

func (n *subscriber) Topic() string {
	return n.s.Subject
}

func (n *subscriber) Unsubscribe() error {
	return n.s.Unsubscribe()
}

func (n *nbroker) Address() string {
	if len(n.addrs) > 0 {
		return n.addrs[0]
	}
	return ""
}

func (n *nbroker) Connect() error {
	if n.conn != nil {
		return nil
	}

	opts := nats.DefaultOptions
	opts.Servers = n.addrs
	c, err := opts.Connect()
	if err != nil {
		return err
	}
	n.conn = c
	return nil
}

func (n *nbroker) Disconnect() error {
	n.conn.Close()
	return nil
}

func (n *nbroker) Init(opts ...broker.Option) error {
	return nil
}

func (n *nbroker) Options() broker.Options {
	return n.opts
}

func (n *nbroker) Publish(topic string, msg *broker.Message, opts ...broker.PublishOption) error {
	b, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return n.conn.Publish(topic, b)
}

func (n *nbroker) Subscribe(topic string, handler broker.Handler, opts ...broker.SubscribeOption) (broker.Subscriber, error) {
	opt := broker.SubscribeOptions{
		AutoAck: true,
	}

	for _, o := range opts {
		o(&opt)
	}

	fn := func(msg *nats.Msg) {
		var m *broker.Message
		if err := json.Unmarshal(msg.Data, &m); err != nil {
			return
		}
		handler(&publication{m: m, t: topic})
	}

	var sub *nats.Subscription
	var err error

	if len(opt.Queue) > 0 {
		sub, err = n.conn.QueueSubscribe(topic, opt.Queue, fn)
	} else {
		sub, err = n.conn.Subscribe(topic, fn)
	}
	if err != nil {
		return nil, err
	}
	return &subscriber{s: sub, opts: opt}, nil
}

func (n *nbroker) String() string {
	return "nats"
}

func NewBroker(addrs []string, opt ...broker.Option) broker.Broker {
	var cAddrs []string
	for _, addr := range addrs {
		if len(addr) == 0 {
			continue
		}
		if !strings.HasPrefix(addr, "nats://") {
			addr = "nats://" + addr
		}
		cAddrs = append(cAddrs, addr)
	}
	if len(cAddrs) == 0 {
		cAddrs = []string{nats.DefaultURL}
	}
	return &nbroker{
		addrs: cAddrs,
	}
}
