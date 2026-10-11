"""Backend-neutral contracts for local model execution.

Model-specific preprocessing and output decoding stay with their perception
stage. Backends own model initialization, tensor execution, runtime metadata,
and cleanup.
"""

from __future__ import annotations

from dataclasses import asdict, dataclass
from typing import Any, Optional, Protocol, runtime_checkable


@dataclass(frozen=True)
class ModelMetadata:
    model_id: str
    task: str
    input_shape: tuple[int, ...]
    input_dtype: str
    input_layout: str
    output_representation: str
    backend_requirement: str = "rknn"
    target: str = "rk3588"
    runtime_data_format: Optional[str] = None

    def as_dict(self) -> dict[str, Any]:
        value = asdict(self)
        value["input_shape"] = list(self.input_shape)
        return value


@dataclass(frozen=True)
class InferenceRuntimeConfig:
    requested_backend: str = "auto"
    target: str = "auto"
    core_mask: Optional[int] = None


@runtime_checkable
class InferenceBackend(Protocol):
    """One initialized model with a single tensor-in/tensor-out execution API."""

    backend: str
    metadata: ModelMetadata

    def infer(self, input_tensor: Any) -> Any: ...

    def close(self) -> None: ...


class ModelUnavailableError(RuntimeError):
    """A requested model/backend cannot be initialized on this runtime."""

    def __init__(self, code: str, model_path: str, message: str):
        super().__init__(message)
        self.code = code
        self.model_path = model_path
        self.message = message

    def as_dict(self) -> dict[str, str]:
        return {"status": "unavailable", "code": self.code, "path": self.model_path, "error": self.message}


class ModelExecutionError(RuntimeError):
    """A backend execution or cleanup operation failed with a stable code."""

    def __init__(self, code: str, model_id: str, message: str):
        super().__init__(message)
        self.code = code
        self.model_id = model_id
        self.message = message


class ModelRunner(InferenceBackend, Protocol):
    """Compatibility name for existing model-stage annotations."""

