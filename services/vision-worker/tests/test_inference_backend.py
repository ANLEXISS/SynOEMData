import os
import sys
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

import numpy as np


ROOT = Path(__file__).resolve().parents[3]
sys.path.insert(0, str(ROOT / "services" / "vision-worker"))
from core.inference_backend import (  # noqa: E402
    InferenceBackend,
    InferenceRuntimeConfig,
    ModelExecutionError,
    ModelMetadata,
    ModelUnavailableError,
)
from core.model_runner import create_model_runner, model_status  # noqa: E402
from core.detector_backend import ExistingDetectorBackend  # noqa: E402
from modules.detect.person_detector import PersonDetector  # noqa: E402
from test_helpers import FakeModelRunner  # noqa: E402


class InferenceBackendContractTests(unittest.TestCase):
    def test_metadata_has_explicit_tensor_and_runtime_contract(self):
        from core.model_specs import PERSON_YOLO

        metadata = PERSON_YOLO.as_dict()
        self.assertEqual(metadata["input_shape"], [1, 640, 640, 3])
        self.assertEqual(metadata["input_layout"], "NHWC")
        self.assertEqual(metadata["backend_requirement"], "rknn")
        self.assertEqual(metadata["target"], "rk3588")

    def test_fake_runner_implements_backend_contract(self):
        self.assertIsInstance(FakeModelRunner(), InferenceBackend)

    def test_unsupported_backend_fails_closed_without_installing_runtime(self):
        contract = ModelMetadata("unit", "test", (), "float32", "NHWC", "test output")
        config = InferenceRuntimeConfig(requested_backend="onnx")
        with self.assertRaises(ModelUnavailableError) as raised:
            create_model_runner("unused.rknn", metadata=contract, runtime_config=config)
        self.assertEqual(raised.exception.code, "backend_unavailable")

    def test_model_status_separates_support_configuration_and_availability(self):
        from core.model_specs import PERSON_YOLO

        with tempfile.TemporaryDirectory() as directory:
            model = Path(directory) / "yolo.rknn"
            model.write_bytes(b"fake model")
            with patch.dict(os.environ, {"VISION_BACKEND": "auto"}):
                status = model_status(str(model), PERSON_YOLO, loaded=True)
        self.assertEqual(status["support_state"], "SUPPORTED")
        self.assertEqual(status["configuration_state"], "CONFIGURED")
        self.assertEqual(status["availability_state"], "AVAILABLE")

    def test_detector_does_not_overstate_unsupported_backend_capability(self):
        with tempfile.TemporaryDirectory() as directory:
            model = Path(directory) / "yolo.rknn"
            model.write_bytes(b"test fixture")
            with patch.dict(os.environ, {"SYNORA_YOLO_MODEL_PATH": str(model), "VISION_BACKEND": "onnx"}):
                detector = PersonDetector(runner_factory=lambda *args, **kwargs: FakeModelRunner())
            capability = detector.capability()
            self.assertEqual(capability["support_state"], "UNAVAILABLE")
            self.assertEqual(capability["availability_state"], "AVAILABLE")
            detector.close()

    def test_fake_backend_runs_existing_detector_and_releases_on_close(self):
        rows = np.zeros((1, 84, 8400), dtype=np.float32)
        rows[0, :4, 0] = (320, 320, 200, 300)
        rows[0, 4, 0] = 0.92
        fake = FakeModelRunner([rows])
        with tempfile.TemporaryDirectory() as directory:
            model = Path(directory) / "yolo.rknn"
            model.write_bytes(b"test fixture")
            with patch.dict(os.environ, {"SYNORA_YOLO_MODEL_PATH": str(model)}):
                detector = PersonDetector(runner_factory=lambda *args, **kwargs: fake)
            backend = ExistingDetectorBackend(detector)
            frame = np.zeros((480, 640, 3), dtype=np.uint8)
            detections = backend.detect(frame, frame_timestamp_ms=123)
            self.assertEqual(len(detections), 1)
            self.assertEqual(detections[0]["class"], "human")
            self.assertEqual(detections[0]["frame_timestamp_ms"], 123)
            self.assertEqual(fake.calls, 1)
            self.assertEqual(fake.last_input.shape, (1, 640, 640, 3))
            self.assertEqual(fake.last_input.dtype, np.float32)
            backend.close()
            backend.close()
            self.assertTrue(fake.closed)

    def test_fake_backend_execution_error_is_contained_by_detector_boundary(self):
        class FailingRunner(FakeModelRunner):
            def infer(self, input_tensor):
                raise ModelExecutionError("inference_failed", "test", "synthetic failure")

        with tempfile.TemporaryDirectory() as directory:
            model = Path(directory) / "yolo.rknn"
            model.write_bytes(b"test fixture")
            with patch.dict(os.environ, {"SYNORA_YOLO_MODEL_PATH": str(model)}):
                detector = PersonDetector(runner_factory=lambda *args, **kwargs: FailingRunner())
            backend = ExistingDetectorBackend(detector)
            self.assertEqual(backend.detect(np.zeros((480, 640, 3), dtype=np.uint8)), [])
            backend.close()


if __name__ == "__main__":
    unittest.main()
