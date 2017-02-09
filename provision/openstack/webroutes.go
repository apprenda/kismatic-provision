package openstack

import (
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
)

type Response struct {
	Status string `json:"status"`
}

func parseBody(r *http.Request) (KetBag, error) {
	//get the post data with credentials and setting
	bag := KetBag{}
	//	var conf Config
	//	var nodeData serverData
	bodyData, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Println("Error reading body", err)
	}

	decoded, decerr := b64.StdEncoding.DecodeString(string(bodyData))
	if decerr != nil {
		log.Println("Something wrong with the post data 64bit encoding", decerr)
		return bag, decerr
	}

	unmarshErr := json.Unmarshal(decoded, &bag)
	if unmarshErr != nil {
		log.Println("Cannot deserialize ket bag", unmarshErr)
		return bag, unmarshErr
	}
	fmt.Println("Bag", bag)
	defer r.Body.Close()
	return bag, nil
}

func ProvisionAndInstall(w http.ResponseWriter, r *http.Request) {
	log.Printf("ProvisionAndInstall called")
	log.Println(r.URL.RawQuery)
	q, _ := url.ParseQuery(r.URL.RawQuery)
	var ip string = ""
	if len(q["ip"]) > 0 {
		ip = q["ip"][0]
	}
	bag, bodyErr := parseBody(r)
	if bodyErr != nil {
		log.Println("Body error", bodyErr)
	}

	//kick off all the requested nodes

	err := provisionKetNodes(bag, ip)
	if err != nil {
		log.Println("Error instantiating worker node", err)
	}
}

func NodeUp(w http.ResponseWriter, r *http.Request) {
	log.Println("Node Up called")
	q := r.URL.Query()
	fmt.Println("received", q)
	nodeType := q["type"][0]
	nodeIP := q["ip"][0]
	nodeName := q["name"][0]
	log.Println("Parsed vals:", nodeType, nodeIP, nodeName)

	bag, bodyErr := parseBody(r)
	if bodyErr != nil {
		log.Println("Body error", bodyErr)
	}

	log.Println("Bag", bag)
	//save the provisioned node
	cacheNode(nodeName, nodeIP, bag)

	w.Header().Set("Content-Type", "application/json")
	resp := Response{Status: "Received node"}
	json.NewEncoder(w).Encode(resp)
	checkIfStartKetInstall(bag)
}
