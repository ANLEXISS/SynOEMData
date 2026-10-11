"""RKNN reference adapter for the Vision inference contract."""

from __future__ import annotations

import logging
from typing import Any, Optional

import numpy as np

from .inference_backend import InferenceRuntimeConfig, ModelExecutionError, ModelMetadata, ModelUnavailableError

try:
    from rknnlite.api import RKNNLite
except Exception as exc:  # Runtime dependency is optional on non-Rockchip hosts.
    RKNNLite = None
    RKNN_IMPORT_ERROR = exc
else:
    RKNN_IMPORT_ERROR = None


log = logging.getLogger("synora.vision.rknn")


def rknn_available() -> bool:
    if RKNNLite is None:
        return False
    runtime = None
    try:
        runtime = RKNNLite()
        return True
    except Exception:
        return False
    finally:
        if runtime is not None:
            try:
                runtime.release()
            except Exception:
                log.warning("failed to release RKNN availability probe", exc_info=True)


def detector_core_masks() -> tuple[int, ...]:
    """Return the established three-core strategy without leaking RKNN types."""
    if RKNNLite is None:
        return ()
    masks = tuple(getattr(RKNNLite, name, None) for name in ("NPU_CORE_0", "NPU_CORE_1", "NPU_CORE_2"))
    if any(mask is None for mask in masks):
        return ()
    return masks


class RKNNInferenceBackend:
    """One initialized RKNN model; preprocessing/decoding remain model-owned."""

    backend = "rknn"

    def __init__(
        self,
        model_path: str,
        metadata: ModelMetadata,
        config: InferenceRuntimeConfig,
        input_data_format: Optional[str] = None,
    ):
        if RKNNLite is None:
            raise ModelUnavailableError(
                "backend_unavailable", model_path, f"RKNNLite unavailable: {RKNN_IMPORT_ERROR}"
            )
        self.model_path = model_path
        self.metadata = metadata
        self.config = config
        self._logged_outputs = False
        self._closed = False
        self._input_data_format = input_data_format or metadata.runtime_data_format
        self._runtime = RKNNLite()
        try:
            result = self._runtime.load_rknn(model_path)
            if result != 0:
                raise ModelUnavailableError("invalid_model", model_path, f"RKNN load failed ret={result}")
            result = self._runtime.init_runtime(core_mask=config.core_mask)
            if result != 0:
                raise ModelUnavailableError("rknn_runtime_error", model_path, f"RKNN init runtime failed ret={result}")
        except Exception:
            self._release_after_init_failure()
            raise
        log.info("RKNN backend ready model=%s target=%s", model_path, metadata.target)

    def _release_after_init_failure(self) -> None:
        runtime, self._runtime = self._runtime, None
        if runtime is not None:
            try:
                runtime.release()
            except Exception:
                log.warning("failed to release partially initialized RKNN model", exc_info=True)

    def infer(self, input_tensor: Any) -> Any:
        if self._closed or self._runtime is None:
            raise ModelExecutionError("backend_closed", self.metadata.model_id, "RKNN backend is closed")
        tensor = np.ascontiguousarray(input_tensor)
        kwargs = {"inputs": [tensor]}
        if self._input_data_format is not None:
            kwargs["data_format"] = [self._input_data_format]
        try:
            outputs = self._runtime.inference(**kwargs)
        except Exception as exc:
            raise ModelExecutionError("inference_failed", self.metadata.model_id, str(exc)) from exc
        if not self._logged_outputs and outputs:
            log.info("RKNN output shapes=%s", [np.asarray(output).shape for output in outputs])
            self._logged_outputs = True
        return outputs

    def close(self) -> None:
        if self._closed:
            return
        self._closed = True
        runtime, self._runtime = self._runtime, None
        if runtime is None:
            return
        try:
            runtime.release()
        except Exception as exc:
            raise ModelExecutionError("cleanup_failed", self.metadata.model_id, str(exc)) from exc
