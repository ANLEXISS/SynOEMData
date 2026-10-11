"""Hermetic Vision test doubles shared by software-only tests."""

import sys
from pathlib import Path

import numpy as np

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
from core.inference_backend import ModelMetadata


class FakeModelRunner:
    """Deterministic model boundary; never loads RKNN or touches a device."""

    backend = "fake"

    def __init__(self, outputs=None, metadata=None):
        self.outputs = outputs if outputs is not None else [np.ones((1, 512), dtype=np.float32)]
        self.metadata = metadata or ModelMetadata(
            "deterministic-fake", "test", (), "float32", "tensor",
            "deterministic test arrays", backend_requirement="fake", target="test",
        )
        self.calls = 0
        self.closed = False
        self.last_input = None

    def infer(self, input_tensor):
        if self.closed:
            raise RuntimeError("fake model runner is closed")
        self.calls += 1
        self.last_input = np.array(input_tensor, copy=True)
        return [np.array(output, copy=True) for output in self.outputs]

    def close(self):
        self.closed = True
