package tor

import (
	"fmt"
)

type HiddenService struct {
	Name             string
	ServiceName      string
	ServiceNamespace string
	ServicePort      int
	PublicPort       int
}

type TorConfiguration struct {
	HiddenServices []HiddenService
}

func NewTorConfiguration() TorConfiguration {
	return TorConfiguration{
		HiddenServices: []HiddenService{},
	}
}

func (t *TorConfiguration) AddService(name, namespace string, servicePort, publicPort int) {
	s := HiddenService{
		Name:             fmt.Sprint("%s/%s", namespace, name),
		ServiceName:      name,
		ServiceNamespace: namespace,
		ServicePort:      servicePort,
		PublicPort:       publicPort,
	}
	t.HiddenServices = append(t.HiddenServices, s)
}

func (t *TorConfiguration) RemoveService(name string) {
	var s []HiddenService
	for _, service := range t.HiddenServices {
		if service.Name != name {
			s = append(s, service)
		}
	}
}

func (t *TorConfiguration) GetConfiguration() {
}
