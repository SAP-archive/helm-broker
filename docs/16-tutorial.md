---
title: Tutorial
---

If you have installed Helm Broker with Service Catalog, you can [add your addon repositories](./07-create-addons-repo.md) and provision ServiceInstances. You can find more ready-to-use addons in the [`addons`](https://github.com/kyma-project/addons) repository. Follow this tutorial to configure Helm Broker and provision the Redis instance:

1. Configure the Helm Broker to use the addons repository that contains the Redis addon:
```bash
kubectl apply -f hack/examples/sample-addons.yaml
```

After Helm Broker processes the addons configuration, you can see the Redis ClusterServiceClass:
```bash
kubectl get clusterserviceclass
```

2. Provision the Redis instance:
```bash
kubectl apply -f hack/examples/redis-instance.yaml
```

3. Check the status of the Redis instance:
```bash
kubectl get serviceinstance
```

4. Create a binding for the Redis instance:
```bash
kubectl apply -f hack/examples/redis-binding.yaml
```

5. Check the Secret that contains Redis credentials:
```bash
kubectl get secret redis -o yaml
```

Use the following commands to see the decoded values:
```bash
kubectl get secret redis -o=jsonpath="{.data.HOST}" | base64 -D
kubectl get secret redis -o=jsonpath="{.data.PORT}" | base64 -D
kubectl get secret redis -o=jsonpath="{.data.REDIS_PASSWORD}" | base64 -D
```
