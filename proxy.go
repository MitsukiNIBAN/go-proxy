package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"os/exec"
	"path"
	"strconv"
	"strings"
)

type Log struct {
	Access   string `json:"access"`
	Error    string `json:"error"`
	Loglevel string `json:"loglevel"`
}

type Inbounds struct {
	Listen         string           `json:"listen"`
	Port           int              `json:"port"`
	Protocol       string           `json:"protocol"`
	Settings       IbSettings       `json:"settings"`
	StreamSettings IbStreamSettings `json:"streamSettings"`
}

type IbSettings struct {
	Network        string `json:"network"`
	FollowRedirect bool   `json:"followRedirect"`
}

type IbStreamSettings struct {
	Sockopt Sockopt `json:"sockopt"`
}

type Sockopt struct {
	Tproxy string `json:"tproxy"`
}

type Outbounds struct {
	Protocol       string           `json:"protocol"`
	Settings       ObSettings       `json:"settings"`
	StreamSettings ObStreamSettings `json:"streamSettings"`
}

type ObSettings struct {
	Vnext []Vnext `json:"vnext"`
}
type Vnext struct {
	Address string  `json:"address"`
	Port    int     `json:"port"`
	Users   []Users `json:"users"`
}
type Users struct {
	Id       string `json:"id"`
	AlterId  int    `json:"alterId"`
	Security string `json:"security"`
}

type ObStreamSettings struct {
	Network    string     `json:"network"`
	Security   string     `json:"security"`
	WsSettings WsSettings `json:"wsSettings"`
}
type WsSettings struct {
	ConnectionReuse bool   `json:"connectionReuse"`
	Path            string `json:"path"`
}

type SimpleV2ray struct {
	Log       Log         `json:"log"`
	Inbounds  []Inbounds  `json:"inbounds"`
	Outbounds []Outbounds `json:"outbounds"`
}

func startV2ray() error {
	// path, err := ioutil.ReadFile("./" + V2rayTempFile)
	// if err != nil {
	// 	return err
	// }
	// if len(string(path)) <= 0 {
	// 	panic("缺少v2ray配置路径")
	// }
	// cmd := exec.Command("nohup", "v2ray", "--config="+string(path), ">/dev/null", "2>&1", "&")
	cmd := exec.Command("sudo", "systemctl", "start", "v2ray")
	return cmd.Run()
}

func stopV2ray(pid string) error {
	// cmd := exec.Command("sudo", "kill", "-9", pid)
	cmd := exec.Command("sudo", "systemctl", "stop", "v2ray")
	return cmd.Run()
}

func isV2rayRunning() string {
	var outInfo bytes.Buffer
	cmd := exec.Command("sudo", "pidof", "v2ray")
	cmd.Stdout = &outInfo
	err := cmd.Run()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(outInfo.String())
}

func obtainV2rayConfigPath() (int, string) {
	content, err := ioutil.ReadFile(v2rayTempFilePath())
	if err != nil {
		return 500, "获取配置路径失败:" + err.Error()
	}
	return 200, string(content)
}

func saveV2rayConfigPath(path string) (int, string) {
	if len(path) == 0 {
		return 403, "配置保存失败:未填写配置路径"
	} else {
		err := ioutil.WriteFile(v2rayTempFilePath(), []byte(path), 0644)
		if err != nil {
			return 500, "配置保存失败:" + err.Error()
		}
	}
	return 200, ""
}

func obtainV2rayStatus() (int, string) {
	return 200, isV2rayRunning()
}

func controlV2ray(isStartUp bool) (int, string) {
	var msg string
	if isStartUp {
		msg = "进程已启动"
	} else {
		msg = "进程已停止"
	}

	pid := isV2rayRunning()
	if (len(pid) > 0) == isStartUp {
		return 200, msg
	}

	if isStartUp {
		err := startV2ray()
		if err != nil {
			return 500, "启动进程失败:" + err.Error()
		}
	} else {
		err := stopV2ray(pid)
		if err != nil {
			return 500, "停止进程失败:" + err.Error()
		}
	}

	return 200, msg
}

func obtainCurrentProxyName() (int, string) {
	content, err := ioutil.ReadFile(v2rayTempFilePath())
	if err != nil {
		return 500, "获取代理信息失败:" + err.Error()
	}
	config, err := ioutil.ReadFile(string(content))
	if err != nil {
		return 500, "获取代理信息失败:" + err.Error()
	}

	folder, err := ioutil.ReadFile(configSaveFolderTempFilePath())
	if err != nil {
		return 500, "配置获取失败:" + err.Error()
	}

	var current SimpleV2ray
	err = json.Unmarshal(config, &current)
	if err != nil {
		return 500, "获取代理信息失败:" + err.Error()
	}

	fileName := current.Outbounds[0].Settings.Vnext[0].Address + "~" + strconv.Itoa(current.Outbounds[0].Settings.Vnext[0].Port) + ".json"

	detail, err := ioutil.ReadFile(path.Join(string(folder), fileName))
	if err != nil {
		return 500, "获取代理信息失败:" + err.Error()
	}

	var v V2ray
	err = json.Unmarshal(detail, &v)
	if err != nil {
		return 500, "获取代理信息失败:" + err.Error()
	}
	proxyStr := v.Ps + "(" + v.Add + ":" + strconv.Itoa(v.Port) + ")"
	return 200, proxyStr
}

func modifyV2rayConfig(data string) (int, string) {
	pid := isV2rayRunning()
	if len(pid) > 0 {
		return 500, "修改Vray配置失败:代理进程未停止"
	}

	configPath, err := ioutil.ReadFile(v2rayTempFilePath())
	if err != nil {
		return 500, "修改Vray配置失败:" + err.Error()
	}

	var v2ray V2ray
	err = json.Unmarshal([]byte(data), &v2ray)
	if err != nil {
		return 500, "修改Vray配置失败:" + err.Error()
	}

	log := Log{"", "", "info"}

	ibSettings := IbSettings{"tcp,udp", true}
	sockopt := Sockopt{"redirect"}
	ibStreamSettings := IbStreamSettings{sockopt}
	inbounds := []Inbounds{{"0.0.0.0", 60080, "dokodemo-door", ibSettings, ibStreamSettings}}

	users := []Users{{v2ray.Id, v2ray.Aid, "auto"}}
	vnext := []Vnext{{v2ray.Add, v2ray.Port, users}}
	obSettings := ObSettings{vnext}
	wsSettings := WsSettings{true, v2ray.Path}
	obStreamSettings := ObStreamSettings{v2ray.Net, v2ray.Tls, wsSettings}
	outbounds := []Outbounds{{"vmess", obSettings, obStreamSettings}}

	v2rayConfig := SimpleV2ray{log, inbounds, outbounds}

	v2rayConfigStr, err := json.Marshal(v2rayConfig)
	if err != nil {
		return 500, "修改Vray配置失败:" + err.Error()
	}

	err = ioutil.WriteFile(string(configPath), v2rayConfigStr, 0644)
	if err != nil {
		return 500, "修改Vray配置失败:" + err.Error()
	}

	return 200, ""
}
