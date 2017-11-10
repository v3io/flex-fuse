# k8vol
Kubernetes Volume Driver for iguazio Data Containers (v3io fuse) 

This driver allow accessing iguazio data containers as a shared volume storage for Kubernetes.

the same volume (data container) can be accessed by multiple remote clients (using S3 Object API), application containers or (nuclio) serverless functions, and be viewed or modified in the iguazio UI (browse container view) 

the driver (v3vol.py) need to be placed in the Kubelet volume-plugin directory, in default its:

  `/usr/libexec/kubernetes/kubelet-plugins/volume/exec/igz~v3io/`
  
Need to verify it has execution permissions, and Kubelet is started/restarted after placing the driver 

Requierments:  
 - install v3io-fuse 

## Example POD YAML using the driver:
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
      secretRef: # not supported in this driver, will be added soon  
        name: mysecret
      options:
        container: mydata      # data container name
        cluster: default       # optional, the name of the data cluster in case we use multiple 
        subpath: subdir        # optional, sub directory in the data container
        dedicate: true         # optional, shared fuse mount vs dedicated mount per container
        createnew: false       # optional, if the data container is not found it will create it 

apiVersion: v1
kind: Secret
metadata:
  name: mysecret
data:
  username: YWRtaW4=
  password: MWYyZDFlMmU2N2Rm
```

