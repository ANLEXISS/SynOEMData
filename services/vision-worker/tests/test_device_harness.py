"""Python reader/runner for the shared, bounded H3 scenario fixtures."""
import copy
import datetime
import json
import os
import unittest


REPO = os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname(__file__))))
FIXTURE = os.path.join(REPO, "testdata", "device-harness", "scenarios-v1.json")
SCHEMA = "synora.oem-test-scenario/v1"


def parse_time(value):
    if not value:
        return None
    return datetime.datetime.fromisoformat(value.replace("Z", "+00:00"))


def format_time(value):
    rendered = value.isoformat(timespec="microseconds")
    if "." in rendered:
        head, tail = rendered.split(".", 1)
        fraction, zone = tail.split("+", 1)
        rendered = head + "." + fraction.rstrip("0") + "+" + zone if fraction.rstrip("0") else head + "+" + zone
    return rendered.replace("+00:00", "Z")


def run_scenario(scenario):
    devices = {item["id"]: copy.deepcopy(item) for item in scenario["devices"]}
    requests = {key: [] for key in devices}
    observations = {key: [] for key in devices}
    clock = parse_time(scenario["logical_clock"])
    trace = []

    for step in scenario["steps"]:
        device = devices[step["device_id"]]
        did = step["device_id"]
        action = step["action"]
        fault = step.get("fault", "")
        status = ""

        if action == "operate":
            requests[did].append({"id": step["id"], "operation": step.get("operation", ""), "at": clock})
            if fault == "device_unavailable":
                device["available"] = False
                status = "unavailable"
            elif fault == "device_recovery":
                device["available"], device["connected"] = True, True
                status = "recovered"
            elif fault == "restart":
                device = copy.deepcopy(next(item for item in scenario["devices"] if item["id"] == did))
                devices[did] = device
                requests[did], observations[did] = [], []
                status = "restarted"
            elif fault == "connection_interruption":
                device["connected"] = False
                status = "connection_interrupted"
            elif not device.get("connected", True):
                status = "connection_interrupted"
            elif not device["available"]:
                status = "unavailable"
            elif fault == "unsupported_operation" or step.get("capability", "") not in device["capabilities"]:
                status = "unsupported"
            elif step.get("response") == "unsupported":
                status = "unsupported"
            elif step.get("response") == "unavailable":
                status = "unavailable"
            elif step.get("response") == "error":
                status = "adapter_error"
            elif fault == "timeout":
                status = "timeout"
            elif fault == "delayed_response":
                clock += datetime.timedelta(milliseconds=step.get("delay_ms", 0))
                status = "delayed"
            elif fault in ("adapter_error", "store_write_failure"):
                status = "adapter_error"
            elif fault == "outbox_retry":
                status = "retry_scheduled"
            elif fault == "malformed_payload":
                status = "malformed_payload"
            else:
                device.setdefault("state", {}).update(step.get("state_after", {}))
                status = "simulated"
        elif action == "observe":
            if fault == "missing_observation":
                status = "missing"
            elif fault == "malformed_payload" or not isinstance(step.get("payload"), dict):
                status = "malformed_payload"
            else:
                observed = {"id": step["event_id"], "sequence": step.get("sequence", 0), "payload": step["payload"]}
                if fault == "duplicate_observation":
                    observations[did].extend([observed, copy.deepcopy(observed)])
                    status = "duplicate_observation"
                else:
                    observations[did].append(observed)
                    status = fault if fault in ("out_of_order_observation", "stale_observation", "sequence_gap") else "observed"
        elif action == "recover":
            device["available"], device["connected"] = True, True
            status = "recovered"
        elif action == "restart":
            devices[did] = copy.deepcopy(next(item for item in scenario["devices"] if item["id"] == did))
            requests[did], observations[did] = [], []
            device = devices[did]
            status = "restarted"
        trace.append(f"{step['id']}:{status}")

    # Current scenario schema intentionally uses one device per executable
    # case. Keep arrays in the core model so multi-device fixtures can be added.
    device = next(iter(devices.values()))
    observation_ids = [item["id"] for values in observations.values() for item in values]
    return {
        "trace": trace,
        "observation_ids": observation_ids,
        "device_state": device.get("state", {}),
        "device_available": device["available"],
        "device_connected": device.get("connected", True),
        "physical_action_executed": False,
        "cleanup_complete": True,
        "final_clock": format_time(clock),
    }


class DeviceHarnessFixtureTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        with open(FIXTURE, encoding="utf-8") as handle:
            cls.fixture = json.load(handle)

    def test_all_scenarios_match_expected_and_repeat(self):
        self.assertEqual(self.fixture["schema_version"], SCHEMA)
        scenarios = self.fixture["scenarios"]
        self.assertEqual(len(scenarios), 10)
        for scenario in scenarios:
            with self.subTest(scenario=scenario["scenario_id"]):
                self.assertEqual(scenario["schema_version"], SCHEMA)
                first = run_scenario(scenario)
                second = run_scenario(scenario)
                self.assertEqual(first, second)
                expected = dict(scenario["expected"])
                expected["final_clock"] = format_time(parse_time(expected["final_clock"]))
                self.assertEqual(first, expected)

    def test_fixed_logical_clock_and_test_only_status(self):
        for scenario in self.fixture["scenarios"]:
            result = run_scenario(scenario)
            self.assertFalse(result["physical_action_executed"])
            self.assertEqual(result["final_clock"], scenario["expected"]["final_clock"])


if __name__ == "__main__":
    unittest.main()
