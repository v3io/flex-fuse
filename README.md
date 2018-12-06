# Flex Fuse
Kubernetes FlexVolume Driver for iguazio Data Platform (v3io fuse) 

This driver allow accessing iguazio data containers (or a sub-path) as a shared volume storage for Kubernetes.

the same volume (data container) can be accessed **simultaneously** by multiple application containers or ([nuclio](https://github.com/nuclio/nuclio)) serverless functions, by multiple remote clients (e.g. via S3 Object API or DynamoDB API), and can be viewed or modified in the iguazio UI (browse container view). 


> Note: iguazio data platform provide unique multi-model capabilities, files and objects can also be viewed as database, document or stream records when using appropriate data APIs. Updates to data are immediately committed ensuring full consistency regardless of the API semantics.   


## Installation

When Kubernetes loads the plugin, v3io driver will install the required packages and v3io-fuse executable.
The following commands will be executed (check [install.sh](hack/scripts/install.sh) for complete overview):
```bash
$ yum install fuse librdmacm
$ rpm -ivh igz-fuse.rpm
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
    - name: v3io
      mountPath: /v3io
    ports:
    - containerPort: 80
  volumes:
  - name: v3io
    flexVolume:
      driver: "v3io/fuse"
      secretRef:   
        name: v3io-fuse-user
      options:
        container: bigdata      # data container name
        cluster: default        # which cluster to connect to (optional, default to "default")
---
apiVersion: v1
kind: Secret
metadata:
  name: v3io-fuse-user
type: v3io/fuse
data:
  accessKey: YThhNHl6dlBMb2g2UU5JcQo=
```

