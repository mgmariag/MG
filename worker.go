package main

import (
	"encoding/binary"
	"fastfood/orders"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// worker code for the worker running in the same thread.
func worker(wid int, conn *amqp.Connection) {

	ch, err := conn.Channel()
	if err != nil {
		panic(err)
	}
	defer ch.Close()

	// ORDERS_QUEUE where messages are consumed from
	q, err := ch.QueueDeclare(
		ORDER_QUEUE, // name
		false,       // durable
		false,       // delete when unused
		false,       // exclusive
		false,       // no-wait
		nil,         // arguments
	)
	if err != nil {
		panic(err)
	}

	err = ch.Qos(
		1,     // prefetch count
		0,     // prefetch size
		false, // global
	)
	if err != nil {
		panic(err)
	}

	msgs, err := ch.Consume(
		q.Name, // queue
		"",     // consumer
		false,  // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	if err != nil {
		panic(err)
	}

	var forever chan struct{}

	go func() {
		for d := range msgs {
			oid := binary.BigEndian.Uint32(d.Body)

			OrdersDB.ChangeOrderStatus(oid, orders.ORDER_PREPARING)
			fmt.Printf("[worker %d] Started preparing order %d\n", wid, oid)
			time.Sleep(10 * time.Second)
			fmt.Printf("[worker %d] Order with id %d finished\n", wid, oid)
			OrdersDB.ChangeOrderStatus(oid, orders.ORDER_READY)

			d.Ack(false)
		}
	}()
	<-forever
}
