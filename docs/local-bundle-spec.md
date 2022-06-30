# Local based bundles

## Summary

A Bundle can reference content in a configmap instead of a remote container image or a git repository by using the
`local` source type in the Bundle manifest. This enables one to easily source content locally without external repositories/registries.

For example, for a plain+v0 bundle, the local backing the bundle, or "bundle configmap", must have a certain
data structure in order to produce a valid Bundle that works with the plain provisioner. It should have a map of manifest file contents with its filename as the key
in the `data` of the configmap.
The name and namespace of the configmap must be specified in `spec.source.local.configmap.name` and `spec.source.local.configmap.namespace` respectively.
The configmap can be created with the command
```bash
kubectl create configmap <configmap name> --from-file=<manifests directory>
```

### Example

1. Create the configmap
``` bash
kubectl create configmap my-configmap --from-file=../testdata/bundles/plain-v0/valid/manifests -n rukpak-system
```

2. Create a bundle

```bash
kubectl apply -f -<<EOF
apiVersion: core.rukpak.io/v1alpha1
kind: Bundle
metadata:
  name: combo-v0.1.0
spec:
  source:
    type: local
    local:
      configmap:
        name: my-configmay
        namespace: rukpak-system
  provisionerClassName: core.rukpak.io/plain
EOF
```
