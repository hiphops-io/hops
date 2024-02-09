# Hiphops Helm Chart

This is a working example of how you might deploy Hiphops using Helm.

## Deployment

1. Create the `hiphops` namespace if it doesn't already exist
    ```shell
    kubectl get namespace hiphops || kubectl create namespace hiphops
    ```

2. Copy or symlink in your automations to the `hiphops.automationsPath` directory (default `automations`)
   
   eg. To copy all the automations from the Hiphops MacOS application you could run.
   ```shell
   cp -R "$HOME/Library/Application Support/Hiphops/automations/" ./deploy/helm/hiphops/automations/
   ```

3. Run the helm install command
   ```shell
   helm upgrade hiphops ./deploy/helm/hiphops -f ./deploy/helm/hiphops/values.yaml --set=hiphops.key="<secret_key>" --install
   ```
