package main

import (
	"context"
	"fmt"

	"cloud.google.com/go/pubsub"
	"google.golang.org/api/option"
)

const iDProduto = "FornecidoPelaLifeApps"

func subscribe(fila string, callBack func(msg *pubsub.Message, cancel context.CancelFunc)) error {
	ctx := context.Background()
	client, err := pubsub.NewClient(
		ctx,
		"paje-datastore",
		option.WithCredentialsFile("./../../configPubSub.json"))
	if err != nil {
		panic(err)
	}
	sub := client.Subscription(fila)
	cctx, cancel := context.WithCancel(ctx)
	return sub.Receive(cctx, func(ctx context.Context, msg *pubsub.Message) {
		callBack(msg, cancel)
	})
}
func main() {
	err := subscribe(iDProduto+".pedidos", func(msg *pubsub.Message, cancel context.CancelFunc) {
		defer msg.Ack()
		fmt.Println("JSON do pedido: " + string(msg.Data))
	})
	if err != nil {
		panic(err)
	}
}
