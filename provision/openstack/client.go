package openstack

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	s "strings"
	"time"

	"github.com/Jeffail/gabs"
)

const (
	KeystonePort = "5000"
	ComputePort  = "8774"
	NetworkPort  = "9696"
)

type Config struct {
	Urlauth          string
	Apiverauth       string
	Apivernet        string
	Urlcomp          string
	Urlnet           string
	Apivercomp       string
	InstallscriptURL string
}

// Credentials to be used for accessing the AI
type Credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Openstack auth structure
type Auth struct {
	Body struct {
		Credentials Credentials `json:"passwordCredentials"`
		Tenant      string      `json:"tenantId"`
	} `json:"auth"`
}

// Client for provisioning machines on Openstack
type Client struct {
	Token   string    `json:"token"`
	Expires time.Time `json:"expires"`
}

//Openstack server info structure
type serverData struct {
	Server struct {
		Name            string     `json:"name"`
		User_data       string     `json:"user_data"`
		Key_name        string     `json:"key_name,omitempty"`
		ImageRef        string     `json:"imageRef"`
		FlavorRef       string     `json:"flavorRef"`
		Networks        []network  `json:"networks"`
		Security_groups []secgroup `json:"security_groups"`
	} `json:"server"`
}

type network struct {
	Uuid string `json:"uuid"`
}

type secgroup struct {
	Name string `json:"name"`
}

type floatingIPAction struct {
	AddFloatingIp struct {
		Address string `json:"address"`
	} `json:"addFloatingIp"`
}

func isValidJSON(s string) bool {
	var js map[string]interface{}
	return json.Unmarshal([]byte(s), &js) == nil
}

func (c *Client) login(a Auth, conf Config) (string, error) {
	var fileName = a.Body.Credentials.Username + ".token"
	dat, readerr := ioutil.ReadFile(fileName)
	if readerr == nil && dat != nil && len(dat) > 0 {
		var savedCreds Client
		deserr := json.Unmarshal(dat, &savedCreds)
		if deserr == nil {
			now := time.Now()
			exptime := savedCreds.Expires
			if now.Before(exptime) {
				c.Token = savedCreds.Token
				fmt.Println("Will use existing token", c.Token)
				return c.Token, nil
			}
		}
	}

	url := conf.Urlauth + conf.Apiverauth + "/tokens"

	jsonStr, parseErr := json.Marshal(a)
	if parseErr != nil {
		fmt.Errorf("Something is wrong with auth body", parseErr)
		return "", fmt.Errorf("Something is wrong with auth body: %v", parseErr)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	tr := http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{
		Transport: &tr,
		Timeout:   30 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)

	jsonParsed, err := gabs.ParseJSON(body)

	c.Token = jsonParsed.Path("access.token.id").String()
	c.Token = s.Trim(c.Token, "\"")
	var exp string = jsonParsed.Path("access.token.expires").String()
	exp = s.Trim(exp, "\"")
	layout := "2006-01-02T15:04:05Z"
	c.Expires, _ = time.Parse(layout, exp)
	credsJson, _ := json.Marshal(*c)
	errwrite := ioutil.WriteFile(fileName, credsJson, 0644)
	if errwrite != nil {
		fmt.Errorf("Issues serializing token file %v\n", errwrite)
	}

	return c.Token, err
}

func (c *Client) getAPIClient(auth Auth, conf Config) error {
	if c.Token == "" {
		_, err := c.login(auth, conf)
		if err != nil {
			return fmt.Errorf("Error with credentials provided: %v", err)
		}
	}
	return nil
}

func (c *Client) buildNode(auth Auth, conf Config, nodeData serverData, nodeType string) (string, error) {
	token, err := c.login(auth, conf)
	if err != nil {
		return "", fmt.Errorf("Error with auth: %v", err)
	}

	var url = conf.Urlcomp + conf.Apivercomp + "/" + auth.Body.Tenant + "/servers"

	jsonStr, parseErr := json.Marshal(nodeData)
	if parseErr != nil {
		return "", fmt.Errorf("Error with server data format: %v", parseErr)
	}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
	req.Header.Set("X-Auth-Token", token)
	req.Header.Set("Content-Type", "application/json")

	tr := http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{
		Transport: &tr,
		Timeout:   30 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)

	if isValidJSON(string(body)) == false {
		return "", errors.New("Not a valid JSON response: " + string(body))
	}

	jsonParsed, err := gabs.ParseJSON(body)

	var message, nodeID string

	if jsonParsed.Exists("overLimit", "code") {
		var code = jsonParsed.Path("overLimit.code").String()
		if code == "413" {
			message = "Out of space"
			fmt.Println("message:", message)
		}
	}

	if jsonParsed.Exists("badRequest") {
		message = jsonParsed.Path("badRequest.message").String()
		fmt.Println("message:", message)

	}
	if jsonParsed.Exists("server", "id") {
		nodeID = jsonParsed.Path("server.id").String()
		fmt.Println("nodeID:", nodeID)
	}
	if nodeID == "" {
		message = "Unknown error"
	}
	if message != "" {
		return "", errors.New(message)
	}

	return nodeID, nil

}

func (c *Client) downloadInitScript(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)

	return string(body), nil

}

