# k8vol
Kubernetes FlexVolume Driver for iguazio Data Platform (v3io fuse) 

This driver allow accessing iguazio data containers (or a sub-path) as a shared volume storage for Kubernetes.

the same volume (data container) can be accessed **simultaneously** by multiple application containers or ([nuclio](https://github.com/nuclio/nuclio)) serverless functions, by multiple remote clients (e.g. via S3 Object API or DynamoDB API), and can be viewed or modified in the iguazio UI (browse container view). 


> Note: iguazio data platform provide unique multi-model capabilities, files and objects can also be viewed as database, document or stream records when using appropriate data APIs. Updates to data are immediately committed ensuring full consistency regardless of the API semantics.   


## Installation

Requierments:  
 - install v3io-fuse

```bash
$ yum install fuse librdmacm
$ rpm -iv v3io-fuse.rpm
```

## Security and Session Authentication 
Access to iguazio data platform must be authenticated, each identity may have different read or write permissions to individual files and directories. The username and password are provided by using `username` and `password`  options.

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
      driver: "v3io/fuse"
      secretRef:   
        name: v3io-fuse-user
      options:
        container: bigdata      # data container name
        username: myuser        # username
        password: mypassword    # you get it, right?
```

