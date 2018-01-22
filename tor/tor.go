package tor

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"text/template"
)

const configFormat = `
{{ range .HiddenServices }}
HiddenServiceDir /run/tor/{{ .ServiceNamespace}}_{{ .ServiceName }}/
HiddenServicePort {{ .PublicPort }} {{ .ServiceClusterIP }}:{{ .ServicePort }}
{{ end }}
`

var configTemplate = template.Must(template.New("config").Parse(configFormat))

type HiddenService struct {
	ServiceName      string
	ServiceNamespace string
	ServiceClusterIP string
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
		ServicePort:      servicePort,
		PublicPort:       publicPort,
	}
	t.HiddenServices[fmt.Sprintf("%s/%s", namespace, name)] = s
	return &s
}

func (t *TorConfiguration) RemoveService(name string) {
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
	data, err := ioutil.ReadFile(fmt.Sprintf("/run/tor/%s_%s/hostname", s.ServiceNamespace, s.ServiceName))
	if err != nil {
		return "", err
	}
	return string(data), nil
}
