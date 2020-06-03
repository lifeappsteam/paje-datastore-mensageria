package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"paje-datastore-mensageria/clientHttp"
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
	formaAtuacao := os.Args[2]
	switch formaAtuacao {
	case "ativo":
		modoAtivo(os.Args[1], os.Args[3])
		break
	case "passivo":
		modoPassivo(os.Args[1], os.Args[3])
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

func modoPassivo(iDProduto string, port string) {
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

	err := subscribe(iDProduto+".pedidos", func(msg *pubsub.Message) {
		fmt.Println("JSON do pedido: " + string(msg.Data))
		memory.Add(msg.Data)
		msg.Ack()
	})
	if err != nil {
		panic(err)
	}
}

func modoAtivo(iDProduto string, destino string) {
	err := subscribe(iDProduto+".pedidos", func(msg *pubsub.Message) {
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
	retorno := "{"
	retorno += "\"type\": \"service_account\","
	retorno += "\"project_id\": \"paje-datastore\","
	retorno += "\"private_key_id\": \"dab7e5f5ef96b846d9989b9b63c9de16680fa40d\","
	retorno += "\"private_key\": \"-----BEGIN PRIVATE KEY-----\\nMIIEvAIBADANBgkqhkiG9w0BAQEFAASCBKYwggSiAgEAAoIBAQC+SWi8DTpL7elK\\nvV+kqoEJ6ERgjnHDgAkX75zyWn8EsbNxQ82vymRcCURzzDnqA3OMYbUC0FPh7DP/\\nJonx7sRD619y5ZcAMYDsyl/KmMAQgQ4HWcczTJpq8Al8Ka2FcVBsEyVMz7isATYE\\n2FACbSPdUzIzvVTVvYKqpqyC9snI0BlEvtEapLQ3XOGGnkx5AZz9rnpRDPajUM/q\\nk7mkz3nfD5JaYpoVfpGjJfM40yBR1SNnli/xNwcbtwVAIpLZhru3kpYd7GuJmNSe\\nA2HKHDkq/v9hokR1zYthK26QzqOQ2jDPlwmIL+n+zyCP9MMVn9aOinaZd3FLO04a\\nJxUDbwZzAgMBAAECggEAPQIkRkanbji1F3Vn+M+B179UTPDeoKOoRrhYRYumNccT\\nlTj79WSakLeX7tiHqPO6VEPvWRuaCVFFyoR8rcizvGL2k0vxAerdPw4TcE1RJvl5\\nmfm62EOzLp4PLHPgYmxWMJBi4SGoP92TiDIiVOTRHuDRs6z8ShscjcIqhULCp1Mi\\nMeLMxjUIIvsNQ6H+vckUuCf3yfZt2+as8f6q+NrNgieMB5uhBGyDFpXQAIeZ7aLc\\nDEB91yb2KYjJGvD8pTVKvtP9UHuDdDAu+YPeNA7xlrETdF0MWenXUhjvKqGGD9TE\\n1v9ZkRu8u7yajix2UCARV07Jea6Tj0FD5P2dl9ZRIQKBgQD8xs+EsoHqXbJcus9I\\n075/ihKNL421aKQRDcHDQVKP/UtXXOmv+SBZo14wUpZKKs9aC5O+OtGTPPSh5Go6\\n0eBfhudMpMRCVmLhH1Neb761Rq505tOPKAeF0k100oz8uLMfUJWpwIFuI/yxkPTK\\naLOb3WKbK+X4eMtbt8iWmL8pbQKBgQDAtpmwZZRoYcpKVM8hqxzHkS6qSv3Rvvsy\\nc5upMx9ERZwYfyhl8YMdyo0/ujxRuhemxODwOv3s62Tuyiy5z/jRlshIM0+Waqyt\\ncX6mICmw2nySBBXNcJd3NOaig7frvOAzavkuENDFKD5/bzKZowyrFJJU3hg03UuB\\nDNKL/u/jXwKBgCNxKMWdVdProUeZNdkrP0mYrXM4WLE349E0UZe0AASKalbsgyOW\\nVa/b1SgHXGU3zWz9tJB2pM31PQO6CB8JMGGUg7feXlpCzIhuIP1bw6ydJXbkqoMn\\nBK8BxrR7lSMWLp9UaDet8zfjOdoXzgrXVV+kUeAZ7pvBLBpHEYv0DNGdAoGAAfGV\\niT7tCUR7OtayJB/KsYSYWOVavAPWGsMpvcIjPZgKJAEcUjLmZKWHWS4yr4xV8run\\nnSSrLPmO0g29973OP6bqrDfARL8csL8lTN2kLgF2Ii7iXWkWTgB9lwQHFdyY0kvw\\n6XMH+AUY5EYl14Dafts9Qpfe3KGiwlFzyi+vEbcCgYBvZnqdHkAf/TFKtJbke8tD\\nlQbJIrNUel4w+IeUYAtTEg9P2zQxo2W+xA4KuMUeWAwozqF7EuL2GjdIoQauqeHY\\nnY//T6ZAMx2c3aEXxOMTpikt40+7dNpJkqTk/hu43IC3Szhb4PR+7CZymSZmMKnX\\nRDhncOTPqYhd01H0uYjqHQ==\\n-----END PRIVATE KEY-----\","
	retorno += "\"client_email\": \"paje-datastore-pubsub-client@paje-datastore.iam.gserviceaccount.com\","
	retorno += "\"client_id\": \"111020010196241245203\","
	retorno += "\"auth_uri\": \"https://accounts.google.com/o/oauth2/auth\","
	retorno += "\"token_uri\": \"https://oauth2.googleapis.com/token\","
	retorno += "\"auth_provider_x509_cert_url\": \"https://www.googleapis.com/oauth2/v1/certs\","
	retorno += "\"client_x509_cert_url\": \"https://www.googleapis.com/robot/v1/metadata/x509/paje-datastore-pubsub-client%40paje-datastore.iam.gserviceaccount.com\""
	retorno += "}"
	return []byte(retorno)
}
