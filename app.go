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
	retorno := "{\"type\":\"service_account\",\"project_id\":\"paje-datastore\",\"private_key_id\":\"69ff253c38424ad5e1fe742222f4b41b20b07ebe\",\"private_key\":\"-----BEGIN PRIVATE KEY-----\\nMIIEvgIBADANBgkqhkiG9w0BAQEFAASCBKgwggSkAgEAAoIBAQDRzokk40JGU8Ed\\nbtEE3I4tBep8H8p+243D33g1RP4EUQmbQkY0gw3vFCq4AIq1OmSvp6mXGEnXkiUa\\nUUXu6DVFZSI6E0JMZXZa0dtOPWIIaD4mNREBN6Fp6VfFzCpVC3CGbJZ3Dgnwbdb0\\nVtrgS3+yJ8yVcg1+to3PQm41Dsr//W6hbnv850abRFVlSN39mYav1rq4PA4wWY+t\\nP+pOM+PKhFfV9npN6ABLfcgd2yY+u5XetJfQmi797/4f55VRbVWroQG9enHhIM90\\nO4awNBICY4bRqpFOUuDcFSWW3lC8+JXjcnfYYLdqN0efGwFpTEuubUewN/UIjYvX\\nIvwINkWdAgMBAAECggEAFeV8BzNHpq7VjASVgHAjT3wbU45+0/wGgN+I/GhBqYIq\\noj9Js+/Mi7vbVH5L+6uHOBTjvG7uv/aS9DrwtUUbUC2Eo7KAXBhHwKU/wdviqBV3\\nDQZDStD7QeI2RKCw91S9Du50yqKWs3bHNRN+fuOqRVXlgmBXm7aiOLQKa/OqCIrC\\naABypMmiV+XHPSPDISWVjpvUdyXVcJJmYGT92QTczGD5R33bt/hFblkrRDeZsjGO\\n30yquXI0Q0siY7pDD7rOT3/bUzGEDr+BH5bhh4onSR95xPhGaRZn1wJbaS+lh4a7\\n3+9prHENrlz+uhVtFfUi2ijpZzu7mFzZhFTNSTKUwQKBgQD0hpE2YRis5ThJKlWM\\nsiN+cpgTFzpol3k+axDHBgQUXRv6XpnQ+XFVb3pojT4YQeMc0ZTigV7yLBlF8VpY\\nBc0MHcsTyCu+f5N/Ydiw731U8uNUJ1aJwx095oStVJGpQhkksGTp/pZ0sff2sHU0\\nRVB1eoXfJsBVnsNZCWrkrHdmwQKBgQDbpuX38gzR7VyKoOx6ESFmNH2iavoVRJgC\\nB9Q79MeDjR4x8EAHwi9VNbAEKPOXkCAscODKWWyF1T087SZF1rbrqcUEvaXvxWO8\\nGuHTG5LN98CKh74r0SHlV2nFnLu7tnx3glV/HAUv1Lbb0POzsZXVRlFwtao7be1/\\ncMg6UP/R3QKBgQDajLeXIdtbFJhlFHhYAxOkPancTkN/HftYpXreV2soBDwwX4Mc\\n+wWntbZzYeIg7iqeJFfsxnJlArMoB1qXF8A31x0dtiga4I2lKX/yTGr5lQlHus0m\\n3gPxwmnNPave2tv8JchcN5akADi+/OIUcOtDxNmIJGt9WyQAHWGzts4VQQKBgDCp\\nVfNLPYnYcxMHObyFRQf2gwrTdln121M/1sX9oaHERrc7iYPugjv3a+pQBD9En8wY\\nqcRKcV9o8WspArygJ+AnuU0mkrd+3GyU7Aiv6CMXSyGllvzwFPlRF06/PVwFvqdf\\nSX+ifoetMWGbdhIOOqqILIyywmbuIJqGKuW4giRFAoGBAJJnMvKktFekaf7ZxW3L\\ngZR7mtKVn7ndN3CUZENCumLwhWodeui/GUll/wRpJcnWNoORh/yrGh3Ck18Oim+u\\nmUr5Ow+SpBiQMUUL/30QkOaTTj4S39/feGumEt26Jm7Z4STN+Qlh9ZHAF0HpJrRI\\nyfM0bG8LNeJubiI4ZzKpvcWD\\n-----END PRIVATE KEY-----\\n\",\"client_email\":\"paje-pubsub-client-mensageria@paje-datastore.iam.gserviceaccount.com\",\"client_id\":\"115713454999748890984\",\"auth_uri\":\"https://accounts.google.com/o/oauth2/auth\",\"token_uri\":\"https://oauth2.googleapis.com/token\",\"auth_provider_x509_cert_url\":\"https://www.googleapis.com/oauth2/v1/certs\",\"client_x509_cert_url\":\"https://www.googleapis.com/robot/v1/metadata/x509/paje-pubsub-client-mensageria%40paje-datastore.iam.gserviceaccount.com\"}"
	return []byte(retorno)
}
