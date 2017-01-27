# k8s-v3io
Kubernetes Volume Driver for iguazio Data Containers (v3io fuse) 

this driver allow accessing iguazio data containers as a shared volume storage for Kubernetes 
the same volume (data container) can be accessed by multiple clients including remote S3 users
and be viewed or modified in the iguazio UI (browse container view) 

the driver need to be placed in the Kubelet volume-plugin directory, in default its:

  /usr/libexec/kubernetes/kubelet-plugins/volume/exec/igz~v3io/
  
Need to verify it has execution permissions, and Kubelet is started/restarted after placing the driver 

Requierments:  
 - install jq utility 
 - install v3io-fuse 

##Example POD YAML using the driver:
(note the Authentication feature is still not enabled and should be ignored) 

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: ngvol
spec:
  containers:
  - name: nginx
    image: nginx
    volumeMounts:
    - name: test
      mountPath: /data
    ports:
    - containerPort: 80
  volumes:
  - name: test
    flexVolume:
      driver: "igz/v3io"
      secretRef: # for future use, not supported yet 
        name: mysecret
      options:
        url: "tcp://192.168.1.1"
        container: vol1

apiVersion: v1
kind: Secret
metadata:
  name: mysecret
data:
  username: YWRtaW4=
  password: MWYyZDFlMmU2N2Rm
```

