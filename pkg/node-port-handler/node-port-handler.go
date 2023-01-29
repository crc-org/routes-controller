package node_port_handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"net/http"
	"time"
)

const vmVirtualIP = "192.168.127.2"

func NodePortHandler(clientset *kubernetes.Clientset) cache.SharedIndexInformer {
	factory := informers.NewSharedInformerFactory(clientset, 5*time.Minute)
	informer := factory.Core().V1().Services().Informer()

	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			service := obj.(*v1.Service)
			log.Infof("Added service '%s' of type '%s'", service.GetName(), service.Spec.Type)
			if service.Spec.Type != v1.ServiceTypeNodePort {
				log.Debugf("Service '%s' is not of type NodePort. Nothing to do.", service.GetName())
				return
			}

			for _, port := range service.Spec.Ports {
				expose(port.NodePort)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldService := oldObj.(*v1.Service)
			newService := newObj.(*v1.Service)

			log.Infof("Updated service '%s'", newService.GetName())
			if oldService.Spec.Type == v1.ServiceTypeNodePort && newService.Spec.Type != v1.ServiceTypeNodePort {
				log.Infof("Type of service '%s' changed ('%s' -> '%s'). Unexposing NodePorts...", newService.GetName(), oldService.Spec.Type, newService.Spec.Type)
				for _, port := range oldService.Spec.Ports {
					unexpose(port.NodePort)
				}
			} else if oldService.Spec.Type != v1.ServiceTypeNodePort && newService.Spec.Type == v1.ServiceTypeNodePort {
				log.Infof("Type of service '%s' changed ('%s' -> '%s'). Exposing NodePorts...", newService.GetName(), oldService.Spec.Type, newService.Spec.Type)
				for _, port := range newService.Spec.Ports {
					expose(port.NodePort)
				}
			} else if oldService.Spec.Type == v1.ServiceTypeNodePort && newService.Spec.Type == v1.ServiceTypeNodePort {
				log.Infof("Type of service '%s' didn't change ('%s' -> '%s'). Making sure that correct ports are exposed...", newService.GetName(), oldService.Spec.Type, newService.Spec.Type)
				for _, port := range oldService.Spec.Ports {
					unexpose(port.NodePort)
				}

				for _, port := range newService.Spec.Ports {
					expose(port.NodePort)
				}
			} else {
				log.Debugf("Neiter old nor new version of service '%s' was of type NodePort. Nothing to do...", newService.GetName())
			}
		},
		DeleteFunc: func(obj interface{}) {
			service := obj.(*v1.Service)
			log.Infof("Deleted service '%s' of type '%s'", service.GetName(), service.Spec.Type)
			if service.Spec.Type != v1.ServiceTypeNodePort {
				log.Debugf("Service '%s' is not of type NodePort. Nothing to do.", service.GetName())
				return
			}

			for _, port := range service.Spec.Ports {
				unexpose(port.NodePort)
			}
		},
	})

	return informer
}

type ExposeRequest struct {
	Local  string
	Remote string
}

type UnexposeRequest struct {
	Local string
}

func expose(nodePort int32) {
	exposeRequest := ExposeRequest{
		Local:  fmt.Sprintf(":%d", nodePort),
		Remote: fmt.Sprintf("%s:%d", vmVirtualIP, nodePort),
	}
	log.Infof("Exposing port (%s -> %s)", exposeRequest.Local, exposeRequest.Remote)

	bin, err := json.Marshal(exposeRequest)
	if err != nil {
		log.Error(err)
	}
	_, err = http.Post("http://host/services/forwarder/expose", "application/json", bytes.NewReader(bin))
	if err != nil {
		log.Error(err)
	}
}

func unexpose(nodePort int32) {
	unexposeRequest := UnexposeRequest{
		Local: fmt.Sprintf(":%d", nodePort),
	}
	log.Infof("Unexposing port '%s'", unexposeRequest.Local)

	bin, err := json.Marshal(unexposeRequest)
	if err != nil {
		log.Error(err)
	}
	_, err = http.Post("http://host/services/forwarder/unexpose", "application/json", bytes.NewReader(bin))
	if err != nil {
		log.Error(err)
	}
}
