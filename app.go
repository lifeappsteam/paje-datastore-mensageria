package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"paje-datastore-mensageria/clientHttp"
	"strings"
	"sync"

	"cloud.google.com/go/pubsub"
	"github.com/gin-gonic/gin"
	"google.golang.org/api/option"
)

const pathTempFile = "./configPubSub.json"

func subscribe(fila string, callBack func(msg *pubsub.Message)) error {
	err := ioutil.WriteFile(pathTempFile, configPubSub(), 0644)
	if err != nil {
		panic(err)
	}

	ctx := context.Background()
	client, err := pubsub.NewClient(
		ctx,
		"paje-datastore",
		option.WithCredentialsFile(pathTempFile))
	if err != nil {
		panic(err)
	}

	os.Remove(pathTempFile)

	sub := client.Subscription(fila)
	cctx, _ := context.WithCancel(ctx)
	return sub.Receive(cctx, func(ctx context.Context, msg *pubsub.Message) {
		callBack(msg)
	})
}

func main() {
	idProdutoFila := os.Args[1]
	formaAtuacao := os.Args[2]
	var filaAtual = ".pedidos"
	if !strings.Contains(idProdutoFila, ".") {
		idProdutoFila += filaAtual
	}
	switch formaAtuacao {
	case "ativo":
		modoAtivo(idProdutoFila, os.Args[3])
		break
	case "passivo":
		modoPassivo(idProdutoFila, os.Args[3])
		break
	default:
		panic("A forma de atuação deve ser: ativo (envia os pedidos para o ip informado por http post) ou passivo (aguarda requisição para consultar a fila do pubsub)")
	}
}

type memory struct {
	lock  sync.Mutex
	items [][]byte
}

func (m *memory) Add(item []byte) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.items = append(m.items, item)
}

func (m *memory) Items() [][]byte {
	m.lock.Lock()
	defer m.lock.Unlock()
	var retorno [][]byte
	for _, item := range m.items {
		retorno = append(retorno, item)
	}
	return retorno
}

func (m *memory) Clear() {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.items = m.items[:0]
}

func modoPassivo(idProdutoFila string, port string) {
	gin.SetMode(gin.ReleaseMode)

	r := gin.Default()
	cors := func() gin.HandlerFunc {
		return func(c *gin.Context) {
			c.Writer.Header().Add("Access-Control-Allow-Origin", "*")
			c.Next()
		}
	}

	r.Use(cors())

	memory := memory{}

	r.GET("/v1/pedidos", func(c *gin.Context) {
		items := memory.Items()
		var pedidos []interface{}
		for _, item := range items {
			var pedido interface{}
			err := json.Unmarshal(item, &pedido)
			if err != nil {
				log.Println(err)
				c.Error(err)
				return
			}
			pedidos = append(pedidos, pedido)
		}
		memory.Clear()

		if len(pedidos) > 0 {
			c.JSON(200, pedidos)
		} else {
			c.Writer.Header().Set("Content-Type", "application/json; charset=utf-8")
			c.Writer.WriteHeader(200)
			c.Writer.Write([]byte("[]"))
		}
	})

	go r.Run("0.0.0.0:" + port)
	err := subscribe(idProdutoFila, func(msg *pubsub.Message) {
		fmt.Println("JSON do pedido: " + string(msg.Data))
		memory.Add(msg.Data)
		msg.Ack()
	})
	if err != nil {
		panic(err)
	}
}

func modoAtivo(idProdutoFila string, destino string) {
	err := subscribe(idProdutoFila, func(msg *pubsub.Message) {
		fmt.Println("JSON do pedido: " + string(msg.Data))
		_, _, requestError := clientHttp.DoRequest("http://"+destino, "POST", msg.Data)
		if requestError != nil {
			log.Println(requestError.Error())
		} else {
			log.Printf("Pedido transferido com sucesso!")
			msg.Ack()
		}
	})
	panic(err)
}

func configPubSub() []byte {
	retorno := "" // TODO ADICIONAR CHAVE PRIVADA FORNECIDA PELA LIFEAPPS
	return []byte(retorno)
}
