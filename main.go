package main

import (
	"flag"
	routeshandler "github.com/code-ready/routes-controller/pkg/routes-handler"
	routeclientset "github.com/openshift/client-go/route/clientset/versioned"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	master     string
	kubeconfig string
	debug      bool
)

func main() {
	// setup args
	flag.BoolVar(&debug, "debug", false, "Print debug info")
	flag.StringVar(&master, "master", "", "master url")
	flag.StringVar(&kubeconfig, "kubeconfig", "", "absolute path to the kubeconfig file")
	flag.Parse()

	// setup logging
	if debug {
		log.SetLevel(log.DebugLevel)
	}

	// build config
	config, err := clientcmd.BuildConfigFromFlags(master, kubeconfig)
	if err != nil {
		log.Fatal(err)
	}

	// setup informer stop channel
	stop := make(chan struct{})
	defer close(stop)

	// run routes handler
	routesClientSet, err := routeclientset.NewForConfig(config)
	if err != nil {
		log.Fatal(err)
	}
	routePortHandler := routeshandler.RoutesHandler(routesClientSet)
	routePortHandler.Run(stop)
}
