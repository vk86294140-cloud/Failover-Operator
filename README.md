# failover-operator

A small Kubernetes operator, written in Go with [controller-runtime], that
manages a `FailoverApp` custom resource. It's deliberately modest in scope — the
point is to implement the watch–reconcile pattern end to end rather than to ship
a large platform.

Given a `FailoverApp`, the operator:

1. ensures a `Deployment` exists for the workload (image, replicas, port),
2. sets an owner reference so the Deployment is garbage-collected with the CR,
3. reconciles drift on the fields it owns (replica count and image),
4. reports a `Ready` status and an `Available` condition based on whether the
   live Deployment is holding at or above a `minHealthy` replica floor.

The `minHealthy` floor is the idea I cared about: a workload isn't "up" because
the object exists, it's up when enough replicas are actually available. That
mirrors the reliability/failover work I do day to day — the question that
matters is whether the system is holding on a bad day, not whether it was
created.

## The custom resource

```yaml
apiVersion: ops.failover.dev/v1alpha1
kind: FailoverApp
metadata:
  name: demo
spec:
  image: nginx:1.27
  replicas: 3
  minHealthy: 2   # below 2 available pods, status.ready = false
  port: 80
```

```
$ kubectl get failoverapps
NAME   READY   AVAILABLE   DESIRED
demo   true    3           3
```

## How it works

- **Level-triggered reconcile.** `Reconcile` reads the live world, computes the
  desired `Deployment`, makes it so, then writes back what it observed. Running
  it repeatedly for the same object is safe and converges to the same result.
- **Watches.** `SetupWithManager` watches `FailoverApp` and `Owns` the
  `Deployment`, so editing either re-triggers reconciliation. controller-runtime
  serves these from a shared informer cache (list-watch with resource
  versions), not by polling the API server.
- **Status separate from spec.** Readiness and conditions are written through
  the `/status` subresource, so status updates don't fight spec edits.

See [`internal/controller/failoverapp_controller.go`](internal/controller/failoverapp_controller.go)
for the reconcile loop and [`failoverapp_controller_test.go`](internal/controller/failoverapp_controller_test.go)
for the unit tests covering defaulting and drift detection.

## Run it locally

Prerequisites: Go 1.22+, a Kubernetes cluster in `~/.kube/config` (kind or
minikube is fine), and `kubectl`.

```bash
# fetch dependencies
make tidy

# unit tests (no cluster needed)
make test

# install the CRD, then run the controller against your cluster
make install
make run

# in another shell: create a sample and watch it reconcile
make sample
kubectl get failoverapps -w
```

Build the container image:

```bash
make docker-build IMG=failover-operator:dev
```

## Layout

```
api/v1alpha1/        # FailoverApp types + deepcopy + scheme registration
internal/controller/ # reconcile loop and unit tests
cmd/                 # manager entrypoint
config/crd/          # CustomResourceDefinition
config/rbac/         # ClusterRole for the controller
config/samples/      # example FailoverApp
```

## Also in this repo

To work in more than one operator model, the project also includes:

- **envtest integration tests** ([`internal/controller/suite_test.go`](internal/controller/suite_test.go))
  that run the controller against a real API server. Gated by the `integration`
  build tag; run with `make test-integration`.
- **A finalizer** so deletion runs explicit teardown before the object is
  removed.
- **A Juju (Charmed) operator** ([`charm/`](charm/)) built with the `ops`
  framework — the same idea expressed as a charm, to compare the
  controller-runtime and Charmed-operator models.
- **Snap packaging** ([`snap/snapcraft.yaml`](snap/snapcraft.yaml)) for the
  manager binary as a strictly confined snap.

## Roadmap

- metrics dashboard and alerting on `minHealthy` breaches
- `pytest-operator` integration test for the charm
- a relation from the charm to a database charm

[controller-runtime]: https://github.com/kubernetes-sigs/controller-runtime
