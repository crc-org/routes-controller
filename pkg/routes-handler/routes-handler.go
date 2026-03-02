package routes_handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"

	v1 "github.com/openshift/api/route/v1"
	routeclientset "github.com/openshift/client-go/route/clientset/versioned"
	informers "github.com/openshift/client-go/route/informers/externalversions"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/tools/cache"
)

func RoutesHandler(clientset *routeclientset.Clientset) cache.SharedIndexInformer {
	factory := informers.NewSharedInformerFactory(clientset, 5*time.Minute)
	informer := factory.Route().V1().Routes().Informer()
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			route := obj.(*v1.Route)
			log.Infof("added: %s %s", route.GetName(), route.Spec.Host)
			if err := expose(route.Spec.Host); err != nil {
				log.Error(err)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			old := oldObj.(*v1.Route)
			route := newObj.(*v1.Route)
			if old.Spec.Host != route.Spec.Host {
				log.Infof("updated: %s (%s -> %s)", route.GetName(), old.Spec.Host, route.Spec.Host)
				if err := unexpose(old.Spec.Host); err != nil {
					log.Error(err)
				}
				if err := expose(route.Spec.Host); err != nil {
					log.Error(err)
				}
			}
		},
		DeleteFunc: func(obj interface{}) {
			route := obj.(*v1.Route)
			log.Infof("deleted: %s %s", route.GetName(), route.Spec.Host)
			if err := unexpose(route.Spec.Host); err != nil {
				log.Error(err)
			}
		},
	})
	return informer
}

var addHostURLs = []string{
	"http://gateway/hosts/add",
	"http://host:9764/hosts/add",
}

var removeHostURLs = []string{
	"http://gateway/hosts/remove",
	"http://host:9764/hosts/remove",
}

// postJSON sends a POST request with JSON body to url and returns an error on failure or non-2xx status.
func postJSON(url string, body []byte) error {
	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Errorf("%s: server returned %d", url, resp.StatusCode)
	}
	return nil
}

func expose(host string) error {
	bin, err := json.Marshal([]string{host})
	if err != nil {
		return err
	}
	for _, url := range addHostURLs {
		if err := postJSON(url, bin); err != nil {
			return err
		}
	}
	return nil
}

func unexpose(host string) error {
	bin, err := json.Marshal([]string{host})
	if err != nil {
		return err
	}
	for _, url := range removeHostURLs {
		if err := postJSON(url, bin); err != nil {
			return err
		}
	}
	return nil
}
