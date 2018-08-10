package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	// Uncomment the following line to load the gcp plugin (only required to authenticate against GKE clusters).
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

var (
	lockName   string
	namespace  string
	clientName string
	kubeconfig *string
)

func setupFlags() {
	hostname, err := os.Hostname()
	if err != nil {
		panic(err)
	}
	flag.StringVar(&lockName, "lockname", "kubelock", "name of the lock to aquire")
	flag.StringVar(&namespace, "namespace", "default", "namespace of the lock to aquire")
	flag.StringVar(&clientName, "clientName", hostname, "name of the lock holder, defaults to hostname")

	if home := homeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()
}

func main() {

	setupFlags()
	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	// Make sure the lock exists
	fmt.Println("Checking that lock exists...")
	cm, err := ensureLockExists(clientset, lockName, namespace)
	if err != nil {
		panic(err)
	}

	for {
		fmt.Println("Trying to get lock...")
		if err := maybeGetLock(clientset, cm); err != nil {
			fmt.Printf("error getting lock: %s\n", err)
			fmt.Println("retrying")
			time.Sleep(5 * time.Second)
			continue
		}
		break
	}

	fmt.Println("Got lock!")
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}

func ensureLockExists(clientset *kubernetes.Clientset, name, namespace string) (*v1.ConfigMap, error) {
	cm, err := clientset.CoreV1().ConfigMaps(namespace).Get(lockName, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		cm, err = clientset.CoreV1().ConfigMaps(namespace).Create(&v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name: lockName,
				Annotations: map[string]string{
					"expiry": "0",
					"holder": "",
				},
			},
		})
	}
	return cm, err
}

func isLockFree(cm *v1.ConfigMap, now time.Time) error {

	expiry := "0"
	if _, ok := cm.Annotations["expiry"]; ok {
		expiry = cm.Annotations["expiry"]
	}

	ts, err := strconv.Atoi(expiry)
	if err != nil {
		return nil
	}
	expiryTime := time.Unix(int64(ts), 0)
	remaining := expiryTime.Sub(now).Seconds()
	if remaining > 0 {
		return fmt.Errorf("%s already holds the lock for %v seconds", cm.Annotations["holder"], remaining)
	}
	return nil
}

func maybeGetLock(clientset *kubernetes.Clientset, cm *v1.ConfigMap) error {
	holder := cm.Annotations["holder"]

	now := time.Now().UTC()
	if err := isLockFree(cm, now); err != nil {
		if holder == clientName {
			return fmt.Errorf("%s already holds the lock. Use refresh to extend", holder)
		}
		return err
	}

	newExpiry := now.Add(60 * time.Second).Unix()
	cm.Annotations["expiry"] = strconv.FormatInt(newExpiry, 10)
	cm.Annotations["holder"] = clientName

	return UpdateLock(clientset, cm)
}

var UpdateLock = func(clientset *kubernetes.Clientset, cm *v1.ConfigMap) error {
	_, err := clientset.CoreV1().ConfigMaps(cm.Namespace).Update(cm)
	return err
}
