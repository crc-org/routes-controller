package routeshandler

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
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
	_, err := informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
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
	if err != nil {
		log.Errorf("failed to add event handler: %v", err)
	}
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

func hostsAPIToken() string {
	tok := os.Getenv("CRC_REST_API_TOKEN")
	if tok == "" {
		log.Warn("CRC_REST_API_TOKEN is not set; hosts API requests will be unauthenticated")
	}
	return tok
}

// postJSON sends a POST request with JSON body to url and returns an error on failure or non-2xx status.
func postJSON(url string, body []byte) error {
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if tok := hostsAPIToken(); tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	//nolint:gosec // URL are constants from addHostURLs or removeHostURLs
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("%s: server returned %d", url, resp.StatusCode)
	}
	return nil
}

// postJSONFirstSuccess tries URLs in order and returns nil on the first successful POST.
// It does not call later URLs after success. If every URL fails, returns errors joined from each attempt.
func postJSONFirstSuccess(urls []string, body []byte) error {
	var errs []error
	for _, url := range urls {
		if err := postJSON(url, body); err != nil {
			errs = append(errs, err)
			continue
		}
		return nil
	}
	return errors.Join(errs...)
}

func expose(host string) error {
	bin, err := json.Marshal([]string{host})
	if err != nil {
		return err
	}
	return postJSONFirstSuccess(addHostURLs, bin)
}

func unexpose(host string) error {
	bin, err := json.Marshal([]string{host})
	if err != nil {
		return err
	}
	return postJSONFirstSuccess(removeHostURLs, bin)
}
