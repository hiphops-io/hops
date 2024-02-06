# Hiphops Helm Chart

This is a working example of how you might deploy Hiphops using Helm.

## Installation

1. Create the `hiphops` namespace if it doesn't already exist
    ```shell
    kubectl get namespace hiphops || kubectl create namespace hiphops
    ```

2. Run the helm install command
   ```shell
    helm upgrade hiphops ./deploy/helm/hiphops -f ./deploy/helm/hiphops/values.yaml --set=hiphops.key="<secret_key>" --install
   ```
