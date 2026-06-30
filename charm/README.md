# failover-app (Charmed operator)

A minimal Kubernetes charm built with the Juju [`ops`](https://ops.readthedocs.io)
framework. It runs a workload container through Pebble and exposes a
`min-healthy` floor and `port` as Juju config.

This charm is the Juju-side companion to the controller-runtime operator in the
parent repository. I built both to work in the two operator models directly and
compare them:

- **controller-runtime operator** (`../`): a native Kubernetes controller that
  reconciles a `FailoverApp` CRD against owned Deployments.
- **Charmed operator** (here): the same idea expressed as a Juju charm, where
  lifecycle and configuration are driven through Juju and Pebble.

## Try it locally

Prerequisites: a Kubernetes cluster (MicroK8s is the easiest path on Ubuntu),
Juju, and charmcraft.

```bash
# from this charm/ directory
charmcraft pack

juju add-model failover-demo
juju deploy ./failover-app_*.charm \
  --resource workload-image=nginx:1.27

juju config failover-app min-healthy=2 port=80
juju status --watch 1s
```

## Layout

```
charmcraft.yaml   # charm metadata, container, oci-image resource, config
src/charm.py      # the ops charm: pebble-ready + config-changed handlers
requirements.txt  # ops framework
```

## Notes / next steps

This is an early, deliberately small charm built to learn the Charmed-operator
model. Next steps: an integration test with `pytest-operator`, a relation to a
database charm, and an action to trigger a manual failover.
