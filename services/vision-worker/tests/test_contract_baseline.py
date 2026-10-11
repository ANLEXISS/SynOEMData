import json
import os
import sys
import unittest
from unittest import mock

ROOT = os.path.dirname(os.path.dirname(__file__))
REPO = os.path.dirname(os.path.dirname(ROOT))
if ROOT not in sys.path:
    sys.path.insert(0, ROOT)

from core.events import EventBuilder
from worker import VisionWorker, validate_event_contract


FIXTURES = os.path.join(REPO, "testdata", "contract-baseline", "vision")


class VisionContractBaselineTests(unittest.TestCase):
    def fixture(self, name):
        with open(os.path.join(FIXTURES, name), encoding="utf-8") as handle:
            return json.load(handle)

    def test_go_request_fixture_is_valid_worker_input(self):
        request = self.fixture("go-clip-request-v1.json")
        response = VisionWorker(dry_run=True).process_request(request)
        self.assertEqual(response["request_id"], request["request_id"])
        self.assertEqual(len(response["events"]), 1)
        self.assertEqual(response["events"][0]["type"], "vision.unknown")
        self.assertEqual(response["events"][0]["source"], "camera-h1")
        self.assertEqual(response["events"][0]["payload"]["clip_id"], "clip-h1")

    def test_python_event_builder_matches_shared_response_fixture(self):
        context = {
            "camera_id": "camera-h1", "device_id": "camera-h1", "clip_id": "clip-h1",
            "clip_path": "/tmp/synora-h1-fixture.mp4",
            "node_id": "node-h1", "activation_id": "activation-h1",
            "sequence_key": "sequence-h1", "clip_index": 2,
        }
        with mock.patch("core.events.time.time", return_value=1700000000.25):
            event = EventBuilder(context).unknown("camera-h1", "scene-h1", "track-h1", 1)
        self.assertEqual(
            {"request_id": "h1-python-response-001", "events": [event]},
            self.fixture("python-worker-response-v1.json"),
        )
        self.assertEqual(event["type"], "vision.unknown")
        self.assertEqual(event["payload"]["identity"], "unknown")
        self.assertEqual(event["payload"]["persons"], 1)
        validate_event_contract(event)

    def test_unknown_event_names_are_rejected_by_existing_builder(self):
        with self.assertRaisesRegex(ValueError, "unsupported vision event type"):
            EventBuilder()._base("device.command", "camera-h1", "scene-h1")


if __name__ == "__main__":
    unittest.main()
