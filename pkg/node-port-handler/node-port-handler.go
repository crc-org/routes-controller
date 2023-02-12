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
				if err := expose(port.NodePort); err != nil {
					log.Error(err)
				}
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldService := oldObj.(*v1.Service)
			newService := newObj.(*v1.Service)

			log.Infof("Service '%s' of type '%s' changed!", newService.GetName(), oldService.Spec.Type)

			if oldService.Spec.Type == v1.ServiceTypeNodePort {
				log.Debugf("Unexposing previous NodePorts...")
				for _, port := range oldService.Spec.Ports {
					if err := unexpose(port.NodePort); err != nil {
						log.Error(err)
					}
				}
			}

			if newService.Spec.Type == v1.ServiceTypeNodePort {
				log.Debugf("Exposing current NodePorts...")
				for _, port := range newService.Spec.Ports {
					if err := expose(port.NodePort); err != nil {
						log.Error(err)
					}
				}
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
				if err := unexpose(port.NodePort); err != nil {
					log.Error(err)
				}
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

func expose(nodePort int32) error {
	exposeRequest := ExposeRequest{
		Local:  fmt.Sprintf(":%d", nodePort),
		Remote: fmt.Sprintf("%s:%d", vmVirtualIP, nodePort),
	}
	log.Infof("Exposing port (%s -> %s)", exposeRequest.Local, exposeRequest.Remote)

	bin, err := json.Marshal(exposeRequest)
	if err != nil {
		return err
	}

	_, err = http.Post("http://host/services/forwarder/expose", "application/json", bytes.NewReader(bin))
	return err
}

func unexpose(nodePort int32) error {
	unexposeRequest := UnexposeRequest{
		Local: fmt.Sprintf(":%d", nodePort),
	}
	log.Infof("Unexposing port '%s'", unexposeRequest.Local)

	bin, err := json.Marshal(unexposeRequest)
	if err != nil {
		return err
	}

	_, err = http.Post("http://host/services/forwarder/unexpose", "application/json", bytes.NewReader(bin))
	return err
}
