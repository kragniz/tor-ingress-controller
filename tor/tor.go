package tor

import (
	"bytes"
	"fmt"
	"text/template"
)

const configFormat = `
{{ range .HiddenServices }}
HiddenServiceDir /run/tor/{{ .ServiceNamespace}}_{{ .ServiceName }}/
HiddenServicePort {{ .PublicPort }} {{ .ServiceName }}.{{ .ServiceNamespace}}:{{ .ServicePort }}
{{ end }}
`

var configTemplate = template.Must(template.New("config").Parse(configFormat))

type HiddenService struct {
	ServiceName      string
	ServiceNamespace string
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

func (t *TorConfiguration) AddService(name, serviceName, namespace string, servicePort, publicPort int) {
	s := HiddenService{
		ServiceName:      serviceName,
		ServiceNamespace: namespace,
		ServicePort:      servicePort,
		PublicPort:       publicPort,
	}
	t.HiddenServices[fmt.Sprintf("%s/%s", namespace, name)] = s
}

func (t *TorConfiguration) RemoveService(name string) {
	delete(t.HiddenServices, name)
}

func (t *TorConfiguration) GetConfiguration() string {
	var tmp bytes.Buffer
	configTemplate.Execute(&tmp, t)
	return tmp.String()
}
