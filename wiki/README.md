# wiki

# K8s token & ca.crt

## api server host
```
export KUBERNETES_SERVICE_HOST=127.0.0.1
export KUBERNETES_SERVICE_PORT=6443
```

## 绑定 namespace token 
token
```

```

ca.crt
```

```


## 声明 K8s clientset

```go
// creates the in-cluster config
config, errConfig := rest.InClusterConfig()
if errConfig != nil {
    panic(errConfig.Error())
}
// creates the clientset
clientset, errClientSet := kubernetes.NewForConfig(config)
if errClientSet != nil {
    panic(errClientSet.Error())
}

```