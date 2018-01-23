/*
Copyright 2018 Louis Taylor <louis@kragniz.eu>.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package tor

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"
	"text/template"
)

const configFormat = `
SocksPort 0

{{ range .HiddenServices }}
HiddenServiceDir {{ .ServiceDir }}
HiddenServicePort {{ .PublicPort }} {{ .ServiceClusterIP }}:{{ .ServicePort }}
{{ end }}
`

var configTemplate = template.Must(template.New("config").Parse(configFormat))

type HiddenService struct {
	ServiceName      string
	ServiceNamespace string
	ServiceClusterIP string
	ServiceDir       string
	ServicePort      int
	PublicPort       int
}

type TorConfiguration struct {
	HiddenServices map[string]HiddenService
}

func NewTorConfiguration() TorConfiguration {
	return TorConfiguration{
		HiddenServices: make(map[string]HiddenService),
	}
}

func (t *TorConfiguration) AddService(name, serviceName, namespace, clusterIP string, servicePort, publicPort int) *HiddenService {
	s := HiddenService{
		ServiceName:      serviceName,
		ServiceNamespace: namespace,
		ServiceClusterIP: clusterIP,
		ServiceDir:       fmt.Sprintf("/run/tor/%s_%s_%s_%d/", namespace, name, serviceName, servicePort),
		ServicePort:      servicePort,
		PublicPort:       publicPort,
	}
	t.HiddenServices[fmt.Sprintf("%s/%s", namespace, name)] = s
	return &s
}

func (t *TorConfiguration) RemoveService(name string) {
	err := os.RemoveAll(t.HiddenServices[name].ServiceDir)
	if err != nil {
		fmt.Printf("error removing dir: %v", err)
	}
	delete(t.HiddenServices, name)
}

func (t *TorConfiguration) GetConfiguration() string {
	var tmp bytes.Buffer
	configTemplate.Execute(&tmp, t)
	return tmp.String()
}

func (t *TorConfiguration) SaveConfiguration() {
	file, err := os.Create("/run/tor/torfile")
	if err != nil {
		log.Fatal("Cannot create file", err)
	}
	defer file.Close()

	configTemplate.Execute(file, t)
}

func (s *HiddenService) FindHostname() (string, error) {
	data, err := ioutil.ReadFile(path.Join(s.ServiceDir, "/hostname"))
	if err != nil {
		return "", err
	}
	return strings.Trim(string(data), "\n"), nil
}
