package main

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/gotoolkits/esCluster/common"
	log "github.com/sirupsen/logrus"
)

const (
	API_GET_KV  = "/v1/kv"
	API_SCHEME  = "http"
	GET_KV_PATH = "/paas/elasticsearch/discovery.zen.ping.unicast.hosts?raw"
	PUT_KV_PATH = "/paas/elasticsearch/discovery.zen.ping.unicast.hosts"
	//DEFAULT_ES_WORK_PATH = "/opt/elk/elasticsearch-2.4.5"
)

func main() {

	var cluster string

	//获取本地ip
	hostIP, err := common.IntranetIP()
	esWorkPath := os.Getenv("ES_WORK_PATH")

	if nil != err {
		log.Errorln(err)
	} else {
		//加入集群
		cluster = joinCluster(hostIP[0])
	}

	log.Infoln("Join the ES Cluster:", cluster)

	//添加主机到Consul集群配置
	err = setKV(cluster)

	if nil != err {
		log.Errorln(err)
	}

	//设置本地配置文件
	err = setConf(cluster, esWorkPath)

	if nil == err {
		status := []byte("ok")
		err = ioutil.WriteFile("/tmp/CONFIG_INIT", status, 0644)

		if err != nil {
			log.Errorln("Set Env CONFIG_INIT file", err)
		}
	}

}

//获取consul KV值
func getKV(url string) []byte {
	resp, err := http.Get(url)
	if nil != err {
		log.Errorln(err)
		return nil
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	return body
}

//集群配置写入Consul KV,替换原有配置
func setKV(kv string) error {
	consulAddr := os.Getenv("CONSUL_ADDR")
	if !strings.Contains(consulAddr, ":") {
		consulAddr = consulAddr + ":" + "8500"
	}

	url := API_SCHEME + "://" + consulAddr + API_GET_KV + PUT_KV_PATH

	client := http.Client{}
	req, _ := http.NewRequest("PUT", url, strings.NewReader(kv))

	resp, err := client.Do(req)
	if nil != err {
		log.Errorln(err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Errorln(err)
		return err
	}
	return nil

}

//修改elasticsearch yml 配置文件
func setConf(hosts, path string) error {

	repl := "discovery.zen.ping.unicast.hosts: " + hosts

	yml, err := ioutil.ReadFile(path + "/config/" + "/elasticsearch.yml")
	if nil != err {
		log.Errorln("Read File:", err)
		return err
	}

	bufStr := bytes.NewBuffer(yml).String()
	re, err := regexp.Compile(`discovery\.zen\.ping\.unicast\.hosts:.*`)
	if nil != err {
		log.Errorln("regexp compile:", err)
		return err
	}

	ymlStr := re.ReplaceAllString(bufStr, repl)
	//fmt.Println(ymlStr)
	yml = bytes.NewBufferString(ymlStr).Bytes()

	err = ioutil.WriteFile(path+"/config/"+"elasticsearch.yml", yml, 0644)

	if nil != err {
		log.Errorln("Write File:", err)
		return err
	}

	return nil
}

//主机加入集群(获取consul kv值并组合成集群配置)
func joinCluster(host string) string {

	var clusterHosts string
	consulAddr := os.Getenv("CONSUL_ADDR")
	if !strings.Contains(consulAddr, ":") {
		consulAddr = consulAddr + ":" + "8500"
	}

	url := API_SCHEME + "://" + consulAddr + API_GET_KV + GET_KV_PATH
	ret := getKV(url)

	if strings.Contains(string(ret), host) {
		return string(ret)
	}

	if !strings.Contains(string(ret), "NULL") {
		Hosts := string(ret[0 : len(ret)-1])
		clusterHosts = Hosts + "," + "\"" + host + "\"" + "]"
	}

	if !strings.Contains(string(ret), "null") {
		Hosts := string(ret[0 : len(ret)-1])
		clusterHosts = Hosts + "," + "\"" + host + "\"" + "]"
	}

	return clusterHosts
}