func (c *Client) parseObj(body []byte, objNode string, idfield string, namefield string) (map[string]string, error) {
	var objMap map[string]string = make(map[string]string)
	jsonParsed, err := gabs.ParseJSON(body)
	if err != nil {
		return nil, err
	}
	if jsonParsed.Exists(objNode) {
		ids := s.TrimLeft(jsonParsed.Path(fmt.Sprintf("%s.%s", objNode, idfield)).String(), "[")
		ids = s.TrimRight(ids, "]")
		idarray := s.Split(ids, ",")
		names := s.TrimLeft(jsonParsed.Path(fmt.Sprintf("%s.%s", objNode, namefield)).String(), "[")
		names = s.TrimRight(names, "]")
		namearray := s.Split(names, ",")
		count := len(idarray)
		for i := 0; i < count; i++ {
			objMap[idarray[i]] = namearray[i]
		}
	}
	return objMap, nil
}

func (c *Client) listImages(auth Auth, conf Config) (map[string]string, error) {
	objType := "images"
	url := conf.Urlcomp + conf.Apivercomp + "/" + auth.Body.Tenant + "/" + objType
	body, err := c.listObjects(auth, conf, url, objType)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Cannot load images. %v", err))
	}
	objMap, errParse := c.parseObj(body, "images", "id", "name")
	if errParse != nil {
		return nil, errParse
	}

	return objMap, nil
}

func (c *Client) listFlavors(auth Auth, conf Config) (map[string]string, error) {
	objType := "flavors"
	url := conf.Urlcomp + conf.Apivercomp + "/" + auth.Body.Tenant + "/" + objType
	body, err := c.listObjects(auth, conf, url, objType)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Cannot load flavors. %v", err))
	}

	objMap, errParse := c.parseObj(body, "flavors", "id", "name")
	if errParse != nil {
		return nil, errParse
	}
	return objMap, nil
}

func (c *Client) listNetworks(auth Auth, conf Config) (map[string]string, error) {
	objType := "networks"
	url := conf.Urlnet + conf.Apivernet + "/" + objType
	body, err := c.listObjects(auth, conf, url, objType)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Cannot load networks. %v", err))
	}

	objMap, errParse := c.parseObj(body, "networks", "id", "name")
	if errParse != nil {
		return nil, errParse
	}

	return objMap, nil
}

func (c *Client) listSecGroups(auth Auth, conf Config) (map[string]string, error) {
	objType := "os-security-groups"
	url := conf.Urlcomp + conf.Apivercomp + "/" + auth.Body.Tenant + "/" + objType
	body, err := c.listObjects(auth, conf, url, objType)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Cannot load security groups. %v", err))
	}
	objMap, errParse := c.parseObj(body, "security_groups", "id", "name")
	if errParse != nil {
		return nil, errParse
	}

	return objMap, nil
}

func (c *Client) listFloatingIPs(auth Auth, conf Config) (map[string]string, error) {
	objType := "os-floating-ips"
	url := conf.Urlcomp + conf.Apivercomp + "/" + auth.Body.Tenant + "/" + objType
	body, err := c.listObjects(auth, conf, url, objType)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Cannot load security groups. %v", err))
	}
	objMap, errParse := c.parseObj(body, "floating_ips", "ip", "ip")
	if errParse != nil {
		return nil, errParse
	}

	return objMap, nil
}

func (c *Client) assignFloatingIP(auth Auth, conf Config, serverID string, ip string) error {
	token, err := c.login(auth, conf)
	if err != nil {
		return fmt.Errorf("Error with auth: %v", err)
	}

	url := conf.Urlcomp + conf.Apivercomp + "/" + auth.Body.Tenant + "/servers/" + serverID + "/action"

	var actionObj floatingIPAction
	actionObj.AddFloatingIp.Address = ip

	jsonStr, parseErr := json.Marshal(actionObj)
	if parseErr != nil {
		log.Println("Something is wrong with action body", parseErr)
		return fmt.Errorf("Something is wrong with auth body: %v", parseErr)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
	req.Header.Set("X-Auth-Token", token)
	req.Header.Set("Content-Type", "application/json")

	tr := http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{
		Transport: &tr,
		Timeout:   30 * time.Second,
	}
	resp, err := client.Do(req)
	body, _ := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	log.Println("Assign IP body", (body))
	defer resp.Body.Close()

	return err
}

func (c *Client) listObjects(auth Auth, conf Config, url string, objType string) ([]byte, error) {
	token, err := c.login(auth, conf)
	if err != nil {
		return nil, fmt.Errorf("Error with auth: %v", err)
	}

	req, err := http.NewRequest("GET", url, nil)
	req.Header.Set("X-Auth-Token", token)

	tr := http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{
		Transport: &tr,
		Timeout:   30 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	fmt.Println("response Status:", resp.Status)

	body, _ := ioutil.ReadAll(resp.Body)

	if isValidJSON(string(body)) == false {
		return nil, errors.New("Not a valid JSON response: " + string(body))
	}
	return body, nil
}
