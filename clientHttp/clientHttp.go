package clientHttp

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"paje-datastore-mensageria/errors"
	"time"
)

func DoRequest(url string, method string, body []byte) (int, []byte, *errors.RequestError) {
	req, _ := http.NewRequest(method, url, bytes.NewBuffer(body))
	defer req.Body.Close()
	req.Header.Add("Content-Type", "application/json")
	timeout := time.Duration(15 * time.Second)
	client := http.Client{
		Timeout: timeout,
	}
	res, err := client.Do(req)
	if err != nil {
		return 500, nil, errors.NewRequestError(0, err.Error())
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode > 299 {
		b, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return res.StatusCode, nil, errors.NewRequestError(0, err.Error())
		}
		responseBody := string(b)
		return res.StatusCode, nil, errors.NewRequestError(res.StatusCode, responseBody)
	}

	retorno, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return res.StatusCode, retorno, errors.NewRequestError(0, err.Error())
	} else {
		return res.StatusCode, retorno, nil
	}
}
