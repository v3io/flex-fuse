# k8vol
Kubernetes FlexVolume Driver for iguazio Data Platform (v3io fuse) 

This driver allow accessing iguazio data containers (or a sub-path) as a shared volume storage for Kubernetes.

the same volume (data container) can be accessed **simultaneously** by multiple application containers or ([nuclio](https://github.com/nuclio/nuclio)) serverless functions, by multiple remote clients (e.g. via S3 Object API or DynamoDB API), and can be viewed or modified in the iguazio UI (browse container view). 


> Note: iguazio data platform provide unique multi-model capabilities, files and objects can also be viewed as database, document or stream records when using appropriate data APIs. Updates to data are immediately committed ensuring full consistency regardless of the API semantics.   


## Installation

The driver (v3vol.py) need to be placed in the Kubelet volume-plugin directory, by default its:

  `/usr/libexec/kubernetes/kubelet-plugins/volume/exec/igz~v3io/`
  
Need to verify it has execution permissions, and Kubelet is started/restarted after placing the driver.

The address to the iguazio data platform(s) should be set in the `/etc/v3io/v3io.conf` file or using `./v3vol.py config  <v3io IP address>`

Requierments:  
 - install v3io-fuse 

## Security and Session Authentication 
Access to iguazio data platform must be authenticated, each identity may have different read or write permissions to individual files and directories. The username and password are provided through Kubernetes secrets (see example below), or by using `username` and `password`  options.

The username and password strings are used to form a unique user session per application container.

## Example POD YAML using the driver:

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
      secretRef:   
        name: mysecret
      options:
        container: mydata      # data container name
        cluster: default       # optional, the name of the data cluster in case we use multiple 
        subpath: subdir        # optional, sub directory in the data container
        createnew: false       # optional, if the data container is not found it will create it 

apiVersion: v1
kind: Secret
metadata:
  name: mysecret
data:
  username: YWRtaW4=
  password: MWYyZDFlMmU2N2Rm
```

