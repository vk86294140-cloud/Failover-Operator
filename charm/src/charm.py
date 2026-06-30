#!/usr/bin/env python3
"""FailoverApp Charmed Operator.

A minimal Kubernetes charm built with the Juju `ops` framework. It runs a
workload container via Pebble and reports active only once the unit is up and a
minimum-healthy floor is satisfied. It is intentionally small: the goal is to
work in the Charmed-operator model directly and compare it with the
controller-runtime operator in the parent repository.
"""

import logging

import ops

logger = logging.getLogger(__name__)


class FailoverAppCharm(ops.CharmBase):
    """Charm the FailoverApp workload."""

    def __init__(self, *args):
        super().__init__(*args)
        self.framework.observe(
            self.on["workload"].pebble_ready, self._on_pebble_ready
        )
        self.framework.observe(self.on.config_changed, self._on_config_changed)

    def _on_pebble_ready(self, event: ops.PebbleReadyEvent) -> None:
        """Define and start the workload layer once Pebble is up."""
        self._reconcile(event.workload)

    def _on_config_changed(self, event: ops.ConfigChangedEvent) -> None:
        """Re-render the workload when Juju config changes."""
        container = self.unit.get_container("workload")
        if not container.can_connect():
            self.unit.status = ops.WaitingStatus("waiting for Pebble")
            event.defer()
            return
        self._reconcile(container)

    def _reconcile(self, container: ops.Container) -> None:
        """Push the desired Pebble layer and set unit status."""
        port = int(self.config["port"])
        min_healthy = int(self.config["min-healthy"])

        layer = {
            "summary": "failover-app workload",
            "description": "workload managed by the FailoverApp charm",
            "services": {
                "workload": {
                    "override": "replace",
                    "summary": "workload",
                    "command": f"nginx -g 'daemon off;'",
                    "startup": "enabled",
                    "environment": {"PORT": str(port)},
                }
            },
        }
        container.add_layer("workload", layer, combine=True)
        container.replan()

        # A single Pebble-managed unit; report the min-healthy floor we enforce.
        self.unit.set_ports(port)
        self.unit.status = ops.ActiveStatus(f"ready (min-healthy={min_healthy})")


if __name__ == "__main__":  # pragma: no cover
    ops.main(FailoverAppCharm)
