"""Static backend selection and compatibility factory for Vision models.

This module describes model/runtime needs and delegates concrete execution to
an adapter. It intentionally has no import of RKNN or another runtime.
"""

from __future__ import annotations

import logging
import os
from pathlib import Path
from typing import Optional

from .inference_backend import (
    InferenceBackend,
    InferenceRuntimeConfig,
    ModelMetadata,
    ModelRunner,
    ModelUnavailableError,
)


log = logging.getLogger("synora.vision.model_runner")


def selected_backend(requested: Optional[str] = None, metadata: Optional[ModelMetadata] = None) -> str:
    name = (requested or os.getenv("VISION_BACKEND", "auto")).strip().lower()
    if name == "auto":
        return (metadata.backend_requirement if metadata is not None else "rknn").lower()
    return name


def _rknn_adapter():
    # Static built-in adapter selection; no discovery or dynamic plugin loading.
    from . import rknn_backend
    return rknn_backend


def rknn_available() -> bool:
    return _rknn_adapter().rknn_available()


def detector_core_masks() -> tuple[int, ...]:
    """Hardware-specific core selection is delegated to the selected adapter."""
    if selected_backend() != "rknn":
        return ()
    return _rknn_adapter().detector_core_masks()


def model_status(
    model_path: str,
    metadata: Optional[ModelMetadata] = None,
    *,
    loaded: bool = False,
) -> dict[str, object]:
    """Return legacy status fields plus separate support/config/load states."""
    metadata = metadata or ModelMetadata(
        model_id=Path(model_path).stem or "unidentified-model",
        task="model-defined",
        input_shape=(),
        input_dtype="model-defined",
        input_layout="model-defined",
        output_representation="model-defined",
    )
    backend = selected_backend(metadata=metadata)
    if not model_path:
        legacy = {"status": "missing", "path": "", "error": "model path is empty"}
    elif not Path(model_path).is_file():
        legacy = {"status": "missing", "path": model_path, "error": "model file is missing"}
    elif metadata.backend_requirement == "rknn" and not model_path.lower().endswith(".rknn"):
        legacy = {"status": "invalid", "path": model_path, "error": "RKNN model extension required"}
    elif backend != metadata.backend_requirement:
        legacy = {"status": "unavailable", "path": model_path, "error": f"backend {backend} does not support {metadata.model_id}"}
    elif backend == "rknn" and not rknn_available():
        adapter = _rknn_adapter()
        legacy = {"status": "unavailable", "path": model_path, "error": f"RKNNLite unavailable: {adapter.RKNN_IMPORT_ERROR}"}
    else:
        legacy = {"status": "present", "path": model_path}

    supported = metadata.backend_requirement == backend and backend == "rknn"
    configured = bool(model_path) and Path(model_path).is_file() and legacy["status"] not in {"invalid"}
    available = bool(loaded)
    legacy.update({
        "model_id": metadata.model_id,
        "task": metadata.task,
        "backend": backend,
        "target": metadata.target,
        "support_state": "SUPPORTED" if supported else "UNAVAILABLE",
        "configuration_state": "CONFIGURED" if configured else "UNAVAILABLE",
        "availability_state": "AVAILABLE" if available else "UNAVAILABLE",
        "metadata": metadata.as_dict(),
    })
    return legacy


def create_model_runner(
    model_path: str,
    core_mask: Optional[int] = None,
    input_data_format: Optional[str] = None,
    *,
    metadata: Optional[ModelMetadata] = None,
    runtime_config: Optional[InferenceRuntimeConfig] = None,
) -> InferenceBackend:
    """Initialize the configured static backend for one typed model contract.

    The old positional arguments are preserved for source compatibility. New
    model stages pass explicit metadata; default backend selection remains
    RKNN for existing installations.
    """
    if metadata is None:
        metadata = ModelMetadata(
            model_id=Path(model_path).stem or "unidentified-model",
            task="model-defined",
            input_shape=(),
            input_dtype="model-defined",
            input_layout="model-defined",
            output_representation="model-defined",
            runtime_data_format=input_data_format,
        )
    if runtime_config is None:
        requested = os.getenv("VISION_BACKEND", "auto").strip().lower()
        target = os.getenv("SYNORA_VISION_TARGET", "auto").strip().lower()
        runtime_config = InferenceRuntimeConfig(requested, target, core_mask)
    backend = selected_backend(runtime_config.requested_backend, metadata)
    if backend != metadata.backend_requirement or backend != "rknn":
        raise ModelUnavailableError(
            "backend_unavailable", model_path,
            f"backend {backend} is unavailable for model {metadata.model_id}; supported backend is {metadata.backend_requirement}",
        )
    if metadata.backend_requirement == "rknn" and not model_path.lower().endswith(".rknn"):
        raise ModelUnavailableError("invalid_model", model_path, f"RKNN model required: {model_path}")
    if not os.path.isfile(model_path):
        raise ModelUnavailableError("missing_file", model_path, f"RKNN model file missing: {model_path}")
    return _rknn_adapter().RKNNInferenceBackend(
        model_path,
        metadata,
        runtime_config,
        input_data_format=input_data_format,
    )


# Public protocol aliases retained for callers that imported these names from
# ``core.model_runner`` before the backend boundary was made explicit.
__all__ = [
    "InferenceBackend", "InferenceRuntimeConfig", "ModelMetadata", "ModelRunner",
    "ModelUnavailableError", "create_model_runner", "detector_core_masks",
    "model_status", "rknn_available", "selected_backend",
]

