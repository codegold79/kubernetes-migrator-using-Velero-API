# Kubernetes Workload Migration using Velero API

Velero is a back up and restore tool for Kubernetes clusters. Most of the back and restore functions are exported and can be used as external packages.

Here is an example of being able to back up workloads with the minimal Velero imports.

## Instructions

- Set up two Kubernetes clusters, one to hold the secret, and the other to run backup or restore on
- Before a backup, install an example application on the migration cluster in its own namespace
- Install a secret into the backup cluster (example given in /remote-secret). For more info on how to build a remote cluster secret, see the `Remote Cluster Secret` section in this readme
- Build the binary with `go build -o migrator` and set the chmod to be able to execute.
- Run the binary with flags. For example,

  ```bash
  ./migrator -p /Users/codegold/kubebackups/backup.tar.gz -s remotecluster -n remotesecret -a backup
  ```

- Before a restore, delete the namespace being restored if using the same cluster as backup. Then run the binary using flags. For example,

  ```bash
  ./migrator -p /Users/codegold/kubebackups/backup.tar.gz -s remotecluster -n remotesecret -a restore
  ```

- A file containing the Velero backup files should be generated at `/Users/codegold/kubebackups/backup.tar.gz`

## More TODOs not in code comments

- Include server plugins in order to backup cluster-scoped resources
- Enable using plaintext file instead of secrets to pass in remote cluster credentials

## Remote Cluster Secret

Modify and apply secrets, one for each remote cluster.

- Choose a secret name, for example `remotecluster` and apply it to a namespace
- Fill out the secret data in one of two ways:
    1. The secret must contain the host URL associated with the `host` key, and the service account token for the `sa-token` key.
    2. Or, provide the contents of the remote cluster's `kubeconfig` file in the `kubeconfig` key.
- In the secret, you can optionally provide an HTTPS proxy url to use under the `https_proxy` key.

```yaml
apiVersion: v1
kind: Secret
metadata:
name: <secret name>
namespace: <secret namespace>
type: Opaque
data:
    host: <base64 encoded host URL>
    sa-token: <base64 encoded service account token here>
    kubeconfig: <base64 encoded kubeconfig file contents here>
    https_proxy: <base64 encoded https proxy URL here>
```
